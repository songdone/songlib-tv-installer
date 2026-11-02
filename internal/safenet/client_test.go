package safenet

import "testing"

func TestValidate(t *testing.T) {
	for _, u := range []string{"http://github.com/a", "https://127.0.0.1/a", "https://evil.test/a", "https://github.com@evil.test/a", "https://github.com:443/a"} {
		if Validate(u) == nil {
			t.Fatal(u)
		}
	}
	if Validate("https://github.com/songdone/a") != nil {
		t.Fatal("valid host")
	}
}
