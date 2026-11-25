// Package webupdate installs signed, complete frontend snapshots without
// modifying the packaged application binary, backend, signing engine,
// configuration, or user pairing data.
package webupdate

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	MaxManifestSize = 256 << 10
	MaxFileSize     = 8 << 20
	MaxTotalSize    = 32 << 20
	MaxFiles        = 512
)

var (
	ErrBusy         = errors.New("web update operation already in progress")
	ErrUnavailable  = errors.New("web updates unavailable")
	ErrIncompatible = errors.New("web update incompatible with installed backend")
	ErrStale        = errors.New("web update revision changed or sequence is stale")
)

type Config struct {
	DataDir        string
	PackagedFS     fs.FS
	BackendVersion string
	BackendID      string
	FeedURL        string
	KeyID          string
	PublicKey      ed25519.PublicKey
	Client         *http.Client
	// AllowHTTP permits literal loopback IPs and localhost for development only.
	AllowHTTP bool
}

type Manifest struct {
	Schema            int    `json:"schema"`
	Version           string `json:"version"`
	Sequence          uint64 `json:"sequence"`
	MinBackendVersion string `json:"min_backend_version"`
	BackendID         string `json:"backend_id"`
	Files             []File `json:"files"`
	Notes             string `json:"notes,omitempty"`
}

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

