package web

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/bitxeno/atvloadly/internal/app/build"
	"github.com/bitxeno/atvloadly/internal/service"
	"github.com/bitxeno/atvloadly/internal/webupdate"
	"github.com/gofiber/fiber/v2"
)

func loginManagementTestSession(t *testing.T, app *fiber.App, password string) (*http.Cookie, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://localhost:15533/management/login", strings.NewReader(`{"password":"`+password+`"}`))
	req.Header.Set("Origin", "http://localhost:15533")
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("login request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d, want 200", resp.StatusCode)
	}
	var payload map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode login payload: %v", err)
	}
	cookies := resp.Cookies()
	if len(cookies) == 0 {
		t.Fatal("missing session cookie")
	}
	return cookies[0], payload["csrf"]
}

func TestManagementPasswordChangeAndSessionRotation(t *testing.T) {
	keysDir := t.TempDir()
	t.Setenv("SONGLIB_ADMIN_PASSWORD", "testing-only-password-1")
	auth, err := loadManagementAuth(keysDir)
	if err != nil {
		t.Fatal(err)
	}
	server := fiber.New()
	auth.mount(server)

	cookie1, csrf1 := loginManagementTestSession(t, server, "testing-only-password-1")
	cookieOther, _ := loginManagementTestSession(t, server, "testing-only-password-1")

	// Wrong current password rejected with 401.
	req := httptest.NewRequest(http.MethodPost, "http://localhost:15533/management/password", strings.NewReader(`{"current_password":"testing-only-wrong","new_password":"testing-only-password-2"}`))
	req.AddCookie(cookie1)
	req.Header.Set("Origin", "http://localhost:15533")
	req.Header.Set("X-CSRF-Token", csrf1)
	resp, err := server.Test(req, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("wrong current password status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Too-short new password rejected with 400.
	req = httptest.NewRequest(http.MethodPost, "http://localhost:15533/management/password", strings.NewReader(`{"current_password":"testing-only-password-1","new_password":"short"}`))
	req.AddCookie(cookie1)
	req.Header.Set("Origin", "http://localhost:15533")
	req.Header.Set("X-CSRF-Token", csrf1)
	resp, err = server.Test(req, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("short new password status = %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// Valid password change rotates session and invalidates other sessions.
	req = httptest.NewRequest(http.MethodPost, "http://localhost:15533/management/password", strings.NewReader(`{"current_password":"testing-only-password-1","new_password":"testing-only-password-2"}`))
	req.AddCookie(cookie1)
	req.Header.Set("Origin", "http://localhost:15533")
	req.Header.Set("X-CSRF-Token", csrf1)
	resp, err = server.Test(req, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("change password status = %d, want 200", resp.StatusCode)
	}
	rotatedCookies := resp.Cookies()
	_ = resp.Body.Close()
	if len(rotatedCookies) == 0 {
		t.Fatal("expected rotated session cookie")
	}

	// Old second session must now be invalidated.
	checkOld := httptest.NewRequest(http.MethodGet, "http://localhost:15533/management/session", nil)
	checkOld.AddCookie(cookieOther)
	rOld, err := server.Test(checkOld)
	if err != nil {
		t.Fatal(err)
	}
	if rOld.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old session status = %d, want 401", rOld.StatusCode)
	}
	_ = rOld.Body.Close()

	// Reload from disk verifies persistence of new password hash.
	reloaded, err := loadManagementAuth(keysDir)
	if err != nil {
		t.Fatal(err)
	}
	if !managementPasswordMatches(reloaded.hash, "testing-only-password-2") {
		t.Fatal("persisted hash does not match new password")
	}
}

func TestWebUpdateEndToEndWithAuthMutexAndRecovery(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	const keyID = "songlib-test-key"
	backendID := build.BackendID()

	snapFiles := map[string][]byte{
		"index.html":    []byte(`<!doctype html><html><head><title>hot-v1</title></head><body>hot-v1</body></html>`),
		"assets/app.js": []byte(`export const version = "hot-v1";`),
	}
	manifest := webupdate.Manifest{
		Schema:            1,
		Version:           "1.1.0-ui.1",
		Sequence:          1,
		MinBackendVersion: "1.1.0",
		BackendID:         backendID,
		Notes:             "Hot UI update",
	}
	for p, content := range snapFiles {
		sum := sha256.Sum256(content)
		manifest.Files = append(manifest.Files, webupdate.File{
			Path:   p,
			Size:   int64(len(content)),
			SHA256: hex.EncodeToString(sum[:]),
		})
	}
	payloadBytes, _ := json.Marshal(manifest)
	envBytes, _ := json.Marshal(webupdate.Envelope{
		KeyID:     keyID,
		Payload:   payloadBytes,
		Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(priv, payloadBytes)),
	})
	revSum := sha256.Sum256(payloadBytes)
	rev1 := hex.EncodeToString(revSum[:])

	feedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/feed/manifest.json" {
			_, _ = w.Write(envBytes)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/feed/files/") {
			rel := strings.TrimPrefix(r.URL.Path, "/feed/files/")
			if b, ok := snapFiles[rel]; ok {
				_, _ = w.Write(b)
				return
			}
		}
		http.NotFound(w, r)
	}))
	defer feedServer.Close()

	dataDir := t.TempDir()
	t.Setenv("SONGLIB_ADMIN_PASSWORD", "songlib-test-pass")
	t.Setenv("SONGLIB_WEB_UPDATE_FEED", feedServer.URL+"/feed/manifest.json")
	t.Setenv("SONGLIB_WEB_UPDATE_KEY_ID", keyID)
	t.Setenv("SONGLIB_WEB_UPDATE_PUBLIC_KEY", base64.StdEncoding.EncodeToString(pub))
	t.Setenv("SONGLIB_WEB_UPDATE_ALLOW_HTTP", "1")

	packaged := fstest.MapFS{
		"index.html":       {Data: []byte(`<!doctype html><html><head><title>packaged</title></head><body>packaged</body></html>`)},
		"assets/app.js":    {Data: []byte(`console.log("packaged");`)},
		"songlib-logo.svg": {Data: []byte(`<svg></svg>`)},
		"songlib-mark.png": {Data: []byte("png")},
	}

	auth, err := loadManagementAuth(filepath.Join(dataDir, "keys"))
	if err != nil {
		t.Fatal(err)
	}
	ctrl := newWebUpdateController(dataDir, packaged)
	if ctrl.manager == nil {
		t.Fatalf("webUpdateController manager nil: %s", ctrl.initErr)
	}
	taskBusy := false
	ctrl.hasActiveTasks = func() bool { return taskBusy }

	server := fiber.New()
	auth.mount(server)
	ctrl.mountRecoveryAndAssets(server)
	api := server.Group("/api")
	ctrl.registerAPI(api)
	server.Use("/", ctrl.frontendMiddleware())

	// Pre-auth login asset works without session cookie.
	rLogo, err := server.Test(httptest.NewRequest(http.MethodGet, "http://localhost:15533/songlib-logo.svg", nil))
	if err != nil || rLogo.StatusCode != http.StatusOK {
		t.Fatalf("pre-auth logo status = %v err=%v", rLogo, err)
	}
	_ = rLogo.Body.Close()

	cookie, csrf := loginManagementTestSession(t, server, "songlib-test-pass")

	// 1. Check update
	req := httptest.NewRequest(http.MethodPost, "http://localhost:15533/api/web-update/check", strings.NewReader(`{}`))
	req.AddCookie(cookie)
	req.Header.Set("Origin", "http://localhost:15533")
	req.Header.Set("X-CSRF-Token", csrf)
	req.Header.Set("Content-Type", "application/json")
	resp, err := server.Test(req, 5000)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("check failed: status=%d err=%v", resp.StatusCode, err)
	}
	_ = resp.Body.Close()

	// 2. While an interactive session is active, Apply must be blocked with 409 Conflict!
	releaseSession, ok := service.BeginInteractiveSession()
	if !ok {
		t.Fatal("failed to start interactive session")
	}
	applyBody := `{"revision":"` + rev1 + `","expected_active_revision":""}`
	reqBusy := httptest.NewRequest(http.MethodPost, "http://localhost:15533/api/web-update/apply", strings.NewReader(applyBody))
	reqBusy.AddCookie(cookie)
	reqBusy.Header.Set("Origin", "http://localhost:15533")
	reqBusy.Header.Set("X-CSRF-Token", csrf)
	reqBusy.Header.Set("Content-Type", "application/json")
	respBusy, err := server.Test(reqBusy, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if respBusy.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict while interactive session active, got %d", respBusy.StatusCode)
	}
	_ = respBusy.Body.Close()
	releaseSession()

	// 2b. While a background task is active, Apply must also be blocked with 409 Conflict!
	taskBusy = true
	reqTaskBusy := httptest.NewRequest(http.MethodPost, "http://localhost:15533/api/web-update/apply", strings.NewReader(applyBody))
	reqTaskBusy.AddCookie(cookie)
	reqTaskBusy.Header.Set("Origin", "http://localhost:15533")
	reqTaskBusy.Header.Set("X-CSRF-Token", csrf)
	reqTaskBusy.Header.Set("Content-Type", "application/json")
	respTaskBusy, err := server.Test(reqTaskBusy, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if respTaskBusy.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict while background task active, got %d", respTaskBusy.StatusCode)
	}
	_ = respTaskBusy.Body.Close()
	taskBusy = false

	// 3. Apply succeeds once idle.
	reqApply := httptest.NewRequest(http.MethodPost, "http://localhost:15533/api/web-update/apply", strings.NewReader(applyBody))
	reqApply.AddCookie(cookie)
	reqApply.Header.Set("Origin", "http://localhost:15533")
	reqApply.Header.Set("X-CSRF-Token", csrf)
	reqApply.Header.Set("Content-Type", "application/json")
	respApply, err := server.Test(reqApply, 5000)
	if err != nil || respApply.StatusCode != http.StatusOK {
		t.Fatalf("apply failed: status=%d err=%v", respApply.StatusCode, err)
	}
	_ = respApply.Body.Close()

	// 4. Root "/" serves injected base href for rev1 and no-store header.
	reqRoot := httptest.NewRequest(http.MethodGet, "http://localhost:15533/", nil)
	reqRoot.AddCookie(cookie)
	respRoot, err := server.Test(reqRoot)
	if err != nil || respRoot.StatusCode != http.StatusOK {
		t.Fatalf("GET / failed: %v", err)
	}
	bodyBytes, _ := io.ReadAll(respRoot.Body)
	_ = respRoot.Body.Close()
	if !strings.Contains(string(bodyBytes), `<base href="/_web/`+rev1+`/" />`) || !strings.Contains(string(bodyBytes), "hot-v1") {
		t.Fatalf("unexpected injected index HTML: %s", string(bodyBytes))
	}
	if got := respRoot.Header.Get(fiber.HeaderCacheControl); got != documentCacheControl {
		t.Fatalf("Cache-Control = %q, want %q", got, documentCacheControl)
	}

	// 5. Immutable asset served under /_web/<rev1>/assets/app.js; missing asset returns 404.
	reqAsset := httptest.NewRequest(http.MethodGet, "http://localhost:15533/_web/"+rev1+"/assets/app.js", nil)
	reqAsset.AddCookie(cookie)
	respAsset, err := server.Test(reqAsset)
	if err != nil || respAsset.StatusCode != http.StatusOK {
		t.Fatalf("GET immutable asset failed: %v", err)
	}
	assetBytes, _ := io.ReadAll(respAsset.Body)
	_ = respAsset.Body.Close()
	if !strings.Contains(string(assetBytes), "hot-v1") {
		t.Fatalf("unexpected asset content: %s", string(assetBytes))
	}

	reqMissing := httptest.NewRequest(http.MethodGet, "http://localhost:15533/_web/"+rev1+"/assets/missing.js", nil)
	reqMissing.AddCookie(cookie)
	respMissing, _ := server.Test(reqMissing)
	if respMissing.StatusCode != http.StatusNotFound {
		t.Fatalf("missing JS status = %d, want 404", respMissing.StatusCode)
	}
	_ = respMissing.Body.Close()

	// 6. Runtime corruption of active index.html returns 503 with recovery link.
	resolvedDir, _ := filepath.EvalSymlinks(dataDir)
	indexOnDisk := filepath.Join(resolvedDir, "web-updates", "versions", rev1, "index.html")
	if err := os.WriteFile(indexOnDisk, []byte("corrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	reqCorrupt := httptest.NewRequest(http.MethodGet, "http://localhost:15533/", nil)
	reqCorrupt.AddCookie(cookie)
	respCorrupt, err := server.Test(reqCorrupt)
	if err != nil {
		t.Fatal(err)
	}
	corruptBody, _ := io.ReadAll(respCorrupt.Body)
	_ = respCorrupt.Body.Close()
	if respCorrupt.StatusCode != http.StatusServiceUnavailable || !strings.Contains(string(corruptBody), "/management/web-update/recovery") {
		t.Fatalf("expected 503 with recovery link, got %d: %s", respCorrupt.StatusCode, string(corruptBody))
	}

	// 7. Standalone recovery page restores packaged baseline via form POST.
	reqRecGet := httptest.NewRequest(http.MethodGet, "http://localhost:15533/management/web-update/recovery", nil)
	reqRecGet.AddCookie(cookie)
	respRecGet, err := server.Test(reqRecGet)
	if err != nil || respRecGet.StatusCode != http.StatusOK {
		t.Fatalf("GET recovery failed: %v", err)
	}
	if got := respRecGet.Header.Get("Referrer-Policy"); got != "same-origin" {
		t.Fatalf("recovery Referrer-Policy = %q, want same-origin", got)
	}
	_ = respRecGet.Body.Close()

	form := url.Values{
		"csrf":     {csrf},
		"action":   {"restore"},
		"expected": {rev1},
	}
	reqRecPost := httptest.NewRequest(http.MethodPost, "http://localhost:15533/management/web-update/recovery", strings.NewReader(form.Encode()))
	reqRecPost.AddCookie(cookie)
	reqRecPost.Header.Set("Origin", "http://localhost:15533")
	reqRecPost.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	respRecPost, err := server.Test(reqRecPost, 5000)
	if err != nil || respRecPost.StatusCode != http.StatusSeeOther {
		t.Fatalf("POST recovery restore status = %d err=%v, want 303", respRecPost.StatusCode, err)
	}
	_ = respRecPost.Body.Close()

	// Root "/" now serves packaged baseline again.
	reqAfter := httptest.NewRequest(http.MethodGet, "http://localhost:15533/", nil)
	reqAfter.AddCookie(cookie)
	respAfter, err := server.Test(reqAfter)
	if err != nil || respAfter.StatusCode != http.StatusOK {
		t.Fatalf("GET / after restore failed: %v", err)
	}
	afterBody, _ := io.ReadAll(respAfter.Body)
	_ = respAfter.Body.Close()
	if !strings.Contains(string(afterBody), "packaged") {
		t.Fatalf("expected packaged index after restore, got: %s", string(afterBody))
	}
}
