// songlib-web-release writes a signed, complete frontend snapshot feed without
// publishing it or generating private keys.
//
//	Usage: go run ./cmd/songlib-web-release -web web/static/dist \
//	  -out build/<id>/release -version 1.1.0-ui.1 -sequence 1 \
//	  -min-backend 1.1.0 -backend-id <exact-backend-id> -key-id <id> \
//	  -signing-key /path/to/ed25519-pkcs8.pem [-notes-file /path/to/notes.txt]
//
// SONGLIB_WEB_SIGNING_KEY_FILE may supply the key file path instead. Input and
// output paths must not contain symlinks. Output must be a brand-new directory.
package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bitxeno/atvloadly/internal/webupdate"
	"golang.org/x/sys/unix"
)

const (
	maxNotesSize = 16 << 10
	maxKeySize   = 16 << 10
)

type options struct {
	web, out, version, minBackend, backendID, keyID, signingKey, notesFile string
	sequence                                                               uint64
}

type snapshotFile struct {
	metadata webupdate.File
	content  []byte
}

type releaseSummary struct {
	PublicKey string `json:"public_key"`
	KeyID     string `json:"key_id"`
	BackendID string `json:"backend_id"`
	Revision  string `json:"revision"`
}

func main() {
	if err := run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "songlib-web-release:", err)
		os.Exit(1)
	}
}

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) error {
	var opts options
	flags := flag.NewFlagSet("songlib-web-release", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&opts.web, "web", "", "complete built frontend directory (required)")
	flags.StringVar(&opts.out, "out", "", "new release directory; existing paths are refused (required)")
	flags.StringVar(&opts.version, "version", "", "frontend semantic version / UI release ID (required)")
	flags.Uint64Var(&opts.sequence, "sequence", 0, "positive release sequence (required)")
	flags.StringVar(&opts.minBackend, "min-backend", "", "minimum backend semantic version (required)")
	flags.StringVar(&opts.backendID, "backend-id", "", "exact compatible backend build ID (required)")
	flags.StringVar(&opts.keyID, "key-id", "", "trusted public signing key ID (required)")
	flags.StringVar(&opts.signingKey, "signing-key", "", "Ed25519 PKCS8 PEM file path (or SONGLIB_WEB_SIGNING_KEY_FILE)")
	flags.StringVar(&opts.notesFile, "notes-file", "", "optional UTF-8 release notes file, at most 16 KiB")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: songlib-web-release -web web/static/dist -out build/<id>/release -version <ui-id> -sequence N -min-backend 1.1.0 -backend-id <exact-backend-id> -key-id <id> -signing-key <PKCS8 PEM path> [-notes-file <path>]")
		fmt.Fprintln(stderr, "Writes manifest.json and the complete files/ snapshot locally. No publication or key generation. Source and output symlinks are refused.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	if opts.signingKey == "" {
		opts.signingKey = getenv("SONGLIB_WEB_SIGNING_KEY_FILE")
	}
	summary, err := publish(opts)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(summary)
}

func publish(opts options) (releaseSummary, error) {
	var summary releaseSummary
	if opts.web == "" || opts.out == "" || opts.version == "" || opts.minBackend == "" || opts.backendID == "" || opts.keyID == "" || opts.signingKey == "" {
		return summary, errors.New("-web, -out, -version, -sequence, -min-backend, -backend-id, -key-id and a signing key file path are required")
	}
	if len(opts.keyID) > 128 || strings.IndexFunc(opts.keyID, func(r rune) bool { return r <= ' ' || r > '~' }) >= 0 {
		return summary, errors.New("key ID must be 1–128 printable ASCII characters without whitespace")
	}
	web, err := filepath.Abs(opts.web)
	if err != nil {
		return summary, fmt.Errorf("resolve frontend path: %w", err)
	}
	out, err := filepath.Abs(opts.out)
	if err != nil {
		return summary, fmt.Errorf("resolve output path: %w", err)
	}
	if within(web, out) || within(out, web) {
		return summary, errors.New("frontend and output paths must not overlap")
	}
	if err := checkNewOutput(out); err != nil {
		return summary, err
	}
	root, err := openDirectory(web, false)
	if err != nil {
		return summary, fmt.Errorf("open frontend without symlinks: %w", err)
	}
	files, err := collectSnapshot(root)
	_ = root.Close()
	if err != nil {
		return summary, err
	}
	manifest := webupdate.Manifest{
		Schema:            1,
		Version:           opts.version,
		Sequence:          opts.sequence,
		MinBackendVersion: opts.minBackend,
		BackendID:         opts.backendID,
		Files:             make([]webupdate.File, 0, len(files)),
	}
	for _, file := range files {
		manifest.Files = append(manifest.Files, file.metadata)
	}
	if opts.notesFile != "" {
		notes, err := readBounded(opts.notesFile, maxNotesSize)
		if err != nil {
			return summary, fmt.Errorf("read release notes: %w", err)
		}
		manifest.Notes = string(notes)
	}
	if err := webupdate.ValidateManifest(manifest); err != nil {
		return summary, fmt.Errorf("validate complete snapshot: %w", err)
	}
	privateKey, err := loadSigningKey(opts.signingKey)
	if err != nil {
		return summary, err
	}
	defer clear(privateKey)
	payload, err := json.Marshal(manifest)
	if err != nil {
		return summary, fmt.Errorf("encode manifest payload: %w", err)
	}
	publicKey := privateKey.Public().(ed25519.PublicKey)
	envelope, err := json.Marshal(webupdate.Envelope{
		KeyID:     opts.keyID,
		Payload:   payload,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, payload)),
	})
	if err != nil {
		return summary, fmt.Errorf("encode signed envelope: %w", err)
	}
	envelope = append(envelope, '\n')
	if _, err := webupdate.VerifyEnvelope(envelope, opts.keyID, publicKey); err != nil {
		return summary, fmt.Errorf("verify generated envelope: %w", err)
	}
	if err := writeSnapshot(out, files, envelope); err != nil {
		return summary, err
	}
	digest := sha256.Sum256(payload)
	return releaseSummary{
		PublicKey: base64.StdEncoding.EncodeToString(publicKey),
		KeyID:     opts.keyID,
		BackendID: opts.backendID,
		Revision:  hex.EncodeToString(digest[:]),
	}, nil
}

