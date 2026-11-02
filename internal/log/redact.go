package log

import (
	"os"
	"regexp"
)

var redactions = []*regexp.Regexp{
	regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`),
	regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
	regexp.MustCompile(`SL-[A-Z2-7]{20,}`),
	regexp.MustCompile(`(?i)(?:password|passwd|token|secret|cookie|authorization|otp)["'\s:=]+[^\s,;]+`),
}

func Redact(s string) string {
	if os.Getenv("SONGLIB_HARDENED") != "1" {
		return s
	}
	for _, re := range redactions {
		s = re.ReplaceAllString(s, "[redacted]")
	}
	return s
}
