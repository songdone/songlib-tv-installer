package web

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestManagementBoundary(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SONGLIB_ADMIN_PASSWORD", "testing-only-password-123")
	auth, err := loadManagementAuth(dir)
	if err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(dir + "/management-password.hash")
	if info.Mode().Perm() != 0600 {
		t.Fatal("unsafe file permissions")
	}
	app := fiber.New()
	auth.mount(app)
	app.Get("/secret", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	app.Post("/action", func(c *fiber.Ctx) error { return c.SendStatus(200) })
	req := httptest.NewRequest("GET", "http://localhost:18533/secret", nil)
	r, _ := app.Test(req)
	if r.StatusCode != 401 {
		t.Fatal(r.StatusCode)
	}
	req = httptest.NewRequest("POST", "http://localhost:18533/management/login", strings.NewReader(`{"password":"testing-only-password-123"}`))
	req.Header.Set("Origin", "http://evil.test")
	r, _ = app.Test(req)
	if r.StatusCode != 403 {
		t.Fatal(r.StatusCode)
	}
	req = httptest.NewRequest("POST", "http://localhost:18533/management/login", strings.NewReader(`{"password":"testing-only-password-123"}`))
	req.Header.Set("Origin", "http://localhost:18533")
	r, _ = app.Test(req, 5000)
	if r.StatusCode != 200 {
		t.Fatal(r.StatusCode)
	}
	cookies := r.Cookies()
	b, _ := io.ReadAll(r.Body)
	var result map[string]string
	_ = json.Unmarshal(b, &result)
	for _, tc := range []struct {
		origin, csrf string
		status       int
	}{{"http://evil.test", result["csrf"], 403}, {"http://localhost:18533", "", 403}, {"http://localhost:18533", result["csrf"], 200}} {
		req = httptest.NewRequest("POST", "http://localhost:18533/action", nil)
		req.AddCookie(cookies[0])
		req.Header.Set("Origin", tc.origin)
		req.Header.Set("X-CSRF-Token", tc.csrf)
		r, _ = app.Test(req)
		if r.StatusCode != tc.status {
			t.Fatal(r.StatusCode, tc.status)
		}
	}
}
func TestManagementRejectsPublicHostAndMissingPassword(t *testing.T) {
	t.Setenv("SONGLIB_ADMIN_PASSWORD", "")
	if _, err := loadManagementAuth(t.TempDir()); err == nil {
		t.Fatal("missing password accepted")
	}
	t.Setenv("SONGLIB_ADMIN_PASSWORD", "testing-only-password-123")
	a, _ := loadManagementAuth(t.TempDir())
	server := fiber.New()
	a.mount(server)

	// No allowlist configured: unrestricted, like upstream - the login page
	// itself is reachable from any host (the password is the boundary).
	r, _ := server.Test(httptest.NewRequest("GET", "http://public.example.test/management/login", nil))
	if r.StatusCode != 200 {
		t.Fatal(r.StatusCode)
	}

	// With an allowlist configured: only the listed hosts (plus private and
	// loopback) may reach the app - the DNS-rebinding hardening is opt-in.
	t.Setenv("SONGLIB_ALLOWED_HOSTS", "other.example.test")
	r, _ = server.Test(httptest.NewRequest("GET", "http://public.example.test/management/login", nil))
	if r.StatusCode != 403 {
		t.Fatal(r.StatusCode)
	}
	r, _ = server.Test(httptest.NewRequest("GET", "http://other.example.test/management/login", nil))
	if r.StatusCode != 200 {
		t.Fatal(r.StatusCode)
	}
}
func TestManagementShortSimpleUnicodeAndInitialPasswordFile(t *testing.T) {
	for _, password := range []string{"12345678", "pass#$!8", "音乐小岛1234", strings.Repeat("音", 64)} {
		t.Run(password[:8], func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("SONGLIB_ADMIN_PASSWORD", "")
			if err := os.WriteFile(dir+"/initial-password", []byte(password), 0600); err != nil {
				t.Fatal(err)
			}
			auth, err := loadManagementAuth(dir)
			if err != nil {
				t.Fatal(err)
			}
			if !managementPasswordMatches(auth.hash, password) {
				t.Fatal("password was altered")
			}
			if _, err := os.Stat(dir + "/initial-password"); !os.IsNotExist(err) {
				t.Fatal("initial password was not removed")
			}
		})
	}
	for _, password := range []string{"1234567", strings.Repeat("a", 65), "12345678\n"} {
		if validManagementPassword(password) {
			t.Fatal("invalid password accepted")
		}
	}
}
func TestExistingManagementHashIsPreserved(t *testing.T) {
	dir := t.TempDir()
	password := "old-password-123456"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(dir+"/management-password.hash", hash, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SONGLIB_ADMIN_PASSWORD", "12345678")
	auth, err := loadManagementAuth(dir)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := os.ReadFile(dir + "/management-password.hash")
	if err != nil {
		t.Fatal(err)
	}
	if string(stored) != string(hash) || !managementPasswordMatches(auth.hash, password) {
		t.Fatal("existing password was reset")
	}
}
