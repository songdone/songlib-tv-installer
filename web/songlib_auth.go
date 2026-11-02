// SongLib modification 2026-11-02: authenticated management boundary.
package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type managementSession struct {
	csrf    string
	expires time.Time
}
type attempt struct {
	count int
	until time.Time
}
type managementAuth struct {
	hash     []byte
	mu       sync.Mutex
	sessions map[[32]byte]managementSession
	attempts map[string]attempt
}

func randomToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func loadManagementAuth(dataDir string) (*managementAuth, error) {
	path := filepath.Join(dataDir, "management-password.hash")
	hash, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		password := os.Getenv("SONGLIB_ADMIN_PASSWORD")
		if len(password) < 16 {
			return nil, fmt.Errorf("set a management password of at least 16 characters during first setup")
		}
		hash, err = bcrypt.GenerateFromPassword([]byte(password), 12)
		if err != nil {
			return nil, err
		}
		if err = os.MkdirAll(dataDir, 0700); err != nil {
			return nil, err
		}
		if err = os.WriteFile(path, hash, 0600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if _, err = bcrypt.Cost(hash); err != nil {
		return nil, fmt.Errorf("invalid management password file")
	}
	_ = os.Chmod(path, 0600)
	return &managementAuth{hash: hash, sessions: map[[32]byte]managementSession{}, attempts: map[string]attempt{}}, nil
}
func sameOrigin(c *fiber.Ctx) bool {
	return c.Get("Origin") != "" && c.Get("Origin") == c.Protocol()+"://"+string(c.Request().Host())
}
func allowedHost(c *fiber.Ctx) bool {
	host := c.Hostname()
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
		return true
	}
	for _, h := range strings.Split(os.Getenv("SONGLIB_ALLOWED_HOSTS"), ",") {
		if h != "" && h == host {
			return true
		}
	}
	return false
}
func (a *managementAuth) mount(server *fiber.App) {
	server.Get("/ready", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "running", "app_expiry": "check_authenticated_dashboard"})
	})
	server.Use(func(c *fiber.Ctx) error {
		c.Set("Cache-Control", "no-store")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("X-Frame-Options", "DENY")
		if !allowedHost(c) {
			return c.SendStatus(403)
		}
		if c.Path() == "/management/login" {
			return c.Next()
		}
		token := sha256.Sum256([]byte(c.Cookies("sl_management")))
		a.mu.Lock()
		session, ok := a.sessions[token]
		if ok && time.Now().After(session.expires) {
			delete(a.sessions, token)
			ok = false
		}
		a.mu.Unlock()
		if !ok {
			if c.Method() == "GET" && strings.Contains(c.Get("Accept"), "text/html") {
				return c.Redirect("/management/login", 303)
			}
			return c.SendStatus(401)
		}
		if strings.EqualFold(c.Get("Upgrade"), "websocket") && !sameOrigin(c) {
			return c.SendStatus(403)
		}
		if c.Method() != "GET" && c.Method() != "HEAD" {
			if !sameOrigin(c) || subtle.ConstantTimeCompare([]byte(c.Get("X-CSRF-Token")), []byte(session.csrf)) != 1 {
				return c.SendStatus(403)
			}
		}
		c.Locals("management_csrf", session.csrf)
		return c.Next()
	})
	server.Get("/management/login", func(c *fiber.Ctx) error { c.Type("html"); return c.SendString(managementLoginHTML) })
	server.Post("/management/login", func(c *fiber.Ctx) error {
		if !sameOrigin(c) {
			return c.SendStatus(403)
		}
		a.mu.Lock()
		at := a.attempts[c.IP()]
		if time.Now().After(at.until) {
			at = attempt{until: time.Now().Add(10 * time.Minute)}
		}
		at.count++
		if len(a.attempts) > 4096 {
			for ip, v := range a.attempts {
				if time.Now().After(v.until) {
					delete(a.attempts, ip)
				}
			}
		}
		a.attempts[c.IP()] = at
		a.mu.Unlock()
		if at.count > 10 {
			return c.SendStatus(429)
		}
		var body struct {
			Password string `json:"password"`
		}
		if len(c.Body()) > 1024 || json.Unmarshal(c.Body(), &body) != nil {
			return c.SendStatus(400)
		}
		if bcrypt.CompareHashAndPassword(a.hash, []byte(body.Password)) != nil {
			return c.SendStatus(401)
		}
		token, csrf := randomToken(), randomToken()
		a.mu.Lock()
		for k, s := range a.sessions {
			if time.Now().After(s.expires) {
				delete(a.sessions, k)
			}
		}
		a.sessions[sha256.Sum256([]byte(token))] = managementSession{csrf, time.Now().Add(time.Hour)}
		a.mu.Unlock()
		c.Cookie(&fiber.Cookie{Name: "sl_management", Value: token, HTTPOnly: true, Secure: c.Protocol() == "https", SameSite: "Strict", Path: "/", MaxAge: 3600})
		return c.JSON(fiber.Map{"csrf": csrf})
	})
	server.Get("/management/session", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"csrf": c.Locals("management_csrf")}) })
	server.Post("/management/logout", func(c *fiber.Ctx) error {
		a.mu.Lock()
		delete(a.sessions, sha256.Sum256([]byte(c.Cookies("sl_management"))))
		a.mu.Unlock()
		c.ClearCookie("sl_management")
		return c.SendStatus(204)
	})
}

const managementLoginHTML = `<!doctype html><html lang="zh-CN"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>音屿-Apple TV音乐播放器安装助手</title><style>body{background:#111615;color:#efece2;font:16px/1.7 system-ui;margin:8vh auto;padding:24px;max-width:520px}h1{font-size:25px}input,button{box-sizing:border-box;width:100%;padding:14px;margin:10px 0;border-radius:8px}button{background:#e2c68c;border:0}p{color:#c0c7c1}</style><h1>音屿-Apple TV音乐播放器安装助手</h1><p>登录 NAS 管理服务。此密码与 Apple 账户、音屿激活码分别管理。</p><form><label>管理密码<input type="password" autocomplete="current-password" required maxlength="200"></label><button>登录</button></form><p role="status" id="status"></p><script>document.querySelector('form').onsubmit=async e=>{e.preventDefault();const r=await fetch('/management/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({password:document.querySelector('input').value})});if(r.ok)location.href='/';else document.querySelector('#status').textContent=r.status===429?'尝试过多，请稍后重试':'登录失败，请检查管理密码';}</script></html>`
