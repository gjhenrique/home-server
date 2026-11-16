package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSessionRequiresValidSignedRoleAndExpiry(t *testing.T) {
	key := "test signing key with enough entropy"
	t.Setenv("SLOPPER_JWT_SECRET", key)
	token := func(role string, exp int64) string {
		head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
		body, _ := json.Marshal(map[string]any{"sub": "u1", "email": "u@example.test", "roles": []string{role}, "exp": exp})
		payload := base64.RawURLEncoding.EncodeToString(body)
		m := hmac.New(sha256.New, []byte(key))
		m.Write([]byte(head + "." + payload))
		return head + "." + payload + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	}
	for _, tc := range []struct {
		name, role string
		exp        int64
		valid      bool
	}{{"allowed", "authp/user", time.Now().Add(time.Hour).Unix(), true}, {"wrong-role", "guest", time.Now().Add(time.Hour).Unix(), false}, {"expired", "authp/user", time.Now().Add(-time.Hour).Unix(), false}} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.AddCookie(&http.Cookie{Name: "AUTHP_ACCESS_TOKEN", Value: token(tc.role, tc.exp)})
			_, _, ok := session(r)
			if ok != tc.valid {
				t.Fatalf("session valid=%v want %v", ok, tc.valid)
			}
		})
	}
}

func TestAIForwardingUsesConfiguredModel(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ai/chat/completion" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer anon_fixture" {
			t.Errorf("missing server anon credential")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["model"] != "fixture/model" {
			t.Errorf("model = %v", body["model"])
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	}))
	defer upstream.Close()
	t.Setenv("SLOPPER_UPSTREAM", upstream.URL)
	t.Setenv("SLOPPER_AI_MODEL", "fixture/model")
	t.Setenv("SLOPPER_INSFORGE_ANON_KEY", "anon_fixture")
	r := httptest.NewRequest("POST", "/_slopper/ai/chat/completion", strings.NewReader(`{"model":"user/model","messages":[]}`))
	w := httptest.NewRecorder()
	ai(w, r)
	if w.Code != 200 {
		t.Fatalf("AI forward status %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabaseForwardingDropsCallerCredential(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("caller authorization leaked to PostgREST")
		}
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	t.Setenv("SLOPPER_POSTGREST", upstream.URL)
	t.Setenv("SLOPPER_UPSTREAM", "http://insforge:7130")
	r := httptest.NewRequest("GET", "/_slopper/db/items", nil)
	r.Header.Set("Authorization", "Bearer caller-token")
	w := httptest.NewRecorder()
	proxy(w, r, "/slp_app__items", nil)
	if w.Code != 200 {
		t.Fatalf("PostgREST proxy status %d", w.Code)
	}
}
func TestManifestFailsClosed(t *testing.T) {
	p := t.TempDir() + "/manifest.json"
	t.Setenv("SLOPPER_MANIFEST", p)
	if registered("app", "item", "table") {
		t.Fatal("missing manifest allowed resource")
	}
	if err := os.WriteFile(p, []byte(`{"applications":{"app":{"tables":["item"],"buckets":[]}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if !registered("app", "item", "table") || registered("app", "other", "table") {
		t.Fatal("manifest namespace validation failed")
	}
}

func TestDatabaseQueryAllowlist(t *testing.T) {
	for _, tc := range []struct {
		k, v  string
		valid bool
	}{{"select", "id,name", true}, {"select", "id(*)", false}, {"order", "created_at.desc", true}, {"limit", "10001", false}, {"name", "eq.Alice", true}, {"name", "in.(Alice,Bob)", false}, {"or", "(id.eq.1)", false}} {
		if got := validQueryValue(tc.k, tc.v); got != tc.valid {
			t.Errorf("validQueryValue(%q,%q)=%v want %v", tc.k, tc.v, got, tc.valid)
		}
	}
}

func TestGatewayRejectsSpoofedIdentityAndMissingCSRF(t *testing.T) {
	proxy := "proxy secret that is long enough for test"
	key := "JWT secret that is long enough for testing"
	t.Setenv("SLOPPER_PROXY_SECRET", proxy)
	t.Setenv("SLOPPER_JWT_SECRET", key)
	t.Setenv("SLOPPER_CSRF_SECRET", "CSRF secret that is long enough for tests")
	manifest := t.TempDir() + "/manifest.json"
	if err := os.WriteFile(manifest, []byte(`{"applications":{"app":{"tables":["items"],"buckets":[]}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SLOPPER_MANIFEST", manifest)
	r := httptest.NewRequest("GET", "https://app.slopper.example/_slopper/identity", nil)
	r.Host = "app.slopper.example"
	r.Header.Set("X-Slopper-Proxy", proxy)
	r.Header.Set("X-Forwarded-Email", "spoof@example.test")
	w := httptest.NewRecorder()
	serve(w, r)
	if w.Code != 401 {
		t.Fatalf("spoofed identity returned %d", w.Code)
	}
	r = httptest.NewRequest("POST", "https://app.slopper.example/_slopper/db/items", strings.NewReader(`{"x":1}`))
	r.Host = "app.slopper.example"
	r.Header.Set("Origin", "https://app.slopper.example")
	r.Header.Set("X-Slopper-Proxy", proxy)
	r.AddCookie(&http.Cookie{Name: "AUTHP_ACCESS_TOKEN", Value: signedTestToken(key)})
	w = httptest.NewRecorder()
	serve(w, r)
	if w.Code != 403 {
		t.Fatalf("missing CSRF returned %d", w.Code)
	}
}

func signedTestToken(key string) string {
	head := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256"}`))
	body, _ := json.Marshal(map[string]any{"sub": "u", "email": "u@example.test", "roles": []string{"authp/user"}, "exp": time.Now().Add(time.Hour).Unix()})
	payload := base64.RawURLEncoding.EncodeToString(body)
	m := hmac.New(sha256.New, []byte(key))
	m.Write([]byte(head + "." + payload))
	return head + "." + payload + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}