type Envelope struct {
	KeyID     string          `json:"key_id"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

type Release struct {
	Version           string `json:"version"`
	Revision          string `json:"revision"`
	MinBackendVersion string `json:"min_backend_version"`
	BackendID         string `json:"backend_id"`
	Notes             string `json:"notes,omitempty"`
	Sequence          uint64 `json:"sequence"`
	Compatible        bool   `json:"compatible"`
	Reason            string `json:"reason,omitempty"`
}

type Status struct {
	Configured              bool     `json:"configured"`
	InstalledBackendVersion string   `json:"installed_backend_version"`
	BackendID               string   `json:"backend_id"`
	PackagedVersion         string   `json:"packaged_version"`
	ActiveVersion           string   `json:"active_version"`
	ActiveRevision          string   `json:"active_revision"`
	PreviousVersion         string   `json:"previous_version,omitempty"`
	PreviousRevision        string   `json:"previous_revision"`
	CanRollback             bool     `json:"can_rollback"`
	CanRestore              bool     `json:"can_restore"`
	Busy                    bool     `json:"busy"`
	Available               *Release `json:"available,omitempty"`
	Reason                  string   `json:"reason,omitempty"`
}

// VerifyEnvelope verifies the exact payload bytes without re-marshalling or
// canonicalizing JSON. Revision identifiers are SHA-256 of these same bytes.
func VerifyEnvelope(raw []byte, keyID string, key ed25519.PublicKey) (Manifest, error) {
	var manifest Manifest
	if len(raw) == 0 || len(raw) > MaxManifestSize {
		return manifest, errors.New("signed manifest exceeds size limit or is empty")
	}
	if keyID == "" || len(key) != ed25519.PublicKeySize {
		return manifest, errors.New("trusted Ed25519 key is not configured")
	}
	var envelope Envelope
	if err := decodeStrict(raw, &envelope); err != nil {
		return manifest, fmt.Errorf("invalid signed envelope: %w", err)
	}
	if envelope.KeyID != keyID {
		return manifest, errors.New("signed envelope key_id does not match trusted key")
	}
	signature, err := base64.StdEncoding.Strict().DecodeString(envelope.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(key, envelope.Payload, signature) {
		return manifest, errors.New("invalid Ed25519 manifest signature")
	}
	if err := decodeStrict(envelope.Payload, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest payload: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// ValidateManifest validates a complete frontend snapshot manifest. It is
// shared by the runtime verifier and the offline release publisher.
func ValidateManifest(manifest Manifest) error {
	if manifest.Schema != 1 {
		return errors.New("unsupported web manifest schema")
	}
	if _, err := parseVersion(manifest.Version); err != nil {
		return fmt.Errorf("invalid frontend version: %w", err)
	}
	if _, err := parseVersion(manifest.MinBackendVersion); err != nil {
		return fmt.Errorf("invalid minimum backend version: %w", err)
	}
	if manifest.Sequence == 0 {
		return errors.New("manifest sequence must be positive")
	}
	if !validBackendID(manifest.BackendID) {
		return errors.New("manifest requires a backend build identity")
	}
	if len(manifest.Notes) > 16<<10 || !utf8.ValidString(manifest.Notes) {
		return errors.New("manifest notes exceed limit or are not UTF-8")
	}
	if len(manifest.Files) == 0 || len(manifest.Files) > MaxFiles {
		return errors.New("manifest file count exceeds limit or is empty")
	}
	seen := make(map[string]bool, len(manifest.Files))
	var total int64
	for _, file := range manifest.Files {
		if err := validatePath(file.Path); err != nil {
			return err
		}
		folded := strings.ToLower(file.Path)
		if seen[folded] {
			return fmt.Errorf("duplicate manifest path %q", file.Path)
		}
		seen[folded] = true
		if file.Size <= 0 || file.Size > MaxFileSize {
			return fmt.Errorf("invalid size for %q (limit %d bytes)", file.Path, MaxFileSize)
		}
		total += file.Size
		if total > MaxTotalSize {
			return errors.New("snapshot exceeds total size limit")
		}
		if !validDigest(file.SHA256) {
			return fmt.Errorf("missing or invalid lowercase SHA-256 for %q", file.Path)
		}
	}
	if !seen["index.html"] {
		return errors.New("complete snapshot must include root index.html")
	}
	for name := range seen {
		for parent := path.Dir(name); parent != "."; parent = path.Dir(parent) {
			if seen[parent] {
				return fmt.Errorf("file/directory path collision at %q", parent)
			}
		}
	}
	return nil
}

// ValidPath reports whether name is an allowed built SongLib frontend asset.
func ValidPath(name string) bool { return validatePath(name) == nil }

func validatePath(name string) error {
	bad := func() error { return fmt.Errorf("unsafe or unsupported static path %q", name) }
	if name == "" || len(name) > 1024 || path.Clean(name) != name || strings.HasPrefix(name, "/") {
		return bad()
	}
	parts := strings.Split(name, "/")
	for _, part := range parts {
		if part == "" || len(part) > 255 || strings.HasPrefix(part, ".") || strings.HasSuffix(part, ".") {
			return bad()
		}
		for _, c := range part {
			if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
				return bad()
			}
		}
		switch strings.ToLower(part) {
		case "api", "ws", "management", "songlib", "apps", "ui", "config", "cmd", "backend", "db", "node_modules":
			return bad()
		}
	}
	if name == "index.html" {
		return nil
	}
	inAllowedDir := len(parts) > 1 && (parts[0] == "assets" || parts[0] == "img" || parts[0] == "plan" || parts[0] == "tv")
	switch strings.ToLower(path.Ext(name)) {
	case ".svg", ".png", ".jpg", ".jpeg", ".webp", ".ico":
		if len(parts) == 1 || inAllowedDir {
			return nil
		}
	case ".js", ".css":
		if len(parts) > 1 && parts[0] == "assets" {
			return nil
		}
	case ".woff", ".woff2", ".ttf", ".otf", ".eot", ".txt":
		if inAllowedDir {
			return nil
		}
	}
	return bad()
}

func validDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validBackendID(value string) bool {
	if value == "" || len(value) > 256 {
		return false
	}
	for _, c := range value {
		if c <= ' ' || c > '~' {
			return false
		}
	}
	return true
}

func payloadRevision(raw []byte) string {
	var envelope Envelope
	_ = json.Unmarshal(raw, &envelope) // The caller has already verified it.
	digest := sha256.Sum256(envelope.Payload)
	return hex.EncodeToString(digest[:])
}

func decodeStrict(raw []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if err := checkJSONValue(decoder, 0); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON data")
	}
	decoder = json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func checkJSONValue(decoder *json.Decoder, depth int) error {
	if depth > 32 {
		return errors.New("JSON nesting exceeds limit")
	}
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok || seen[key] {
				return errors.New("duplicate or invalid JSON object key")
			}
			seen[key] = true
			if err := checkJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := checkJSONValue(decoder, depth+1); err != nil {
				return err
			}
		}
	default:
		return errors.New("invalid JSON delimiter")
	}
	_, err = decoder.Token()
	return err
}

type semVersion struct {
	numbers [3]uint64
	pre     []string
}

func parseVersion(value string) (semVersion, error) {
	var result semVersion
	bad := func() (semVersion, error) { return result, fmt.Errorf("%q is not a semantic version", value) }
	if len(value) > 128 || value == "" {
		return bad()
	}
	value = strings.TrimPrefix(value, "v")
	main, buildMeta, hasBuild := strings.Cut(value, "+")
	if hasBuild && !validIdentifiers(buildMeta, false) {
		return bad()
	}
	main, pre, hasPre := strings.Cut(main, "-")
	if hasPre {
		if !validIdentifiers(pre, true) {
			return bad()
		}
		result.pre = strings.Split(pre, ".")
	}
	parts := strings.Split(main, ".")
	if len(parts) != 3 {
		return bad()
	}
	for i, part := range parts {
		if !decimal(part) || len(part) > 1 && part[0] == '0' {
			return bad()
		}
		number, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return bad()
		}
		result.numbers[i] = number
	}
	return result, nil
}

func decimal(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validIdentifiers(value string, prerelease bool) bool {
	for _, part := range strings.Split(value, ".") {
		if part == "" || prerelease && decimal(part) && len(part) > 1 && part[0] == '0' {
			return false
		}
		for _, c := range part {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '-') {
				return false
			}
		}
	}
	return true
}

func compareVersion(a, b semVersion) int {
	for i := range a.numbers {
		if a.numbers[i] < b.numbers[i] {
			return -1
		}
		if a.numbers[i] > b.numbers[i] {
			return 1
		}
	}
	if len(a.pre) == 0 && len(b.pre) == 0 {
		return 0
	}
	if len(a.pre) == 0 {
		return 1
	}
	if len(b.pre) == 0 {
		return -1
	}
	for i := 0; i < len(a.pre) && i < len(b.pre); i++ {
		left, right := a.pre[i], b.pre[i]
		if left == right {
			continue
		}
		ln, rn := decimal(left), decimal(right)
		if ln && rn && len(left) != len(right) {
			if len(left) < len(right) {
				return -1
			}
			return 1
		}
		if ln != rn {
			if ln {
				return -1
			}
			return 1
		}
		if left < right {
			return -1
		}
		return 1
	}
	if len(a.pre) < len(b.pre) {
		return -1
	}
	if len(a.pre) > len(b.pre) {
		return 1
	}
	return 0
}
