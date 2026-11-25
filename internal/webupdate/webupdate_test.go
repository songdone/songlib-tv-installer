package webupdate

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func newPackagedFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":       {Data: []byte(`<!doctype html><html><head><title>packaged</title></head><body><script type="module" src="./assets/app.js"></script></body></html>`)},
		"assets/app.js":    {Data: []byte(`console.log("packaged");`)},
		"songlib-mark.png": {Data: []byte("png")},
	}
}

func makeSignedFeed(t *testing.T, priv ed25519.PrivateKey, keyID string, manifest Manifest, files map[string][]byte) ([]byte, string) {
	t.Helper()
	manifest.Files = nil
	for p, content := range files {
		sum := sha256.Sum256(content)
		manifest.Files = append(manifest.Files, File{
			Path:   p,
			Size:   int64(len(content)),
			SHA256: hex.EncodeToString(sum[:]),
		})
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	env, err := json.Marshal(Envelope{
		KeyID:     keyID,
		Payload:   payload,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payload)),
	})
	if err != nil {
		t.Fatal(err)
	}
	rev := sha256.Sum256(payload)
	return env, hex.EncodeToString(rev[:])
}

func TestValidPathAllowlistAndRejections(t *testing.T) {
	allowed := []string{
		"index.html",
		"songlib-logo.svg",
		"songlib-mark.png",
		"apple-touch-icon.png",
		"assets/index.abc123.js",
		"assets/index.abc123.css",
		"tv/playing.jpg",
		"plan/cover-pro.jpg",
		"img/dummy.jpg",
	}
	for _, p := range allowed {
		if !ValidPath(p) {
			t.Errorf("ValidPath(%q) = false, want true", p)
		}
	}
	rejected := []string{
		"",
		"/index.html",
		"../index.html",
		"assets/app.js.map",
		"assets/.hidden.js",
		"api/status.js",
		"ws/install.js",
		"management/login.html",
		"songlib/config.json",
		"apps/1/icon.png",
		"evil.js",
		"tv/evil.js",
		"config/settings.json",
	}
	for _, p := range rejected {
		if ValidPath(p) {
			t.Errorf("ValidPath(%q) = true, want false", p)
		}
	}
}

func TestVerifyEnvelopeRejectsDuplicateKeysAndBadSig(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	dupPayload := []byte(`{"schema":1,"schema":1,"version":"1.1.0-ui.1","sequence":1,"min_backend_version":"1.1.0","backend_id":"abc","files":[{"path":"index.html","size":1,"sha256":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"}]}`)
	env, _ := json.Marshal(Envelope{
		KeyID:     "k1",
		Payload:   dupPayload,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, dupPayload)),
	})
	if _, err := VerifyEnvelope(env, "k1", pub); err == nil {
		t.Fatal("expected duplicate JSON key rejection")
	}
	if _, err := VerifyEnvelope(env, "wrong-key", pub); err == nil {
		t.Fatal("expected wrong key_id rejection")
	}
}

