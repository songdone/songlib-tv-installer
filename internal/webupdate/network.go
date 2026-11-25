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

const (
	downloadMaxAttempts = 3
	downloadConcurrency = 6
)

var retryBackoffs = []time.Duration{
	250 * time.Millisecond,
	750 * time.Millisecond,
}

type httpStatusError struct {
	statusCode int
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("web update resource returned HTTP %d", e.statusCode)
}

func (e *httpStatusError) retryable() bool {
	switch e.statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests,
		http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}

type nonRetryableError struct {
	err error
}

func (e *nonRetryableError) Error() string { return e.err.Error() }
func (e *nonRetryableError) Unwrap() error { return e.err }

var errRedirectForbidden = errors.New("web update redirects are forbidden")

func shouldRetry(ctx context.Context, err error) bool {
	if err == nil || ctx.Err() != nil {
		return false
	}
	var permanent *nonRetryableError
	if errors.As(err, &permanent) {
		return false
	}
	if errors.Is(err, errRedirectForbidden) {
		return false
	}
	var statusErr *httpStatusError
	if errors.As(err, &statusErr) {
		return statusErr.retryable()
	}
	return true
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func updateClient(configured *http.Client) *http.Client {
	var client http.Client
	if configured != nil {
		client = *configured
		if client.Timeout == 0 {
			client.Timeout = 30 * time.Second
		}
	} else {
		client = http.Client{Timeout: 30 * time.Second}
		if base, ok := http.DefaultTransport.(*http.Transport); ok {
			tr := base.Clone()
			tr.MaxConnsPerHost = 8
			tr.MaxIdleConnsPerHost = 8
			tr.ForceAttemptHTTP2 = true
			client.Transport = tr
		}
	}
	// Do not follow even same-origin redirects: only the configured manifest
	// and its sibling files/ subtree are trusted.
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return errRedirectForbidden
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
		return nil, &nonRetryableError{err: err}
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
		return nil, &httpStatusError{statusCode: response.StatusCode}
	}
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && !strings.EqualFold(encoding, "identity") {
		_ = response.Body.Close()
		return nil, &nonRetryableError{err: errors.New("encoded web update responses are forbidden")}
	}
	return response, nil
}

func (m *Manager) readURL(ctx context.Context, resource string, maxSize int64, limitErr error) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < downloadMaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if attempt > 0 {
			delay := retryBackoffs[len(retryBackoffs)-1]
			if attempt-1 < len(retryBackoffs) {
				delay = retryBackoffs[attempt-1]
			}
			if err := waitRetry(ctx, delay); err != nil {
				return nil, err
			}
		}
		response, err := m.get(ctx, resource)
		if err != nil {
			lastErr = err
			if !shouldRetry(ctx, err) {
				return nil, err
			}
			continue
		}
		if response.ContentLength > maxSize {
			_ = response.Body.Close()
			return nil, &nonRetryableError{err: limitErr}
		}
		data, readErr := io.ReadAll(io.LimitReader(response.Body, maxSize+1))
		_ = response.Body.Close()
		if readErr != nil {
			lastErr = readErr
			if !shouldRetry(ctx, readErr) {
				return nil, readErr
			}
			continue
		}
		if int64(len(data)) > maxSize {
			return nil, &nonRetryableError{err: limitErr}
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return data, nil
	}
	return nil, lastErr
}

func (m *Manager) fetch(ctx context.Context) ([]byte, Manifest, string, error) {
	raw, err := m.readURL(ctx, m.feed.String(), MaxManifestSize, errors.New("signed manifest exceeds size limit"))
	if err != nil {
		return nil, Manifest{}, "", err
	}
	manifest, err := VerifyEnvelope(raw, m.config.KeyID, ed25519.PublicKey(m.config.PublicKey))
	if err != nil {
		return nil, Manifest{}, "", err
	}
	return raw, manifest, payloadRevision(raw), nil
}
