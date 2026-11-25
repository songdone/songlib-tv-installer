package webupdate

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/sys/unix"
)

const envelopeName = ".manifest.json"

// openDirectory traverses directory descriptors with O_NOFOLLOW to prevent
// symlink substitution between checks and writes. Ancestors use traverseFlags
// (O_PATH on Linux) so fnOS package accounts with search-only parent ACLs work.
func openDirectory(name string, create bool) (*os.File, error) {
	if !filepath.IsAbs(name) || filepath.Clean(name) != name {
		return nil, errors.New("filesystem path must be clean and absolute")
	}
	var parts []string
	for _, part := range strings.Split(strings.TrimPrefix(name, string(filepath.Separator)), string(filepath.Separator)) {
		if part != "" {
			parts = append(parts, part)
		}
	}
	rootFlags := traverseFlags
	if len(parts) == 0 {
		rootFlags = unix.O_RDONLY
	}
	fd, err := unix.Open(string(filepath.Separator), rootFlags|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}
	for i, part := range parts {
		flags := traverseFlags
		if i == len(parts)-1 {
			flags = unix.O_RDONLY
		}
		next, openErr := unix.Openat(fd, part, flags|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		if errors.Is(openErr, unix.ENOENT) && create {
			mkdirErr := unix.Mkdirat(fd, part, 0700)
			if mkdirErr != nil && !errors.Is(mkdirErr, unix.EEXIST) {
				_ = unix.Close(fd)
				return nil, mkdirErr
			}
			if mkdirErr == nil {
				if err := syncDescriptor(fd); err != nil {
					_ = unix.Close(fd)
					return nil, err
				}
			}
			next, openErr = unix.Openat(fd, part, flags|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
		}
		_ = unix.Close(fd)
		if openErr != nil {
			return nil, fmt.Errorf("unsafe or inaccessible directory %q: %w", name, openErr)
		}
		fd = next
	}
	return os.NewFile(uintptr(fd), name), nil
}

func syncDescriptor(fd int) error {
	readable, err := unix.Openat(fd, ".", unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return err
	}
	defer func() { _ = unix.Close(readable) }()
	return unix.Fsync(readable)
}

func openRegular(name string, flags int) (*os.File, error) {
	directory, err := openDirectory(filepath.Dir(name), false)
	if err != nil {
		return nil, err
	}
	defer func() { _ = directory.Close() }()
	fd, err := unix.Openat(int(directory.Fd()), filepath.Base(name), flags|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = file.Close()
		if err == nil {
			err = errors.New("snapshot entry is not a regular file")
		}
		return nil, err
	}
	return file, nil
}

func readBounded(name string, limit int64) ([]byte, error) {
	file, err := openRegular(name, unix.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > limit {
		return nil, errors.New("stored file exceeds size limit")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("stored file exceeds size limit")
	}
	return data, nil
}

func writeSynced(name string, data []byte) error {
	file, err := openRegular(name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func randomName(prefix string) (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(value[:]), nil
}

func makeStage(root string) (string, error) {
	directory, err := openDirectory(root, false)
	if err != nil {
		return "", err
	}
	defer func() { _ = directory.Close() }()
	name, err := randomName(".staging-")
	if err != nil {
		return "", err
	}
	if err := unix.Mkdirat(int(directory.Fd()), name, 0700); err != nil {
		return "", err
	}
	if err := directory.Sync(); err != nil {
		_ = unix.Unlinkat(int(directory.Fd()), name, unix.AT_REMOVEDIR)
		return "", err
	}
	return filepath.Join(root, name), nil
}

func removeTree(name string) error {
	parent, err := openDirectory(filepath.Dir(name), false)
	if err != nil {
		return err
	}
	defer func() { _ = parent.Close() }()
	if err := removeEntry(int(parent.Fd()), filepath.Base(name)); err != nil {
		return err
	}
	return parent.Sync()
}

func removeEntry(parent int, name string) error {
	fd, err := unix.Openat(parent, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil
	}
	if errors.Is(err, unix.ENOTDIR) || errors.Is(err, unix.ELOOP) {
		return unix.Unlinkat(parent, name, 0)
	}
	if err != nil {
		return err
	}
	directory := os.NewFile(uintptr(fd), name)
	entries, err := directory.ReadDir(-1)
	if err == nil {
		for _, entry := range entries {
			if err = removeEntry(fd, entry.Name()); err != nil {
				break
			}
		}
	}
	_ = directory.Close()
	if err != nil {
		return err
	}
	return unix.Unlinkat(parent, name, unix.AT_REMOVEDIR)
}

// ValidateSyntax parses JavaScript and CSS with the in-process esbuild parser.
func ValidateSyntax(name string, content []byte) error {
	if err := validatePath(name); err != nil {
		return err
	}
	return syntaxCheck(name, content)
}

func syntaxCheck(name string, content []byte) error {
	var loader api.Loader
	switch strings.ToLower(path.Ext(name)) {
	case ".js":
		loader = api.LoaderJS
	case ".css":
		loader = api.LoaderCSS
	default:
		return nil
	}
	result := api.Transform(string(content), api.TransformOptions{
		Loader: loader, Sourcefile: name, LogLevel: api.LogLevelSilent,
	})
	if len(result.Errors) > 0 {
		return fmt.Errorf("invalid syntax in %q: %s", name, result.Errors[0].Text)
	}
	if loader == api.LoaderCSS {
		for _, warning := range result.Warnings {
			if warning.ID == "css-syntax-error" {
				return fmt.Errorf("invalid CSS syntax in %q: %s", name, warning.Text)
			}
		}
	}
	return nil
}

func (m *Manager) validateSnapshot(ctx context.Context, root, revision string) (Manifest, error) {
	raw, err := readBounded(filepath.Join(root, envelopeName), MaxManifestSize)
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := VerifyEnvelope(raw, m.config.KeyID, m.config.PublicKey)
	if err != nil {
		return Manifest{}, err
	}
	if payloadRevision(raw) != revision {
		return Manifest{}, errors.New("stored manifest revision does not match its directory")
	}
	if reason := m.compatibility(manifest); reason != "" {
		return Manifest{}, fmt.Errorf("%w: %s", ErrIncompatible, reason)
	}
	expected := map[string]bool{envelopeName: true}
	directories := map[string]bool{".": true}
	for _, file := range manifest.Files {
		expected[file.Path] = true
		for directory := path.Dir(file.Path); directory != "."; directory = path.Dir(directory) {
			directories[directory] = true
		}
	}
	if err := inspectTree(root, root, expected, directories); err != nil {
		return Manifest{}, err
	}
	for _, file := range manifest.Files {
		if err := ctx.Err(); err != nil {
			return Manifest{}, err
		}
		content, err := readBounded(filepath.Join(root, filepath.FromSlash(file.Path)), file.Size)
		if err != nil {
			return Manifest{}, fmt.Errorf("validate %q: %w", file.Path, err)
		}
		digest := sha256.Sum256(content)
		if int64(len(content)) != file.Size || hex.EncodeToString(digest[:]) != file.SHA256 {
			return Manifest{}, fmt.Errorf("stored file %q failed size/SHA-256 validation", file.Path)
		}
		if err := syntaxCheck(file.Path, content); err != nil {
			return Manifest{}, err
		}
	}
	return manifest, ctx.Err()
}

func inspectTree(root, current string, expected, directories map[string]bool) error {
	directory, err := openDirectory(current, false)
	if err != nil {
		return err
	}
	entries, err := directory.ReadDir(-1)
	_ = directory.Close()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		absolute := filepath.Join(current, entry.Name())
		relative, err := filepath.Rel(root, absolute)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() && expected[relative] {
			continue
		}
		if info.IsDir() && directories[relative] {
			if err := inspectTree(root, absolute, expected, directories); err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("unexpected or unsafe snapshot entry %q", relative)
	}
	return nil
}

func syncTree(root string) error {
	directory, err := openDirectory(root, false)
	if err != nil {
		return err
	}
	defer func() { _ = directory.Close() }()
	entries, err := directory.ReadDir(-1)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := syncTree(filepath.Join(root, entry.Name())); err != nil {
				return err
			}
		}
	}
	return directory.Sync()
}

func (m *Manager) install(ctx context.Context, raw []byte, manifest Manifest, revision string) (string, error) {
	stage, err := makeStage(m.root)
	if err != nil {
		return "", err
	}
	defer func() { _ = removeTree(stage) }()
	for _, item := range manifest.Files {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		response, err := m.get(ctx, m.fileURL(item.Path))
		if err != nil {
			return "", fmt.Errorf("download %q: %w", item.Path, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(response.Body, item.Size+1))
		_ = response.Body.Close()
		if readErr != nil {
			return "", fmt.Errorf("download %q: %w", item.Path, readErr)
		}
		if err := ctx.Err(); err != nil {
			return "", err
		}
		digest := sha256.Sum256(content)
		if int64(len(content)) != item.Size || hex.EncodeToString(digest[:]) != item.SHA256 {
			return "", fmt.Errorf("downloaded file %q failed size/SHA-256 validation", item.Path)
		}
		if err := syntaxCheck(item.Path, content); err != nil {
			return "", err
		}
		target := filepath.Join(stage, filepath.FromSlash(item.Path))
		directory, err := openDirectory(filepath.Dir(target), true)
		if err != nil {
			return "", err
		}
		_ = directory.Close()
		if err := writeSynced(target, content); err != nil {
			return "", err
		}
	}
	if err := writeSynced(filepath.Join(stage, envelopeName), raw); err != nil {
		return "", err
	}
	if err := syncTree(stage); err != nil {
		return "", err
	}
	if _, err := m.validateSnapshot(ctx, stage, revision); err != nil {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	versions := filepath.Join(m.root, "versions")
	target := filepath.Join(versions, revision)
	parent, err := openDirectory(m.root, false)
	if err != nil {
		return "", err
	}
	defer func() { _ = parent.Close() }()
	destination, err := openDirectory(versions, false)
	if err != nil {
		return "", err
	}
	defer func() { _ = destination.Close() }()
	if _, err := os.Lstat(target); err == nil {
		if _, err := m.validateSnapshot(ctx, target, revision); err == nil {
			if err := m.syncDir(destination); err != nil {
				return "", err
			}
			if err := m.syncDir(parent); err != nil {
				return "", err
			}
			return target, ctx.Err()
		} else if ctx.Err() != nil {
			return "", ctx.Err()
		}
		state := m.diskState()
		if state.Active == revision || state.HasPrevious && state.Previous == revision {
			return "", errors.New("referenced snapshot is damaged; restore the packaged frontend and restart before reinstalling")
		}
		quarantine, err := randomName(".staging-corrupt-")
		if err != nil {
			return "", err
		}
		if err := unix.Renameat(int(destination.Fd()), revision, int(parent.Fd()), quarantine); err != nil {
			return "", err
		}
		defer func() { _ = removeTree(filepath.Join(m.root, quarantine)) }()
		if err := m.syncDir(destination); err != nil {
			return "", err
		}
		if err := m.syncDir(parent); err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if err := unix.Renameat(int(parent.Fd()), filepath.Base(stage), int(destination.Fd()), revision); err != nil {
		return "", err
	}
	if err := m.syncDir(destination); err != nil {
		return "", err
	}
	if err := m.syncDir(parent); err != nil {
		return "", err
	}
	return target, ctx.Err()
}

type diskState struct {
	Schema          int    `json:"schema"`
	BackendID       string `json:"backend_id"`
	Active          string `json:"active"`
	Previous        string `json:"previous"`
	HasPrevious     bool   `json:"has_previous"`
	HighestSequence uint64 `json:"highest_sequence"`
	HighestRevision string `json:"highest_revision"`
}

func (m *Manager) writeState(ctx context.Context, state diskState) (committed bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return false, err
	}
	directory, err := openDirectory(m.root, false)
	if err != nil {
		return false, err
	}
	defer func() { _ = directory.Close() }()
	name, err := randomName(".state-")
	if err != nil {
		return false, err
	}
	defer func() { _ = unix.Unlinkat(int(directory.Fd()), name, 0) }()
	if err := m.writeStateFile(filepath.Join(m.root, name), raw); err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := unix.Renameat(int(directory.Fd()), name, int(directory.Fd()), "state.json"); err != nil {
		return false, err
	}
	return true, m.syncDir(directory)
}
