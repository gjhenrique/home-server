package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"home-server-public/internal/slopper"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var tableRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,23}$`)
var allowedQuery = map[string]bool{"select": true, "order": true, "limit": true, "offset": true, "on_conflict": true}
var scalarFilter = regexp.MustCompile(`^(eq|neq|gt|gte|lt|lte|like|ilike|is)\.[A-Za-z0-9_@:+ -]{1,256}$`)
var scalarProjection = regexp.MustCompile(`^(\*|[A-Za-z_][A-Za-z0-9_]*(,[A-Za-z_][A-Za-z0-9_]*)*)$`)
var scalarOrder = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.(asc|desc))?(,[A-Za-z_][A-Za-z0-9_]*(\.(asc|desc))?)*$`)

type claims struct {
	Sub   string   `json:"sub"`
	Email string   `json:"email"`
	Roles []string `json:"roles"`
	Exp   int64    `json:"exp"`
}
type resourceManifest struct {
	Applications map[string]struct {
		Tables  []string `json:"tables"`
		Buckets []string `json:"buckets"`
	} `json:"applications"`
}

func secret(k string) []byte { return []byte(os.Getenv(k)) }
func main() {
	if len(secret("SLOPPER_PROXY_SECRET")) < 32 || len(secret("SLOPPER_CSRF_SECRET")) < 32 || len(secret("SLOPPER_JWT_SECRET")) < 32 || !strings.HasPrefix(os.Getenv("SLOPPER_INSFORGE_ANON_KEY"), "anon_") || os.Getenv("SLOPPER_AI_MODEL") == "" {
		panic("required Slopper secrets/model are missing")
	}
	http.HandleFunc("/", serve)
	server := &http.Server{Addr: "0.0.0.0:8080", Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		http.DefaultServeMux.ServeHTTP(w, r)
	}), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	_ = server.ListenAndServe()
}
func fail(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
func serve(w http.ResponseWriter, r *http.Request) {
	if !hmac.Equal([]byte(r.Header.Get("X-Slopper-Proxy")), secret("SLOPPER_PROXY_SECRET")) {
		fail(w, http.StatusForbidden, "proxy authorization required")
		return
	}
	c, exp, ok := session(r)
	if !ok {
		fail(w, http.StatusUnauthorized, "authentication required")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/_slopper")
	apiApp := ""
	if strings.HasPrefix(r.Host, "api.") {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) < 2 {
			fail(w, 404, "application-qualified path required")
			return
		}
		apiApp = parts[0]
		parts = parts[1:]
		if len(parts) > 0 && parts[0] == "_slopper" {
			parts = parts[1:]
		}
		path = "/" + strings.Join(parts, "/")
	}
	switch path {
	case "/identity":
		if r.Method != "GET" {
			fail(w, 405, "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": c.Sub, "email": c.Email, "expires_at": exp, "csrf": csrf(c.Sub, exp)})
		return
	case "/csrf":
		if r.Method != "GET" {
			fail(w, 405, "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": csrf(c.Sub, exp)})
		return
	}
	if strings.HasPrefix(path, "/ai/chat/completion") {
		if r.Method != "POST" {
			fail(w, 405, "method not allowed")
			return
		}
		if !mutation(w, r, c, exp) {
			return
		}
		ai(w, r)
		return
	}
	if !strings.HasPrefix(path, "/db/") && !strings.HasPrefix(path, "/objects/") {
		fail(w, 404, "unsupported route")
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "POST" && r.Method != "PUT" && r.Method != "PATCH" && r.Method != "DELETE" {
		fail(w, 405, "method not allowed")
		return
	}
	if r.Method != "GET" && !mutation(w, r, c, exp) {
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" && strings.HasPrefix(path, "/db/") {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	}
	if strings.HasPrefix(path, "/db/") {
		database(w, r, c, path, apiApp)
		return
	}
	objects(w, r, c, path, apiApp)
}
func session(r *http.Request) (claims, int64, bool) {
	name := os.Getenv("SLOPPER_SESSION_COOKIE")
	if name == "" {
		name = "AUTHP_ACCESS_TOKEN"
	}
	ck, e := r.Cookie(name)
	if e != nil {
		return claims{}, 0, false
	}
	p := strings.Split(ck.Value, ".")
	if len(p) != 3 {
		return claims{}, 0, false
	}
	header, err := base64.RawURLEncoding.DecodeString(p[0])
	var headerClaims struct {
		Alg string `json:"alg"`
	}
	if err != nil || json.Unmarshal(header, &headerClaims) != nil || headerClaims.Alg != "HS256" {
		return claims{}, 0, false
	}
	mac := hmac.New(sha256.New, secret("SLOPPER_JWT_SECRET"))
	mac.Write([]byte(p[0] + "." + p[1]))
	sig, e := base64.RawURLEncoding.DecodeString(p[2])
	if e != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return claims{}, 0, false
	}
	b, e := base64.RawURLEncoding.DecodeString(p[1])
	if e != nil {
		return claims{}, 0, false
	}
	var v claims
	if json.Unmarshal(b, &v) != nil || v.Sub == "" || v.Email == "" || v.Exp <= time.Now().Unix() {
		return claims{}, 0, false
	}
	permitted := false
	for _, role := range v.Roles {
		if role == "authp/user" || role == "authp/admin" {
			permitted = true
		}
	}
	if !permitted {
		return claims{}, 0, false
	}
	return v, v.Exp, true
}
func csrf(sub string, exp int64) string {
	m := hmac.New(sha256.New, secret("SLOPPER_CSRF_SECRET"))
	fmt.Fprintf(m, "%s:%d", sub, exp)
	return hex.EncodeToString(m.Sum(nil))
}
func mutation(w http.ResponseWriter, r *http.Request, c claims, exp int64) bool {
	origin := r.Header.Get("Origin")
	u, e := url.Parse(origin)
	if e != nil || u.Host == "" || !strings.EqualFold(u.Host, r.Host) || u.Scheme != "https" {
		fail(w, 403, "origin rejected")
		return false
	}
	if !hmac.Equal([]byte(r.Header.Get("X-Slopper-CSRF")), []byte(csrf(c.Sub, exp))) {
		fail(w, 403, "CSRF token rejected")
		return false
	}
	return true
}
func database(w http.ResponseWriter, r *http.Request, c claims, path, apiApp string) {
	app := apiApp
	if app == "" {
		app = strings.Split(r.Host, ".")[0]
	}
	if !slopper.ValidApp(app) {
		fail(w, 404, "invalid application")
		return
	}
	table := strings.TrimPrefix(path, "/db/")
	if !slopper.ValidResource(table) {
		fail(w, 400, "invalid table")
		return
	}
	if !registered(app, table, "table") {
		fail(w, 404, "unregistered table")
		return
	}
	for k, vs := range r.URL.Query() {
		if !allowedQuery[k] && !isFilter(k) {
			fail(w, 400, "query parameter not allowed")
			return
		}
		for _, v := range vs {
			if isFilter(k) && !scalarFilter.MatchString(v) {
				fail(w, 400, "scalar filters only")
				return
			}
			if !validQueryValue(k, v) {
				fail(w, 400, "query syntax not allowed")
				return
			}
		}
	}
	q := r.URL.Query()
	if q.Get("select") == "" {
		q.Set("select", "*")
	}
	if !scalarProjection.MatchString(q.Get("select")) {
		fail(w, 400, "scalar projections only")
		return
	}
	proxy(w, r, "/slp_"+app+"__"+table, q)
}
func isFilter(k string) bool { return tableRE.MatchString(k) }
func validQueryValue(k, v string) bool {
	switch k {
	case "select":
		return scalarProjection.MatchString(v)
	case "order":
		return scalarOrder.MatchString(v)
	case "limit", "offset":
		n, err := strconv.Atoi(v)
		return err == nil && n >= 0 && n <= 10000
	case "on_conflict":
		return scalarProjection.MatchString(v) && v != "*"
	default:
		return k != "or" && isFilter(k) && scalarFilter.MatchString(v)
	}
}
func objects(w http.ResponseWriter, r *http.Request, c claims, path, apiApp string) {
	if r.Method != "GET" && r.Method != "HEAD" && r.Method != "PUT" && r.Method != "DELETE" {
		fail(w, 405, "method not allowed")
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(path, "/objects/"), "/"), "/")
	app := apiApp
	if app == "" {
		app = strings.Split(r.Host, ".")[0]
	}
	if !slopper.ValidApp(app) || len(parts) < 1 {
		fail(w, 400, "invalid object route")
		return
	}
	bucket := parts[0]
	if !slopper.ValidResource(bucket) || !registered(app, bucket, "bucket") {
		fail(w, 404, "unregistered bucket")
		return
	}
	key := strings.Join(parts[1:], "/")
	if r.Method != "GET" && r.Method != "HEAD" && key == "" {
		fail(w, 400, "object key required")
		return
	}
	for _, segment := range strings.Split(key, "/") {
		if segment == ".." || segment == "." {
			fail(w, 400, "invalid object key")
			return
		}
	}
	if r.Method == "PUT" {
		r.Body = http.MaxBytesReader(w, r.Body, 100<<20)
	}
	if r.ContentLength > 100<<20 {
		fail(w, 413, "upload too large")
		return
	}
	for queryKey, values := range r.URL.Query() {
		if queryKey != "prefix" && queryKey != "limit" && queryKey != "offset" && queryKey != "sort" && queryKey != "search" {
			fail(w, 400, "object query parameter not allowed")
			return
		}
		for _, value := range values {
			if len(value) > 256 || strings.Contains(value, "..") {
				fail(w, 400, "invalid object query")
				return
			}
			if queryKey == "limit" || queryKey == "offset" {
				n, err := strconv.Atoi(value)
				max := 10000
				if queryKey == "limit" {
					max = 1000
				}
				if err != nil || n < 0 || n > max {
					fail(w, 400, "invalid pagination")
					return
				}
			}
		}
	}
	proxy(w, r, "/api/storage/buckets/slp_"+app+"/objects/"+url.PathEscape(key), r.URL.Query())
}
func ai(w http.ResponseWriter, r *http.Request) {
	b, e := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if e != nil {
		fail(w, 400, "invalid body")
		return
	}
	var req map[string]any
	if json.Unmarshal(b, &req) != nil {
		fail(w, 400, "invalid JSON")
		return
	}
	req["model"] = os.Getenv("SLOPPER_AI_MODEL")
	b, _ = json.Marshal(req)
	r.Body = io.NopCloser(strings.NewReader(string(b)))
	r.ContentLength = int64(len(b))
	proxy(w, r, "/api/ai/chat/completion", nil)
}
func proxy(w http.ResponseWriter, r *http.Request, path string, q url.Values) {
	base := os.Getenv("SLOPPER_UPSTREAM")
	if strings.HasPrefix(path, "/slp_") {
		base = os.Getenv("SLOPPER_POSTGREST")
	}
	target := strings.TrimRight(base, "/") + path
	if q != nil {
		target += "?" + q.Encode()
	}
	req, e := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if e != nil {
		fail(w, 502, "upstream error")
		return
	}
	req.Header = r.Header.Clone()
	req.Header.Del("X-Slopper-Proxy")
	req.Header.Del("X-Slopper-CSRF")
	req.Header.Del("Cookie")
	req.Header.Del("Authorization")
	req.Header.Del("X-Api-Key")
	req.Header.Set("Accept-Profile", "public")
	req.Header.Set("Content-Profile", "public")
	if !strings.HasPrefix(path, "/slp_") {
		req.Header.Set("Authorization", "Bearer "+os.Getenv("SLOPPER_INSFORGE_ANON_KEY"))
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		fail(w, 502, "upstream unavailable")
		return
	}
	defer resp.Body.Close()
	for k, v := range resp.Header {
		for _, x := range v {
			w.Header().Add(k, x)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, 101<<20))
}
func registered(app, name, kind string) bool {
	p := os.Getenv("SLOPPER_MANIFEST")
	if p == "" {
		p = "/etc/slopper/manifest.json"
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return false
	}
	var m resourceManifest
	if json.Unmarshal(b, &m) != nil {
		return false
	}
	a, ok := m.Applications[app]
	if !ok {
		return false
	}
	xs := a.Tables
	if kind == "bucket" {
		xs = a.Buckets
	}
	for _, x := range xs {
		if x == name {
			return true
		}
	}
	return false
}
