// SongLib fnOS hardening: HTTPS allowlist, public-IP dial pinning, redirect validation.
package safenet

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

func Enabled() bool { return os.Getenv("SONGLIB_HARDENED") == "1" }
func Validate(raw string) error {
	u, e := url.Parse(raw)
	if e != nil {
		return e
	}
	if u.Scheme != "https" || u.User != nil || u.Port() != "" {
		return fmt.Errorf("only trusted HTTPS download sources are allowed")
	}
	switch u.Hostname() {
	case "songlib.playsong.cn", "github.com", "api.github.com", "raw.githubusercontent.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return nil
	}
	return fmt.Errorf("download host is not allowlisted")
}
func Transport() *http.Transport {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, e := net.SplitHostPort(addr)
		if e != nil {
			return nil, e
		}
		ips, e := net.DefaultResolver.LookupIPAddr(ctx, host)
		if e != nil {
			return nil, e
		}
		for _, ip := range ips {
			if !ip.IP.IsGlobalUnicast() || ip.IP.IsPrivate() || ip.IP.IsLoopback() || ip.IP.IsLinkLocalUnicast() {
				return nil, fmt.Errorf("private or non-public download address rejected")
			}
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no public address")
		}
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}
	return t
}
func Client(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: &guardedTransport{Transport()}, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return fmt.Errorf("too many redirects")
		}
		return Validate(req.URL.String())
	}}
}

type guardedTransport struct{ base http.RoundTripper }

func (t *guardedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if e := Validate(r.URL.String()); e != nil {
		return nil, e
	}
	return t.base.RoundTrip(r)
}
