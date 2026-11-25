package webupdate

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

type Manager struct {
	config         Config
	root           string
	feed           *url.URL
	client         *http.Client
	trust          bool
	syncDir        func(*os.File) error
	writeStateFile func(string, []byte) error

	operation sync.Mutex
	mu        sync.RWMutex
	state     diskState
	active    *Manifest
	previous  *Manifest
	available *Release
	busy      bool
	reason    string
}

// New initializes the snapshot manager without making network requests. It
// verifies any previously committed snapshots referenced by state.json and
// falls back safely to the previous snapshot or packaged go:embed baseline.
func New(config Config) (*Manager, error) {
	if config.DataDir == "" || config.PackagedFS == nil {
		return nil, errors.New("web update data directory and packaged frontend FS are required")
	}
	var err error
	config.DataDir, err = filepath.Abs(config.DataDir)
	if err != nil {
		return nil, err
	}
	// Resolve symlinks in DataDir if any exist on macOS (/var -> /private/var)
	// before strict O_NOFOLLOW traversal inside web-updates/.
	if resolved, evalErr := filepath.EvalSymlinks(config.DataDir); evalErr == nil {
		config.DataDir = resolved
	}
	dir, err := openDirectory(config.DataDir, true)
	if err != nil {
		return nil, err
	}
	_ = dir.Close()

	index, err := fs.ReadFile(config.PackagedFS, "index.html")
	if err != nil || len(index) == 0 {
		return nil, fmt.Errorf("packaged frontend index.html is missing or invalid: %v", err)
	}
	if len(config.PublicKey) != 0 && len(config.PublicKey) != ed25519.PublicKeySize {
		return nil, errors.New("web update public key must be an Ed25519 public key")
	}
	if config.KeyID != "" && (!validBackendID(config.KeyID) || len(config.KeyID) > 128) {
		return nil, errors.New("invalid web update key ID")
	}
	config.PublicKey = append(ed25519.PublicKey(nil), config.PublicKey...)
	root := filepath.Join(config.DataDir, "web-updates")
	manager := &Manager{
		config:         config,
		root:           root,
		client:         updateClient(config.Client),
		trust:          config.KeyID != "" && len(config.PublicKey) == ed25519.PublicKeySize,
		state:          diskState{Schema: 1, BackendID: config.BackendID},
		syncDir:        (*os.File).Sync,
		writeStateFile: writeSynced,
	}
	if config.FeedURL != "" {
		manager.feed, err = parseFeed(config.FeedURL, config.AllowHTTP)
		if err != nil {
			return nil, err
		}
	}
	for _, name := range []string{root, filepath.Join(root, "versions")} {
		directory, err := openDirectory(name, true)
		if err != nil {
			return nil, err
		}
		_ = directory.Close()
	}
	if err := manager.cleanTemporary(); err != nil {
		return nil, fmt.Errorf("clean abandoned web update staging: %w", err)
	}
	manager.recover()
	return manager, nil
}

func (m *Manager) configured() bool {
	_, err := parseVersion(m.config.BackendVersion)
	return m.feed != nil && m.trust && validBackendID(m.config.BackendID) && err == nil
}

func (m *Manager) PackagedRevisionToken() string {
	return "packaged-" + m.config.BackendID
}

func (m *Manager) Status() Status {
	m.mu.RLock()
	defer m.mu.RUnlock()
	status := Status{
		Configured:              m.configured(),
		InstalledBackendVersion: m.config.BackendVersion,
		BackendID:               m.config.BackendID,
		PackagedVersion:         m.config.BackendVersion,
		ActiveVersion:           m.config.BackendVersion,
		ActiveRevision:          m.state.Active,
		PreviousRevision:        m.state.Previous,
		CanRollback:             m.state.HasPrevious,
		CanRestore:              m.state.Active != "",
		Busy:                    m.busy,
		Reason:                  m.reason,
	}
	if m.active != nil {
		status.ActiveVersion = m.active.Version
	}
	if m.state.HasPrevious {
		status.PreviousVersion = m.config.BackendVersion
		if m.previous != nil {
			status.PreviousVersion = m.previous.Version
		}
	}
	if m.available != nil {
		release := *m.available
		status.Available = &release
	}
	return status
}