func within(parent, child string) bool {
	rel, err := filepath.Rel(parent, child)
	return err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func openDirectory(name string, create bool) (*os.File, error) {
	if !filepath.IsAbs(name) {
		return nil, errors.New("directory must be absolute")
	}
	fd, err := unix.Open(string(filepath.Separator), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	current := os.NewFile(uintptr(fd), string(filepath.Separator))
	for _, part := range strings.Split(strings.TrimPrefix(filepath.Clean(name), string(filepath.Separator)), string(filepath.Separator)) {
		if part == "" {
			continue
		}
		next, err := unix.Openat(int(current.Fd()), part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if errors.Is(err, unix.ENOENT) && create {
			if mkdirErr := unix.Mkdirat(int(current.Fd()), part, 0755); mkdirErr != nil && !errors.Is(mkdirErr, unix.EEXIST) {
				_ = current.Close()
				return nil, mkdirErr
			}
			next, err = unix.Openat(int(current.Fd()), part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		}
		_ = current.Close()
		if err != nil {
			return nil, fmt.Errorf("directory component %q must be a real directory: %w", part, err)
		}
		current = os.NewFile(uintptr(next), part)
	}
	return current, nil
}

func checkNewOutput(out string) error {
	if _, err := os.Lstat(out); err == nil {
		return errors.New("output already exists; choose a new release directory")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect output: %w", err)
	}
	parent := filepath.Dir(out)
	for {
		dir, err := openDirectory(parent, false)
		if err == nil {
			_ = dir.Close()
			return nil
		}
		if !errors.Is(err, unix.ENOENT) {
			return fmt.Errorf("unsafe output parent: %w", err)
		}
		next := filepath.Dir(parent)
		if next == parent {
			return err
		}
		parent = next
	}
}

func collectSnapshot(root *os.File) ([]snapshotFile, error) {
	var files []snapshotFile
	var total int64
	var walk func(*os.File, string) error
	walk = func(dir *os.File, prefix string) error {
		entries, err := dir.ReadDir(-1)
		if err != nil {
			return fmt.Errorf("read frontend directory: %w", err)
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
		for _, entry := range entries {
			name := entry.Name()
			rel := path.Join(prefix, name)
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("source symlink is not allowed: %q", rel)
			}
			if entry.IsDir() {
				if !webupdate.ValidPath(path.Join(rel, "asset.png")) {
					return fmt.Errorf("unsupported source directory %q", rel)
				}
				fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
				if err != nil {
					return fmt.Errorf("open source directory %q: %w", rel, err)
				}
				child := os.NewFile(uintptr(fd), rel)
				err = walk(child, rel)
				_ = child.Close()
				if err != nil {
					return err
				}
				continue
			}
			if !webupdate.ValidPath(rel) {
				return fmt.Errorf("unsafe or unsupported source file %q", rel)
			}
			if len(files) >= webupdate.MaxFiles {
				return errors.New("snapshot exceeds file count limit")
			}
			fd, err := unix.Openat(int(dir.Fd()), name, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
			if err != nil {
				return fmt.Errorf("open source file %q: %w", rel, err)
			}
			file := os.NewFile(uintptr(fd), rel)
			content, err := readRegular(file, webupdate.MaxFileSize)
			_ = file.Close()
			if err != nil {
				return fmt.Errorf("read source file %q: %w", rel, err)
			}
			total += int64(len(content))
			if total > webupdate.MaxTotalSize {
				return errors.New("snapshot exceeds total size limit")
			}
			if err := webupdate.ValidateSyntax(rel, content); err != nil {
				return fmt.Errorf("validate source: %w", err)
			}
			digest := sha256.Sum256(content)
			files = append(files, snapshotFile{
				metadata: webupdate.File{Path: rel, Size: int64(len(content)), SHA256: hex.EncodeToString(digest[:])},
				content:  content,
			})
		}
		return nil
	}
	if err := walk(root, ""); err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].metadata.Path < files[j].metadata.Path })
	return files, nil
}

func readRegular(file *os.File, limit int64) ([]byte, error) {
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, errors.New("must be a regular file")
	}
	if info.Size() > limit {
		return nil, fmt.Errorf("file exceeds %d byte limit", limit)
	}
	content, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("file exceeds %d byte limit", limit)
	}
	if int64(len(content)) != info.Size() {
		return nil, errors.New("file size changed while reading")
	}
	return content, nil
}

