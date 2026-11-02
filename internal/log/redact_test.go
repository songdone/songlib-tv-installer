package log

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	t.Setenv("SONGLIB_HARDENED", "1")
	result := Redact("account user@example.com ip 192.168.1.1 password=example token:private SL-ABCDEFGHIJKLMNOPQRSTUVWXYZ234567")
	for _, s := range []string{"user@example.com", "192.168.1.1", "example token", "private", "SL-ABC"} {
		if strings.Contains(result, s) {
			t.Fatal(result)
		}
	}
}
