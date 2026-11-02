package offlineregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	blobs := filepath.Join(root, "blobs", "sha256")
	if err := os.MkdirAll(blobs, 0755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"architecture":"amd64","os":"linux"}`)
	sum := sha256.Sum256(data)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	if err := os.WriteFile(filepath.Join(blobs, strings.TrimPrefix(digest, "sha256:")), data, 0644); err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]any{"schemaVersion": 2, "config": map[string]any{"digest": digest, "size": len(data)}, "layers": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	sum = sha256.Sum256(raw)
	descriptor := Descriptor{Repository: "songlib-tv-installer", Tag: "0.1.2", ManifestDigest: "sha256:" + hex.EncodeToString(sum[:]), Architecture: "amd64"}
	meta, err := json.Marshal(descriptor)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string][]byte{"manifest.json": raw, "descriptor.json": meta} {
		if err := os.WriteFile(filepath.Join(root, name), content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
func TestReadOnlyBundledRegistry(t *testing.T) {
	root := fixture(t)
	registry, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = registry.Close() })
	server := httptest.NewServer(registry)
	defer server.Close()
	for _, test := range []struct {
		method, path string
		status       int
	}{{"GET", "/v2/", 200}, {"GET", "/health", 200}, {"GET", "/v2/songlib-tv-installer/manifests/0.1.2", 200}, {"HEAD", "/v2/songlib-tv-installer/manifests/0.1.2", 200}, {"GET", "/v2/other/manifests/0.1.2", 404}, {"POST", "/v2/songlib-tv-installer/blobs/uploads/", 405}, {"GET", "/v2/songlib-tv-installer/blobs/../../etc/passwd", 404}} {
		req, e := http.NewRequest(test.method, server.URL+test.path, nil)
		if e != nil {
			t.Fatal(e)
		}
		response, e := http.DefaultClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		if response.StatusCode != test.status {
			t.Errorf("%s %s: %d", test.method, test.path, response.StatusCode)
		}
		_ = response.Body.Close()
	}
	for digest := range registry.allowed {
		response, e := http.Get(server.URL + "/v2/songlib-tv-installer/blobs/" + digest)
		if e != nil {
			t.Fatal(e)
		}
		raw, e := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(raw)
		if "sha256:"+hex.EncodeToString(sum[:]) != digest {
			t.Fatal("blob digest mismatch")
		}
	}
}
func TestRejectsCorruptManifestAndMissingBlob(t *testing.T) {
	root := fixture(t)
	if err := os.WriteFile(filepath.Join(root, "manifest.json"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root); err == nil {
		t.Fatal("corrupt manifest accepted")
	}
	root = fixture(t)
	files, err := filepath.Glob(filepath.Join(root, "blobs/sha256/*"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(files[0]); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(root); err == nil {
		t.Fatal("missing blob accepted")
	}
}
func TestStagingCanDisappearDuringImageFetch(t *testing.T) {
	root := fixture(t)
	registry, err := Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = registry.Close() }()
	if err = os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(registry)
	defer server.Close()
	for digest := range registry.allowed {
		response, e := http.Get(server.URL + "/v2/songlib-tv-installer/blobs/" + digest)
		if e != nil {
			t.Fatal(e)
		}
		raw, e := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(raw)
		if response.StatusCode != 200 || "sha256:"+hex.EncodeToString(sum[:]) != digest {
			t.Fatal("staging cleanup interrupted image fetch")
		}
	}
}