func readBounded(name string, limit int64) ([]byte, error) {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return nil, err
	}
	parent, err := openDirectory(filepath.Dir(absolute), false)
	if err != nil {
		return nil, err
	}
	defer func() { _ = parent.Close() }()
	fd, err := unix.Openat(int(parent.Fd()), filepath.Base(absolute), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), absolute)
	defer func() { _ = file.Close() }()
	return readRegular(file, limit)
}

func loadSigningKey(name string) (ed25519.PrivateKey, error) {
	encoded, err := readBounded(name, maxKeySize)
	if err != nil {
		return nil, fmt.Errorf("read signing key file: %w", err)
	}
	defer clear(encoded)
	block, rest := pem.Decode(encoded)
	if block == nil || block.Type != "PRIVATE KEY" || len(block.Headers) != 0 || len(bytes.TrimSpace(rest)) != 0 {
		return nil, errors.New("signing key must contain one unencrypted PKCS8 PRIVATE KEY PEM block")
	}
	defer clear(block.Bytes)
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.New("signing key is not a valid unencrypted PKCS8 private key")
	}
	key, ok := parsed.(ed25519.PrivateKey)
	if !ok || len(key) != ed25519.PrivateKeySize {
		return nil, errors.New("signing key must be an Ed25519 PKCS8 private key")
	}
	return key, nil
}

func writeSnapshot(out string, files []snapshotFile, envelope []byte) error {
	parent, err := openDirectory(filepath.Dir(out), true)
	if err != nil {
		return fmt.Errorf("create safe output parent: %w", err)
	}
	defer func() { _ = parent.Close() }()
	if err := unix.Mkdirat(int(parent.Fd()), filepath.Base(out), 0755); err != nil {
		if errors.Is(err, unix.EEXIST) {
			return errors.New("output already exists; choose a new release directory")
		}
		return fmt.Errorf("create new output: %w", err)
	}
	fd, err := unix.Openat(int(parent.Fd()), filepath.Base(out), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("open new output: %w", err)
	}
	root := os.NewFile(uintptr(fd), out)
	defer func() { _ = root.Close() }()
	for _, file := range files {
		if err := writeRelative(root, "files/"+file.metadata.Path, file.content); err != nil {
			return fmt.Errorf("write snapshot %q: %w", file.metadata.Path, err)
		}
	}
	if err := writeRelative(root, "manifest.json", envelope); err != nil {
		return fmt.Errorf("write signed manifest: %w", err)
	}
	if err := root.Sync(); err != nil {
		return fmt.Errorf("sync release directory: %w", err)
	}
	return nil
}

func writeRelative(root *os.File, name string, content []byte) error {
	parts := strings.Split(name, "/")
	current := root
	var opened []*os.File
	defer func() {
		for _, dir := range opened {
			_ = dir.Close()
		}
	}()
	for _, part := range parts[:len(parts)-1] {
		if err := unix.Mkdirat(int(current.Fd()), part, 0755); err != nil && !errors.Is(err, unix.EEXIST) {
			return err
		}
		fd, err := unix.Openat(int(current.Fd()), part, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}
		current = os.NewFile(uintptr(fd), part)
		opened = append(opened, current)
	}
	fd, err := unix.Openat(int(current.Fd()), parts[len(parts)-1], unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0644)
	if err != nil {
		return err
	}
	file := os.NewFile(uintptr(fd), name)
	if _, err := io.Copy(file, bytes.NewReader(content)); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return current.Sync()
}
