package build

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
)

const (
	// Official public signed web hot-update defaults (Cloudflare R2 custom domain).
	// Contains only public metadata and Ed25519 public key — private keys never enter
	// the repository or built binaries.
	DefaultWebUpdateFeedBase  = "https://update.6uos.de/tv-installer"
	DefaultWebUpdateKeyID     = "songlib-web-2026"
	DefaultWebUpdatePublicKey = "AUfJlb2xfRXdbSN2WKiHCU5mxbCkAcDsNOP3IhS8k+w="
)

var (
	/*********Will auto update by ci build *********/
	Version   = "1.2.0"
	Commit    = "unknown"
	BuildDate = "unknown"

	Mode = "development"

	// Optional compile-time overrides for signed web hot updates. When empty,
	// DefaultWebUpdateConfig derives the version-matched official feed and public
	// signing key from DefaultWebUpdateFeedBase / DefaultWebUpdateKeyID /
	// DefaultWebUpdatePublicKey. Private signing keys never enter this binary.
	WebUpdateFeed      = ""
	WebUpdateKeyID     = ""
	WebUpdatePublicKey = ""
	/*********Will auto update by ci build *********/
)

// DefaultWebUpdateConfig returns the effective built-in public feed URL,
// signing key ID, and Base64 Ed25519 public key for the current build version.
// Non-semver development versions return empty strings unless compile-time
// overrides are set.
func DefaultWebUpdateConfig() (feed, keyID, publicKey string) {
	feed = strings.TrimSpace(WebUpdateFeed)
	if feed == "" {
		if tag := normalizedReleaseTag(Version); tag != "" {
			feed = strings.TrimRight(DefaultWebUpdateFeedBase, "/") + "/" + tag + "/manifest.json"
		}
	}
	if feed == "" {
		return "", "", ""
	}
	keyID = strings.TrimSpace(WebUpdateKeyID)
	if keyID == "" {
		keyID = DefaultWebUpdateKeyID
	}
	publicKey = strings.TrimSpace(WebUpdatePublicKey)
	if publicKey == "" {
		publicKey = DefaultWebUpdatePublicKey
	}
	return feed, keyID, publicKey
}

func normalizedReleaseTag(raw string) string {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(strings.TrimPrefix(v, "v"), "V")
	if v == "" {
		return ""
	}
	core := v
	if idx := strings.IndexAny(core, "-+"); idx >= 0 {
		core = core[:idx]
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return ""
	}
	for _, p := range parts {
		if p == "" {
			return ""
		}
		if _, err := strconv.Atoi(p); err != nil {
			return ""
		}
	}
	return "v" + core
}

// BackendID returns the deterministic build identity bound into signed
// frontend snapshots. Changing any of Version, Commit, or BuildDate produces
// a distinct identity so snapshots built for one container/FPK cannot load on
// another backend build.
func BackendID() string {
	sum := sha256.Sum256([]byte(Version + "\x00" + Commit + "\x00" + BuildDate))
	return hex.EncodeToString(sum[:])
}

// DeployMode reports the runtime deployment target ("fnos", "docker", or
// "desktop") so the UI can show accurate core-upgrade instructions.
func DeployMode() string {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("SONGLIB_DEPLOY_MODE"))) {
	case "fnos":
		return "fnos"
	case "docker":
		return "docker"
	case "desktop":
		return "desktop"
	}
	if os.Getenv("TRIM_PKGVAR") != "" || os.Getenv("TRIM_APPDEST") != "" {
		return "fnos"
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return "docker"
	}
	return "standalone"
}
