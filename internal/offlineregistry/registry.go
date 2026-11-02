// Package offlineregistry exposes exactly one bundled image on a loopback registry.
// It never invokes Docker or accepts writes, and needs no elevated privileges.
package offlineregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const MediaType = "application/vnd.docker.distribution.manifest.v2+json"

var digestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

type Descriptor struct {
	Repository     string `json:"repository"`
	Tag            string `json:"tag"`
	ManifestDigest string `json:"manifest_digest"`
	Architecture   string `json:"architecture"`
}
type Registry struct {
	root       string
	descriptor Descriptor
	manifest   []byte
	allowed    map[string]int64
	files      map[string]*os.File
}

func Open(root string) (*Registry, error) {
	raw, err := os.ReadFile(filepath.Join(root, "descriptor.json"))
	if err != nil {
		return nil, err
	}
	var descriptor Descriptor
	if err = json.Unmarshal(raw, &descriptor); err != nil {
		return nil, err
	}
	if descriptor.Repository != "songlib-tv-installer" || !digestPattern.MatchString(descriptor.ManifestDigest) {
		return nil, fmt.Errorf("invalid bundled image descriptor")
	}
	manifest, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(manifest)
	if "sha256:"+hex.EncodeToString(sum[:]) != descriptor.ManifestDigest {
		return nil, fmt.Errorf("bundled manifest digest mismatch")
	}
	var parsed struct {
		SchemaVersion int `json:"schemaVersion"`
		Config        struct {
			Digest string `json:"digest"`
			Size   int64  `json:"size"`
		} `json:"config"`
		Layers []struct {
			Digest string `json:"digest"`
			Size   int64  `json:"size"`
		} `json:"layers"`
	}
	if err = json.Unmarshal(manifest, &parsed); err != nil {
		return nil, err
	}
	if parsed.SchemaVersion != 2 {
		return nil, fmt.Errorf("unsupported image manifest")
	}
	allowed := map[string]int64{parsed.Config.Digest: parsed.Config.Size}
	for _, layer := range parsed.Layers {
		allowed[layer.Digest] = layer.Size
	}
	files := make(map[string]*os.File)
	success := false
	defer func() {
		if !success {
			for _, file := range files {
				_ = file.Close()
			}
		}
	}()
	for digest, size := range allowed {
		if !digestPattern.MatchString(digest) || size < 0 {
			return nil, fmt.Errorf("invalid image blob descriptor")
		}
		info, e := os.Lstat(filepath.Join(root, "blobs", "sha256", strings.TrimPrefix(digest, "sha256:")))
		if e != nil {
			return nil, e
		}
		if !info.Mode().IsRegular() || info.Size() != size {
			return nil, fmt.Errorf("invalid image blob size")
		}
		file, e := os.Open(filepath.Join(root, "blobs", "sha256", strings.TrimPrefix(digest, "sha256:")))
		if e != nil {
			return nil, e
		}
		files[digest] = file
	}
	success = true
	return &Registry{root: root, descriptor: descriptor, manifest: manifest, allowed: allowed, files: files}, nil
}
func (r *Registry) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Docker-Distribution-Api-Version", "registry/2.0")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		http.Error(w, "只读镜像源", http.StatusMethodNotAllowed)
		return
	}
	path := req.URL.Path
	if path == "/v2/" {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "{}")
		return
	}
	if path == "/health" {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(r.descriptor)
		return
	}
	prefix := "/v2/" + r.descriptor.Repository + "/"
	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, req)
		return
	}
	rest := strings.TrimPrefix(path, prefix)
	if strings.HasPrefix(rest, "manifests/") {
		reference := strings.TrimPrefix(rest, "manifests/")
		if reference != r.descriptor.Tag && reference != r.descriptor.ManifestDigest {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", MediaType)
		w.Header().Set("Docker-Content-Digest", r.descriptor.ManifestDigest)
		w.Header().Set("Content-Length", fmt.Sprint(len(r.manifest)))
		if req.Method == http.MethodGet {
			_, _ = w.Write(r.manifest)
		}
		return
	}
	if strings.HasPrefix(rest, "blobs/") {
		digest := strings.TrimPrefix(rest, "blobs/")
		if _, ok := r.allowed[digest]; !ok {
			http.NotFound(w, req)
			return
		}
		file := r.files[digest]
		info, err := file.Stat()
		if err != nil {
			http.Error(w, "镜像数据不可用", 500)
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Docker-Content-Digest", digest)
		http.ServeContent(w, req, "blob", info.ModTime(), io.NewSectionReader(file, 0, info.Size()))
		return
	}
	http.NotFound(w, req)
}

// Close releases the open blob handles. Keeping handles lets installation staging
// be renamed or unlinked while Docker is still fetching the original image.
func (r *Registry) Close() error {
	var first error
	for _, file := range r.files {
		if err := file.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
