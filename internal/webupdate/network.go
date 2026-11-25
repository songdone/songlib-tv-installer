package webupdate

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

func parseFeed(raw string, allowHTTP bool) (*url.URL, error) {
	if strings.ContainsAny(raw, "?#") {
		return nil, errors.New("web update feed URL must not contain query or fragment")
	}
	feed, err := url.Parse(raw)
	if err != nil || feed == nil {
		return nil, errors.New("invalid web update feed URL")
	}
	if feed.User != nil || feed.Opaque != "" || feed.Hostname() == "" || feed.RawQuery != "" || feed.ForceQuery || feed.Fragment != "" || feed.RawFragment != "" {
		return nil, errors.New("web update feed URL must not contain credentials, query, or fragment")
	}
	if feed.Scheme != "https" && !(feed.Scheme == "http" && allowHTTP && loopbackHost(feed.Hostname())) {
		return nil, errors.New("web update feed requires HTTPS (HTTP is limited to loopback development)")
	}
	if feed.Path == "" || strings.HasSuffix(feed.Path, "/") || path.Clean(feed.Path) != feed.Path || strings.ContainsAny(feed.Path, "\\\x00") || strings.Contains(feed.Path, "%") {
		return nil, errors.New("web update feed URL must name a manifest in a safe directory")
	}
	for _, part := range strings.Split(feed.Path, "/") {
		if part == "." || part == ".." {
			return nil, errors.New("unsafe web update feed directory")
		}
	}
	return feed, nil
}

func loopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func updateClient(configured *http.Client) *http.Client {
	client := http.Client{Timeout: 45 * time.Second}
	if configured != nil {
		client = *configured
		if client.Timeout == 0 {
			client.Timeout = 45 * time.Second
		}
	}
	// Do not follow even same-origin redirects: only the configured manifest
	// and its sibling files/ subtree are trusted.
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errors.New("web update redirects are forbidden")
	}
	client.Jar = nil
	return &client
}

func (m *Manager) fileURL(name string) string {
	resource := *m.feed
	resource.Path = path.Join(path.Dir(m.feed.Path), "files", name)
	resource.RawPath = ""
	return resource.String()
}

func (m *Manager) get(ctx context.Context, resource string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, resource, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("Cache-Control", "no-cache")
	request.Header.Set("User-Agent", "SongLib-WebUpdate/1")
	response, err := m.client.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("web update resource returned HTTP %d", response.StatusCode)
	}
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		_ = response.Body.Close()
		return nil, errors.New("encoded web update responses are forbidden")
	}
	return response, nil
}

func (m *Manager) fetch(ctx context.Context) ([]byte, Manifest, string, error) {
	response, err := m.get(ctx, m.feed.String())
	if err != nil {
		return nil, Manifest{}, "", err
	}
	defer func() { _ = response.Body.Close() }()
	if response.ContentLength > MaxManifestSize {
		return nil, Manifest{}, "", errors.New("signed manifest exceeds size limit")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, MaxManifestSize+1))
	if err != nil {
		return nil, Manifest{}, "", err
	}
	if err := ctx.Err(); err != nil {
		return nil, Manifest{}, "", err
	}
	manifest, err := VerifyEnvelope(raw, m.config.KeyID, ed25519.PublicKey(m.config.PublicKey))
	if err != nil {
		return nil, Manifest{}, "", err
	}
	return raw, manifest, payloadRevision(raw), nil
}