// ReadPackaged reads an allowlisted file from the shipped go:embed frontend.
func (m *Manager) ReadPackaged(name string) ([]byte, error) {
	if !ValidPath(name) {
		return nil, os.ErrNotExist
	}
	data, err := fs.ReadFile(m.config.PackagedFS, name)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxFileSize {
		return nil, errors.New("packaged asset exceeds size limit")
	}
	return data, nil
}

// ReadRevisionFile reads an allowlisted file from either the packaged baseline
// (revision == "packaged" or revision == PackagedRevisionToken()) or a verified
// active/previous hot-update snapshot, rechecking exact size and SHA-256 on the
// opened descriptor for disk snapshots.
func (m *Manager) ReadRevisionFile(revision, name string) ([]byte, error) {
	if !ValidPath(name) {
		return nil, os.ErrNotExist
	}
	if revision == "" || revision == "packaged" || revision == m.PackagedRevisionToken() {
		return m.ReadPackaged(name)
	}
	file, err := m.OpenFile(revision, name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	return io.ReadAll(io.LimitReader(file, MaxFileSize+1))
}

// OpenFile serves only bytes named by an already verified active or previous
// manifest, rechecking the opened descriptor's size and SHA-256.
func (m *Manager) OpenFile(revision, name string) (*os.File, error) {
	m.mu.RLock()
	var manifest *Manifest
	if revision != "" && m.state.Active == revision {
		manifest = m.active
	} else if revision != "" && m.state.HasPrevious && m.state.Previous == revision {
		manifest = m.previous
	}
	var metadata *File
	if manifest != nil {
		for _, entry := range manifest.Files {
			if entry.Path == name {
				copied := entry
				metadata = &copied
				break
			}
		}
	}
	root := m.revisionRoot(revision)
	m.mu.RUnlock()
	if metadata == nil || !ValidPath(name) {
		return nil, os.ErrNotExist
	}
	file, err := openRegular(filepath.Join(root, filepath.FromSlash(name)), unix.O_RDONLY)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || info.Size() != metadata.Size {
		_ = file.Close()
		return nil, os.ErrNotExist
	}
	digest := sha256.New()
	n, err := io.Copy(digest, io.LimitReader(file, metadata.Size+1))
	if err != nil || n != metadata.Size || hex.EncodeToString(digest.Sum(nil)) != metadata.SHA256 {
		_ = file.Close()
		return nil, os.ErrNotExist
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		return nil, err
	}
	return file, nil
}

func (m *Manager) revisionRoot(revision string) string {
	return filepath.Join(m.root, "versions", revision)
}

func (m *Manager) run(ctx context.Context, action func() error) (Status, error) {
	if !m.operation.TryLock() {
		return m.Status(), ErrBusy
	}
	m.mu.Lock()
	m.busy = true
	m.mu.Unlock()
	err := ctx.Err()
	if err == nil {
		err = action()
	}
	m.mu.Lock()
	m.busy = false
	if err != nil {
		m.reason = err.Error()
	} else {
		m.reason = ""
	}
	m.mu.Unlock()
	m.operation.Unlock()
	return m.Status(), err
}

// Check fetches only signed metadata. Incompatible signed releases are reported
// via Available.Compatible/Reason; malformed or downgraded feeds return errors.
func (m *Manager) Check(ctx context.Context) (Status, error) {
	return m.run(ctx, func() error {
		if !m.configured() {
			return ErrUnavailable
		}
		_, manifest, revision, err := m.fetch(ctx)
		if err != nil {
			m.mu.Lock()
			m.available = nil
			m.mu.Unlock()
			return err
		}
		state := m.diskState()
		if err := checkSequence(state, manifest, revision); err != nil {
			return err
		}
		if manifest.Sequence > state.HighestSequence {
			state.HighestSequence, state.HighestRevision = manifest.Sequence, revision
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := m.commit(ctx, state, m.active, m.previous); err != nil {
				return err
			}
		}
		reason := m.compatibility(manifest)
		m.mu.Lock()
		m.available = &Release{
			Version:           manifest.Version,
			Revision:          revision,
			Sequence:          manifest.Sequence,
			MinBackendVersion: manifest.MinBackendVersion,
			BackendID:         manifest.BackendID,
			Notes:             manifest.Notes,
			Compatible:        reason == "",
			Reason:            reason,
		}
		m.mu.Unlock()
		return nil
	})
}

// Apply re-fetches the configured feed, requires the exact payload digest
// confirmed by the user, downloads every listed file into staging, verifies all
// hashes and JS/CSS syntax, and atomically switches state.json.
func (m *Manager) Apply(ctx context.Context, revision string) (Status, error) {
	return m.run(ctx, func() error {
		if !m.configured() {
			return ErrUnavailable
		}
		if !validDigest(revision) {
			return fmt.Errorf("%w: expected a SHA-256 release revision", ErrStale)
		}
		raw, manifest, actual, err := m.fetch(ctx)
		if err != nil {
			return err
		}
		if actual != revision {
			return fmt.Errorf("%w: signed release changed; check for updates again", ErrStale)
		}
		state := m.diskState()
		if err := checkSequence(state, manifest, revision); err != nil {
			return err
		}
		if reason := m.compatibility(manifest); reason != "" {
			return fmt.Errorf("%w: %s", ErrIncompatible, reason)
		}
		if _, err := m.install(ctx, raw, manifest, revision); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		previous := m.active
		if state.Active != revision {
			state.Previous, state.HasPrevious = state.Active, true
			state.Active = revision
		} else {
			previous = m.previous
		}
		if manifest.Sequence > state.HighestSequence {
			state.HighestSequence, state.HighestRevision = manifest.Sequence, revision
		}
		return m.commit(ctx, state, &manifest, previous)
	})
}

// Rollback swaps active and previous without lowering the sequence watermark.
func (m *Manager) Rollback(ctx context.Context) (Status, error) {
	return m.run(ctx, func() error {
		state := m.diskState()
		if !state.HasPrevious {
			return fmt.Errorf("%w: no previous frontend snapshot", ErrUnavailable)
		}
		var next *Manifest
		if state.Previous != "" {
			manifest, err := m.validateSnapshot(ctx, m.revisionRoot(state.Previous), state.Previous)
			if err != nil {
				return err
			}
			next = &manifest
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		state.Active, state.Previous = state.Previous, state.Active
		return m.commit(ctx, state, next, m.active)
	})
}

// RestorePackaged selects the shipped go:embed frontend and preserves the
// previously active hot revision as previous.
func (m *Manager) RestorePackaged(ctx context.Context) (Status, error) {
	return m.run(ctx, func() error {
		state := m.diskState()
		if state.Active == "" {
			return nil
		}
		state.Previous, state.HasPrevious = state.Active, true
		state.Active = ""
		return m.commit(ctx, state, nil, m.active)
	})
}

func (m *Manager) compatibility(manifest Manifest) string {
	if manifest.BackendID != m.config.BackendID || !validBackendID(m.config.BackendID) {
		return "frontend targets a different backend build; install the matching complete application package"
	}
	installed, err := parseVersion(m.config.BackendVersion)
	if err != nil {
		return "installed backend version is not a semantic version"
	}
	minimum, err := parseVersion(manifest.MinBackendVersion)
	if err != nil || compareVersion(installed, minimum) < 0 {
		return "frontend requires backend " + manifest.MinBackendVersion + " or later"
	}
	return ""
}

func checkSequence(state diskState, manifest Manifest, revision string) error {
	if manifest.Sequence < state.HighestSequence || manifest.Sequence == state.HighestSequence && revision != state.HighestRevision {
		return fmt.Errorf("%w: release sequence is older than or conflicts with the highest trusted release", ErrStale)
	}
	return nil
}

func (m *Manager) diskState() diskState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *Manager) commit(ctx context.Context, state diskState, active, previous *Manifest) error {
	committed, err := m.writeState(ctx, state)
	if committed {
		m.mu.Lock()
		m.state, m.active, m.previous = state, active, previous
		m.mu.Unlock()
	}
	if err != nil {
		return fmt.Errorf("save web update state: %w", err)
	}
	return nil
}

func validState(state diskState) bool {
	if state.Schema != 1 || !validBackendID(state.BackendID) || state.Active != "" && !validDigest(state.Active) || state.Previous != "" && !validDigest(state.Previous) {
		return false
	}
	if !state.HasPrevious && state.Previous != "" || state.HasPrevious && state.Active == state.Previous {
		return false
	}
	if state.HighestSequence == 0 {
		return state.HighestRevision == "" && state.Active == "" && !state.HasPrevious
	}
	return validDigest(state.HighestRevision)
}

func (m *Manager) recover() {
	raw, err := readBounded(filepath.Join(m.root, "state.json"), 16<<10)
	if errors.Is(err, os.ErrNotExist) {
		m.reason = "using packaged frontend; no saved update state"
		return
	}
	var state diskState
	if err != nil || decodeStrict(raw, &state) != nil || !validState(state) {
		m.reason = "using packaged frontend; saved update state is invalid"
		return
	}
	if state.BackendID != m.config.BackendID {
		m.reason = "using packaged frontend; backend build changed after a complete application update"
		_, err := m.writeState(context.Background(), m.state)
		if err != nil {
			m.reason += "; could not save reset state: " + err.Error()
		}
		return
	}
	if !m.trust && (state.Active != "" || state.HasPrevious && state.Previous != "") {
		m.reason = "using packaged frontend; trusted signing key is not configured"
		return
	}
	load := func(revision string) (*Manifest, error) {
		if revision == "" {
			return nil, nil
		}
		manifest, err := m.validateSnapshot(context.Background(), m.revisionRoot(revision), revision)
		if err == nil && (manifest.Sequence > state.HighestSequence || manifest.Sequence == state.HighestSequence && revision != state.HighestRevision) {
			err = errors.New("stored revision conflicts with release sequence watermark")
		}
		return &manifest, err
	}
	active, activeErr := load(state.Active)
	var previous *Manifest
	var previousErr error
	if state.HasPrevious {
		previous, previousErr = load(state.Previous)
	}
	changed := false
	if activeErr != nil {
		changed = true
		if state.HasPrevious && previousErr == nil {
			state.Active, active = state.Previous, previous
			m.reason = "active frontend failed validation; restored previous frontend: " + activeErr.Error()
		} else {
			state.Active, active = "", nil
			m.reason = "using packaged frontend; saved frontend failed validation: " + activeErr.Error()
		}
		state.Previous, state.HasPrevious, previous = "", false, nil
	} else if previousErr != nil {
		changed = true
		state.Previous, state.HasPrevious, previous = "", false, nil
		m.reason = "previous frontend failed validation and is unavailable: " + previousErr.Error()
	}
	m.state, m.active, m.previous = state, active, previous
	if changed {
		if _, err := m.writeState(context.Background(), state); err != nil {
			m.reason += "; could not save recovered state: " + err.Error()
		}
	}
}

func (m *Manager) cleanTemporary() error {
	directory, err := openDirectory(m.root, false)
	if err != nil {
		return err
	}
	entries, err := directory.ReadDir(-1)
	_ = directory.Close()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".staging-") || strings.HasPrefix(entry.Name(), ".state-") {
			if err := removeTree(filepath.Join(m.root, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
