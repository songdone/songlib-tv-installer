package web

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
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
	r, _ := server.Test(httptest.NewRequest("GET", "http://public.example.test/management/login", nil))
	if r.StatusCode != 403 {
		t.Fatal(r.StatusCode)
	}
}
