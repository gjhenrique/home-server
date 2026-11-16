package main

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeTar(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	for name, body := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestDeployCreatesAndReusesContentAddressedRelease(t *testing.T) {
	root := filepath.Join(t.TempDir(), "app")
	archive := makeTar(t, map[string]string{"index.html": "hello"})
	if err := deploy(bytes.NewReader(archive), root); err != nil {
		t.Fatal(err)
	}
	first, err := os.Readlink(filepath.Join(root, "current"))
	if err != nil {
		t.Fatal(err)
	}
	if err = deploy(bytes.NewReader(archive), root); err != nil {
		t.Fatal(err)
	}
	second, _ := os.Readlink(filepath.Join(root, "current"))
	if first != second {
		t.Fatalf("identical deploy changed release: %s -> %s", first, second)
	}
	info, err := os.Stat(filepath.Join(root, first))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Fatalf("published release is not traversable: %v", info.Mode())
	}
}

func TestReleaseHashSeparatesAmbiguousFileStreams(t *testing.T) {
	ids := []string{}
	for _, files := range []map[string]string{{"index.html": "AB", "a": "C"}, {"index.html": "A", "a": "BC"}} {
		root := filepath.Join(t.TempDir(), "app")
		if err := deploy(bytes.NewReader(makeTar(t, files)), root); err != nil {
			t.Fatal(err)
		}
		link, err := os.Readlink(filepath.Join(root, "current"))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, filepath.Base(link))
	}
	if ids[0] == ids[1] {
		t.Fatalf("ambiguous file streams got same release id %s", ids[0])
	}
}
func TestDeployRejectsUnsafeArchives(t *testing.T) {
	cases := map[string]map[string]string{"no-index": {"main.js": "x"}, "traversal": {"index.html": "x", "../escape": "x"}, "normalized-traversal": {"index.html": "x", "a/../escape": "x"}, "dotfile": {"index.html": "x", ".env": "x"}}
	for name, files := range cases {
		t.Run(name, func(t *testing.T) {
			if err := deploy(bytes.NewReader(makeTar(t, files)), filepath.Join(t.TempDir(), "app")); err == nil {
				t.Fatal("expected archive rejection")
			}
		})
	}
}

func TestValidateSiteRejectsLinks(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		root := t.TempDir()
		_ = os.WriteFile(filepath.Join(root, "index.html"), []byte("x"), 0644)
		if err := os.Symlink("index.html", filepath.Join(root, "alias")); err != nil {
			t.Fatal(err)
		}
		if err := validateSite(root); err == nil {
			t.Fatal("symlink was accepted")
		}
	})
	t.Run("hardlink", func(t *testing.T) {
		root := t.TempDir()
		_ = os.WriteFile(filepath.Join(root, "index.html"), []byte("x"), 0644)
		if err := os.Link(filepath.Join(root, "index.html"), filepath.Join(root, "alias")); err != nil {
			t.Fatal(err)
		}
		if err := validateSite(root); err == nil {
			t.Fatal("hardlink was accepted")
		}
	})
}
func TestRollbackSwitchesRelease(t *testing.T) {
	root := filepath.Join(t.TempDir(), "app")
	for _, id := range []string{"1111111111111111", "2222222222222222"} {
		d := filepath.Join(root, "releases", id)
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(d, "index.html"), []byte(id), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := rollback(root, "1111111111111111"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.Readlink(filepath.Join(root, "current"))
	if filepath.Base(got) != "1111111111111111" {
		t.Fatalf("rollback points to %s", got)
	}
}

func TestPerAppMutationLockSerializes(t *testing.T) {
	root := filepath.Join(t.TempDir(), "app")
	lockRoot, app := filepath.Dir(root), filepath.Base(root)
	unlock, err := lockApp(lockRoot, app)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	acquired := make(chan struct{})
	done := make(chan func(), 1)
	go func() {
		u, e := lockApp(lockRoot, app)
		if e == nil {
			acquired <- struct{}{}
			done <- u
		}
	}()
	select {
	case <-acquired:
		t.Fatal("second mutation acquired lock concurrently")
	case <-time.After(30 * time.Millisecond):
	}
	unlock()
	select {
	case <-acquired:
	case <-time.After(time.Second):
		t.Fatal("second mutation did not acquire released lock")
	}
	(<-done)()
}

func TestReleaseRetentionPreservesActiveAndRecent(t *testing.T) {
	appRoot := t.TempDir()
	releases := filepath.Join(appRoot, "releases")
	if err := os.MkdirAll(releases, 0755); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"0000000000000001", "0000000000000002", "0000000000000003", "0000000000000004", "0000000000000005", "0000000000000006"} {
		if err := os.Mkdir(filepath.Join(releases, id), 0755); err != nil {
			t.Fatal(err)
		}
	}
	prior, _ := json.Marshal([]string{"0000000000000002", "0000000000000003", "0000000000000004", "0000000000000005", "0000000000000006"})
	if err := os.WriteFile(filepath.Join(appRoot, ".release-order.json"), prior, 0600); err != nil {
		t.Fatal(err)
	}
	if err := retain(appRoot, releases, "0000000000000001"); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(releases)
	if len(entries) != 5 {
		t.Fatalf("retained %d releases", len(entries))
	}
	if _, err := os.Stat(filepath.Join(releases, "0000000000000001")); err != nil {
		t.Fatal("active release removed")
	}
	if _, err := os.Stat(filepath.Join(releases, "0000000000000005")); err != nil {
		t.Fatal("most recently activated rollback target removed")
	}
	if _, err := os.Stat(filepath.Join(releases, "0000000000000006")); err == nil {
		t.Fatal("oldest release was retained")
	}
}

func TestDeploySendReturnsWhenSSHExitsEarly(t *testing.T) {
	bin := t.TempDir()
	sshPath := filepath.Join(bin, "ssh")
	if err := os.WriteFile(sshPath, []byte("#!/bin/sh\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("ok"), 0644); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- send("unused", Request{Command: "deploy", App: "app"}, dir) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected SSH failure")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("send hung after SSH exited")
	}
}

func FuzzArchivePathValidation(f *testing.F) {
	f.Add("../escape")
	f.Add("/absolute")
	f.Add(".secret")
	f.Add("assets/main.js")
	f.Fuzz(func(t *testing.T, name string) {
		if len(name) == 0 || len(name) > 256 {
			return
		}
		archive := makeTar(t, map[string]string{"index.html": "ok", name: "data"})
		root := filepath.Join(t.TempDir(), "app")
		_ = deploy(bytes.NewReader(archive), root)
		if _, err := os.Stat(filepath.Join(filepath.Dir(root), "escape")); err == nil {
			t.Fatal("archive wrote outside staging root")
		}
	})
}
