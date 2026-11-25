package build

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
)

var (
	/*********Will auto update by ci build *********/
	Version   = "1.2.0"
	Commit    = "unknown"
	BuildDate = "unknown"

	Mode = "development"

	// Optional compile-time public configuration for signed web hot updates.
	// All three remain empty by default so builds without an approved release
	// feed stay cleanly offline. Private signing keys never enter this binary.
	WebUpdateFeed      = ""
	WebUpdateKeyID     = ""
	WebUpdatePublicKey = ""
	/*********Will auto update by ci build *********/
)

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
