package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/bitxeno/atvloadly/internal/webupdate"
)

func TestSongLibWebReleaseRoundTrip(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(tmp, "signing.pem")
	if err := os.WriteFile(keyPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8}), 0600); err != nil {
		t.Fatal(err)
	}
	webDir := filepath.Join(tmp, "dist")
	if err := os.MkdirAll(filepath.Join(webDir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "index.html"), []byte("<!doctype html><html></html>"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(webDir, "assets", "app.js"), []byte("console.log('ok');"), 0644); err != nil {
		t.Fatal(err)
	}
	outDir := filepath.Join(tmp, "release-1")

	var stdout, stderr bytes.Buffer
	args := []string{
		"-web", webDir,
		"-out", outDir,
		"-version", "1.1.0-ui.1",
		"-sequence", "1",
		"-min-backend", "1.1.0",
		"-backend-id", "abc123",
		"-key-id", "songlib-2026",
		"-signing-key", keyPath,
	}
	if err := run(args, func(string) string { return "" }, &stdout, &stderr); err != nil {
		t.Fatalf("run failed: %v (stderr=%s)", err, stderr.String())
	}
	var summary releaseSummary
	if err := json.Unmarshal(stdout.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.PublicKey != base64.StdEncoding.EncodeToString(pub) || summary.KeyID != "songlib-2026" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	raw, err := os.ReadFile(filepath.Join(outDir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := webupdate.VerifyEnvelope(raw, "songlib-2026", pub)
	if err != nil {
		t.Fatalf("VerifyEnvelope: %v", err)
	}
	if manifest.Version != "1.1.0-ui.1" || len(manifest.Files) != 2 {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}

	// Refuses overwriting existing output directory.
	if err := run(args, func(string) string { return "" }, &stdout, &stderr); err == nil {
		t.Fatal("expected refusal on existing output directory")
	}
}
