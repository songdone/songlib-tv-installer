package build

import "testing"

func TestDefaultWebUpdateConfig(t *testing.T) {
	origVer, origFeed, origKeyID, origPub := Version, WebUpdateFeed, WebUpdateKeyID, WebUpdatePublicKey
	t.Cleanup(func() {
		Version, WebUpdateFeed, WebUpdateKeyID, WebUpdatePublicKey = origVer, origFeed, origKeyID, origPub
	})

	WebUpdateFeed, WebUpdateKeyID, WebUpdatePublicKey = "", "", ""

	Version = "1.2.0"
	feed, keyID, pub := DefaultWebUpdateConfig()
	if feed != "https://update.6uos.de/tv-installer/v1.2.0/manifest.json" {
		t.Fatalf("feed for 1.2.0 = %q", feed)
	}
	if keyID != DefaultWebUpdateKeyID || pub != DefaultWebUpdatePublicKey {
		t.Fatalf("unexpected default key material: %q / %q", keyID, pub)
	}

	Version = "v1.2.1"
	feed, _, _ = DefaultWebUpdateConfig()
	if feed != "https://update.6uos.de/tv-installer/v1.2.1/manifest.json" {
		t.Fatalf("feed for v1.2.1 = %q", feed)
	}

	Version = "dev"
	feed, keyID, pub = DefaultWebUpdateConfig()
	if feed != "" || keyID != "" || pub != "" {
		t.Fatalf("dev version should stay unconfigured, got %q %q %q", feed, keyID, pub)
	}

	WebUpdateFeed = "https://mirror.example.com/custom/manifest.json"
	feed, keyID, pub = DefaultWebUpdateConfig()
	if feed != "https://mirror.example.com/custom/manifest.json" || keyID != DefaultWebUpdateKeyID || pub != DefaultWebUpdatePublicKey {
		t.Fatalf("override feed unexpected: %q %q %q", feed, keyID, pub)
	}
}
