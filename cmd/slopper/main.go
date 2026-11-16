package main

import (
	"archive/tar"
	"bufio"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"home-server-public/internal/slopper"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

const maxArchive = 100 << 20

type Request struct {
	Command, App, Release string
	Yes                   bool
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "receive" {
		if e := receive(); e != nil {
			fmt.Fprintln(os.Stderr, e)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	host := fs.String("host", os.Getenv("SLOPPER_SSH_HOST"), "restricted SSH host")
	app := fs.String("app", "", "application DNS label")
	dir := fs.String("dir", "dist", "static site directory")
	release := fs.String("release", "", "release id")
	yes := fs.Bool("yes", false, "confirm deletion")
	_ = fs.Parse(os.Args[2:])
	if !slopper.ValidApp(*app) {
		fatal("invalid app label")
	}
	if *host == "" {
		fatal("set --host or SLOPPER_SSH_HOST")
	}
	if cmd == "delete" && !*yes {
		fatal("delete requires --yes")
	}
	req := Request{Command: cmd, App: *app, Release: *release, Yes: *yes}
	if cmd == "deploy" {
		if e := send(*host, req, *dir); e != nil {
			fatal(e.Error())
		}
		return
	}
	if !oneOf(cmd, "list", "status", "rollback", "delete") {
		usage()
	}
	if e := remote(*host, req); e != nil {
		fatal(e.Error())
	}
}
func usage() {
	fmt.Fprintln(os.Stderr, "usage: slopper {deploy|list|status|rollback|delete} --host HOST --app NAME [--dir dist] [--release ID] [--yes]")
	os.Exit(2)
}
func fatal(s string) { fmt.Fprintln(os.Stderr, s); os.Exit(1) }
func oneOf(x string, ys ...string) bool {
	for _, y := range ys {
		if x == y {
			return true
		}
	}
	return false
}
func remote(host string, r Request) error {
	b, _ := json.Marshal(r)
	cmd := exec.Command("ssh", "-T", "-o", "BatchMode=yes", host, "slopper-receiver")
	cmd.Stdin = strings.NewReader(string(b) + "\n")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
func send(host string, r Request, dir string) error {
	if e := validateSite(dir); e != nil {
		return e
	}
	cmd := exec.Command("ssh", "-T", "-o", "BatchMode=yes", host, "slopper-receiver")
	pw, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e := cmd.Start(); e != nil {
		return e
	}
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- writeArchive(pw, r, dir)
	}()
	waitErr := cmd.Wait()
	writeErr := <-writeDone
	if writeErr != nil {
		return writeErr
	}
	return waitErr
}

func writeArchive(dst io.WriteCloser, r Request, dir string) error {
	defer dst.Close()
	if e := json.NewEncoder(dst).Encode(r); e != nil {
		return e
	}
	tw := tar.NewWriter(dst)
	e := filepath.Walk(dir, func(p string, fi os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if p == dir {
			return nil
		}
		rel, e := filepath.Rel(dir, p)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(filepath.Base(p), ".") {
			return fmt.Errorf("dotfile rejected: %s", rel)
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() && !fi.IsDir() {
			return fmt.Errorf("non-regular path rejected: %s", rel)
		}
		h, e := tar.FileInfoHeader(fi, "")
		if e != nil {
			return e
		}
		h.Name = rel
		if e = tw.WriteHeader(h); e != nil {
			return e
		}
		if fi.IsDir() {
			return nil
		}
		f, e := os.Open(p)
		if e != nil {
			return e
		}
		_, copyErr := io.Copy(tw, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if e == nil {
		e = tw.Close()
	}
	if e != nil {
		return e
	}
	return nil
}
func validateSite(dir string) error {
	if !filepath.IsAbs(dir) {
		var e error
		dir, e = filepath.Abs(dir)
		if e != nil {
			return e
		}
	}
	info, e := os.Stat(dir)
	if e != nil || !info.IsDir() {
		return errors.New("site directory missing")
	}
	var total int64
	return filepath.Walk(dir, func(p string, i os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if p != dir && (strings.HasPrefix(filepath.Base(p), ".") || i.Mode()&os.ModeSymlink != 0 || !i.Mode().IsRegular() && !i.IsDir() || hardlinked(i)) {
			return fmt.Errorf("unsafe site entry: %s", p)
		}
		if i.Mode().IsRegular() {
			total += i.Size()
			if total > maxArchive {
				return errors.New("site upload exceeds 100 MiB")
			}
		}
		return nil
	})
}

// receiver consumes one bounded JSON line followed by a tar stream on stdin.
func receive() error {
	br := bufio.NewReaderSize(os.Stdin, 4097)
	line, e := br.ReadSlice('\n')
	if e != nil {
		return e
	}
	if len(line) > 4096 {
		return errors.New("request too large")
	}
	var r Request
	if json.Unmarshal(line, &r) != nil || !slopper.ValidApp(r.App) {
		return errors.New("invalid request")
	}
	root := os.Getenv("SLOPPER_ROOT")
	if root == "" {
		root = "/mnt/tank/slopper/sites"
	}
	appRoot := filepath.Join(root, r.App)
	unlock, err := lockApp(root, r.App)
	if err != nil {
		return err
	}
	defer unlock()
	switch r.Command {
	case "deploy":
		return deploy(br, appRoot)
	case "list":
		return list(appRoot)
	case "status":
		return status(appRoot)
	case "rollback":
		return rollback(appRoot, r.Release)
	case "delete":
		if !r.Yes {
			return errors.New("delete not confirmed")
		}
		return os.RemoveAll(appRoot)
	default:
		return errors.New("unsupported command")
	}
}
func deploy(src io.Reader, root string) error {
	tmp, e := os.MkdirTemp(filepath.Dir(root), ".stage-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(tmp)
	bounded := &countingReader{r: io.LimitReader(src, maxArchive+1)}
	tr := tar.NewReader(bounded)
	seen := map[string]bool{}
	total := int64(0)
	index := false
	for {
		h, e := tr.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		rawName := strings.ReplaceAll(h.Name, `\`, "/")
		traversal := strings.HasPrefix(rawName, "/") || regexp.MustCompile(`^[A-Za-z]:`).MatchString(rawName)
		for _, part := range strings.Split(rawName, "/") {
			if part == ".." {
				traversal = true
			}
		}
		name := filepath.ToSlash(filepath.Clean(rawName))
		dotPath := false
		for _, part := range strings.Split(name, "/") {
			if strings.HasPrefix(part, ".") {
				dotPath = true
			}
		}
		if traversal || name == "." || strings.HasPrefix(name, "/") || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, "../") || dotPath || seen[name] {
			return fmt.Errorf("unsafe or duplicate archive path: %s", name)
		}
		seen[name] = true
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA && h.Typeflag != tar.TypeDir {
			return errors.New("archive contains link or special file")
		}
		total += h.Size
		if total > maxArchive {
			return errors.New("archive too large")
		}
		dst := filepath.Join(tmp, filepath.FromSlash(name))
		if !strings.HasPrefix(dst, tmp+string(os.PathSeparator)) {
			return errors.New("path traversal")
		}
		if h.Typeflag == tar.TypeDir {
			if e = os.MkdirAll(dst, 0755); e != nil {
				return e
			}
			continue
		}
		if e = os.MkdirAll(filepath.Dir(dst), 0755); e != nil {
			return e
		}
		f, e := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		n, e := io.CopyN(f, tr, h.Size)
		ce := f.Close()
		if e != nil {
			return e
		}
		if ce != nil {
			return ce
		}
		if n != h.Size {
			return io.ErrUnexpectedEOF
		}
		if name == "index.html" {
			index = true
		}
	}
	if _, e := io.Copy(io.Discard, bounded); e != nil {
		return e
	}
	if bounded.n > maxArchive {
		return errors.New("archive too large")
	}
	if !index {
		return errors.New("index.html is required")
	}
	hash := sha256.New()
	if e = filepath.Walk(tmp, func(p string, i os.FileInfo, e error) error {
		if e == nil && i.Mode().IsRegular() {
			rel, _ := filepath.Rel(tmp, p)
			name := []byte(filepath.ToSlash(rel))
			_, _ = hash.Write(name)
			_, _ = hash.Write([]byte{0})
			var size [8]byte
			binary.BigEndian.PutUint64(size[:], uint64(i.Size()))
			_, _ = hash.Write(size[:])
			f, openErr := os.Open(p)
			if openErr != nil {
				return openErr
			}
			_, copyErr := io.Copy(hash, f)
			closeErr := f.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
		return e
	}); e != nil {
		return e
	}
	id := hex.EncodeToString(hash.Sum(nil))[:16]
	releaseRoot := filepath.Join(root, "releases")
	if e = os.MkdirAll(releaseRoot, 0755); e != nil {
		return e
	}
	dest := filepath.Join(releaseRoot, id)
	if _, e = os.Stat(dest); os.IsNotExist(e) {
		if e = os.Rename(tmp, dest); e != nil {
			return e
		}
	}
	if e = os.Chmod(dest, 0755); e != nil {
		return e
	}
	if e = os.MkdirAll(root, 0755); e != nil {
		return e
	}
	link := filepath.Join(root, ".current-new")
	os.Remove(link)
	if e = os.Symlink(filepath.Join("releases", id), link); e != nil {
		return e
	}
	if e = os.Rename(link, filepath.Join(root, "current")); e != nil {
		return e
	}
	fmt.Println(id)
	return retain(root, releaseRoot, id)
}
func list(root string) error {
	es, e := os.ReadDir(filepath.Join(root, "releases"))
	if e != nil {
		return e
	}
	for _, x := range es {
		fmt.Println(x.Name())
	}
	return nil
}
func status(root string) error {
	p, e := os.Readlink(filepath.Join(root, "current"))
	if e != nil {
		return e
	}
	fmt.Println(filepath.Base(p))
	return nil
}
func rollback(root, id string) error {
	if !regexp.MustCompile(`^[a-f0-9]{16}$`).MatchString(id) {
		return errors.New("invalid release id")
	}
	if _, e := os.Stat(filepath.Join(root, "releases", id, "index.html")); e != nil {
		return e
	}
	tmp := filepath.Join(root, ".current-new")
	os.Remove(tmp)
	if e := os.Symlink(filepath.Join("releases", id), tmp); e != nil {
		return e
	}
	if e := os.Rename(tmp, filepath.Join(root, "current")); e != nil {
		return e
	}
	return retain(root, filepath.Join(root, "releases"), id)
}
func retain(appRoot, root, active string) error {
	es, e := os.ReadDir(root)
	if e != nil {
		return e
	}
	orderPath := filepath.Join(appRoot, ".release-order.json")
	var prior []string
	if data, readErr := os.ReadFile(orderPath); readErr == nil {
		_ = json.Unmarshal(data, &prior)
	}
	names := []string{}
	exists := map[string]bool{}
	for _, x := range es {
		if x.IsDir() && regexp.MustCompile(`^[a-f0-9]{16}$`).MatchString(x.Name()) {
			names = append(names, x.Name())
			exists[x.Name()] = true
		}
	}
	ordered := []string{active}
	seen := map[string]bool{active: true}
	for _, x := range prior {
		if exists[x] && !seen[x] {
			ordered = append(ordered, x)
			seen[x] = true
		}
	}
	for _, x := range names {
		if !seen[x] {
			ordered = append(ordered, x)
			seen[x] = true
		}
	}
	if len(ordered) > 5 {
		ordered = ordered[:5]
	}
	data, e := json.Marshal(ordered)
	if e != nil {
		return e
	}
	tmp := orderPath + ".new"
	if e = os.WriteFile(tmp, data, 0600); e != nil {
		return e
	}
	if e = os.Rename(tmp, orderPath); e != nil {
		_ = os.Remove(tmp)
		return e
	}
	kept := map[string]bool{}
	for _, x := range ordered {
		kept[x] = true
	}
	for _, x := range names {
		if !kept[x] {
			if e = os.RemoveAll(filepath.Join(root, x)); e != nil {
				return e
			}
		}
	}
	return nil
}

func lockApp(root, app string) (func(), error) {
	lockRoot := filepath.Join(root, ".slopper-locks")
	if err := os.MkdirAll(lockRoot, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(lockRoot, app+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
}

func hardlinked(info os.FileInfo) bool {
	if !info.Mode().IsRegular() {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Nlink > 1
}

type countingReader struct {
	r io.Reader
	n int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, e := r.r.Read(p)
	r.n += int64(n)
	return n, e
}