func TestTwoRoundApplyRollbackRestoreAndRecovery(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const backendID = "backend-abc123"
	const keyID = "songlib-web-2026"

	var currentManifest []byte
	var currentFiles map[string][]byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/feed/manifest.json":
			_, _ = w.Write(currentManifest)
		case strings.HasPrefix(r.URL.Path, "/feed/files/"):
			rel := strings.TrimPrefix(r.URL.Path, "/feed/files/")
			data, ok := currentFiles[rel]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(data)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	dataDir := t.TempDir()
	cfg := Config{
		DataDir:        dataDir,
		PackagedFS:     newPackagedFS(),
		BackendVersion: "1.1.0",
		BackendID:      backendID,
		FeedURL:        srv.URL + "/feed/manifest.json",
		KeyID:          keyID,
		PublicKey:      pub,
		AllowHTTP:      true,
	}
	mgr, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Round 1: apply 1.1.0-ui.1
	files1 := map[string][]byte{
		"index.html":    []byte(`<!doctype html><html><head></head><body>v1</body></html>`),
		"assets/app.js": []byte(`export const v = 1;`),
		"assets/ui.css": []byte(`body { color: #111; }`),
	}
	m1, rev1 := makeSignedFeed(t, priv, keyID, Manifest{
		Schema:            1,
		Version:           "1.1.0-ui.1",
		Sequence:          1,
		MinBackendVersion: "1.1.0",
		BackendID:         backendID,
		Notes:             "round 1",
	}, files1)
	currentManifest, currentFiles = m1, files1

	st, err := mgr.Check(context.Background())
	if err != nil {
		t.Fatalf("Check round 1: %v", err)
	}
	if st.Available == nil || !st.Available.Compatible || st.Available.Revision != rev1 {
		t.Fatalf("unexpected available release: %+v", st.Available)
	}

	st, err = mgr.Apply(context.Background(), rev1)
	if err != nil {
		t.Fatalf("Apply round 1: %v", err)
	}
	if st.ActiveVersion != "1.1.0-ui.1" || st.ActiveRevision != rev1 || !st.CanRollback || !st.CanRestore {
		t.Fatalf("unexpected status after round 1: %+v", st)
	}

	// Round 2: apply 1.1.0-ui.2
	files2 := map[string][]byte{
		"index.html":    []byte(`<!doctype html><html><head></head><body>v2</body></html>`),
		"assets/app.js": []byte(`export const v = 2;`),
	}
	m2, rev2 := makeSignedFeed(t, priv, keyID, Manifest{
		Schema:            1,
		Version:           "1.1.0-ui.2",
		Sequence:          2,
		MinBackendVersion: "1.1.0",
		BackendID:         backendID,
		Notes:             "round 2",
	}, files2)
	currentManifest, currentFiles = m2, files2

	if _, err := mgr.Check(context.Background()); err != nil {
		t.Fatalf("Check round 2: %v", err)
	}
	st, err = mgr.Apply(context.Background(), rev2)
	if err != nil {
		t.Fatalf("Apply round 2: %v", err)
	}
	if st.ActiveVersion != "1.1.0-ui.2" || st.PreviousVersion != "1.1.0-ui.1" || st.PreviousRevision != rev1 {
		t.Fatalf("unexpected status after round 2: %+v", st)
	}

	// Rollback swaps v2 -> v1 without lowering sequence watermark.
	st, err = mgr.Rollback(context.Background())
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if st.ActiveVersion != "1.1.0-ui.1" || st.PreviousVersion != "1.1.0-ui.2" {
		t.Fatalf("unexpected rollback status: %+v", st)
	}

	// Downgrade attempt (sequence 1 after watermark 2) must be refused!
	currentManifest, currentFiles = m1, files1
	if _, err := mgr.Check(context.Background()); !errors.Is(err, ErrStale) {
		t.Fatalf("expected ErrStale on downgrade, got %v", err)
	}

	// Restore packaged baseline keeps v1 as previous.
	st, err = mgr.RestorePackaged(context.Background())
	if err != nil {
		t.Fatalf("RestorePackaged: %v", err)
	}
	if st.ActiveRevision != "" || st.ActiveVersion != "1.1.0" || st.PreviousRevision != rev1 {
		t.Fatalf("unexpected restore status: %+v", st)
	}

	// Rollback restores v1 again, then simulate tampering on disk and restart.
	if _, err := mgr.Rollback(context.Background()); err != nil {
		t.Fatalf("Rollback back to v1: %v", err)
	}
	resolvedDir, _ := filepath.EvalSymlinks(dataDir)
	tampered := filepath.Join(resolvedDir, "web-updates", "versions", rev1, "assets", "app.js")
	if err := os.WriteFile(tampered, []byte(`export const v = 999;`), 0600); err != nil {
		t.Fatal(err)
	}
	// Runtime OpenFile must reject tampered bytes immediately.
	if _, err := mgr.OpenFile(rev1, "assets/app.js"); err == nil {
		t.Fatal("OpenFile served tampered snapshot bytes")
	}
	// Startup recovery must fall back to packaged baseline when active is corrupted.
	restarted, err := New(cfg)
	if err != nil {
		t.Fatalf("Restart New: %v", err)
	}
	if restarted.Status().ActiveRevision != "" {
		t.Fatalf("expected fallback to packaged baseline after active tamper, got %+v", restarted.Status())
	}
}

func TestRejectsBadSyntaxAndBackendMismatch(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const backendID = "backend-abc123"
	const keyID = "songlib-web-2026"

	filesBadJS := map[string][]byte{
		"index.html":    []byte(`<!doctype html><html></html>`),
		"assets/bad.js": []byte(`function broken( {`),
	}
	mBad, revBad := makeSignedFeed(t, priv, keyID, Manifest{
		Schema:            1,
		Version:           "1.1.0-ui.1",
		Sequence:          1,
		MinBackendVersion: "1.1.0",
		BackendID:         backendID,
	}, filesBadJS)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed/manifest.json" {
			_, _ = w.Write(mBad)
			return
		}
		rel := strings.TrimPrefix(r.URL.Path, "/feed/files/")
		_, _ = w.Write(filesBadJS[rel])
	}))
	defer srv.Close()

	mgr, err := New(Config{
		DataDir:        t.TempDir(),
		PackagedFS:     newPackagedFS(),
		BackendVersion: "1.1.0",
		BackendID:      backendID,
		FeedURL:        srv.URL + "/feed/manifest.json",
		KeyID:          keyID,
		PublicKey:      pub,
		AllowHTTP:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mgr.Apply(context.Background(), revBad); err == nil {
		t.Fatal("expected syntax error rejection on broken JS")
	}

	// Backend ID mismatch reports incompatible on Check and refuses Apply.
	filesOK := map[string][]byte{"index.html": []byte(`<!doctype html><html></html>`)}
	mOther, revOther := makeSignedFeed(t, priv, keyID, Manifest{
		Schema:            1,
		Version:           "1.2.0-ui.1",
		Sequence:          2,
		MinBackendVersion: "1.2.0",
		BackendID:         "different-backend",
	}, filesOK)
	srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(mOther)
	}))
	defer srv2.Close()

	mgr2, err := New(Config{
		DataDir:        t.TempDir(),
		PackagedFS:     newPackagedFS(),
		BackendVersion: "1.1.0",
		BackendID:      backendID,
		FeedURL:        srv2.URL + "/feed/manifest.json",
		KeyID:          keyID,
		PublicKey:      pub,
		AllowHTTP:      true,
	})
	if err != nil {
		t.Fatal(err)
	}
	st, err := mgr2.Check(context.Background())
	if err != nil {
		t.Fatalf("Check incompatible should succeed with Compatible=false, got %v", err)
	}
	if st.Available == nil || st.Available.Compatible {
		t.Fatalf("expected Compatible=false, got %+v", st.Available)
	}
	if _, err := mgr2.Apply(context.Background(), revOther); !errors.Is(err, ErrIncompatible) {
		t.Fatalf("expected ErrIncompatible, got %v", err)
	}
}
