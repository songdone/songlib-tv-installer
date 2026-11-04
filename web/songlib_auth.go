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
	"unicode/utf8"
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

// Passwords are Unicode text, not shell or dotenv input. Prehashing avoids bcrypt's 72-byte limit.
func validManagementPassword(password string) bool {
	n := utf8.RuneCountInString(password)
	return utf8.ValidString(password) && n >= 8 && n <= 64 && !strings.ContainsAny(password, "\r\n\x00")
}
func managementPasswordMaterial(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}
func managementPasswordMatches(hash []byte, password string) bool {
	if strings.HasPrefix(string(hash), "sha256$") {
		return bcrypt.CompareHashAndPassword(hash[len("sha256$"):], []byte(managementPasswordMaterial(password))) == nil
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}
func loadManagementAuth(dataDir string) (*managementAuth, error) {
	path := filepath.Join(dataDir, "management-password.hash")
	hash, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		password := os.Getenv("SONGLIB_ADMIN_PASSWORD")
		initialPath := filepath.Join(dataDir, "initial-password")
		if initial, e := os.ReadFile(initialPath); e == nil {
			password = string(initial)
		} else if !os.IsNotExist(e) {
			return nil, e
		}
		if !validManagementPassword(password) {
			return nil, fmt.Errorf("请设置8至64位管理密码，不需要混合大小写或特殊符号")
		}
		hash, err = bcrypt.GenerateFromPassword([]byte(managementPasswordMaterial(password)), 12)
		if err != nil {
			return nil, err
		}
		hash = append([]byte("sha256$"), hash...)

		if err = os.MkdirAll(dataDir, 0700); err != nil {
			return nil, err
		}
		if err = os.WriteFile(path, hash, 0600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}
	if err = os.Remove(filepath.Join(dataDir, "initial-password")); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if _, err = bcrypt.Cost([]byte(strings.TrimPrefix(string(hash), "sha256$"))); err != nil {
		return nil, fmt.Errorf("管理密码文件无效，请从备份恢复")
	}
	_ = os.Chmod(path, 0600)
	return &managementAuth{hash: hash, sessions: map[[32]byte]managementSession{}, attempts: map[string]attempt{}}, nil
}
// requestProtocol resolves the effective scheme. Behind an owner-deployed
// reverse proxy (frp tunnel / LAN gateway — the direct peer is always a
// private or loopback address on the paths this boundary allows), the
// X-Forwarded-Proto header is the only signal that the browser used HTTPS;
// ignoring it breaks same-origin for the login POST and Secure cookies.
func requestProtocol(c *fiber.Ctx) string {
	if p := c.Get("X-Forwarded-Proto"); p == "https" || p == "http" {
		if ip := net.ParseIP(c.IP()); ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
			return p
		}
	}
	return c.Protocol()
}
func sameOrigin(c *fiber.Ctx) bool {
	return c.Get("Origin") != "" && c.Get("Origin") == requestProtocol(c)+"://"+string(c.Request().Host())
}
func allowedHost(c *fiber.Ctx) bool {
	host := string(c.Request().Host())
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	if ip != nil && (ip.IsPrivate() || ip.IsLoopback()) {
		return true
	}
	// No allowlist configured: unrestricted like upstream. The management
	// password (session + CSRF) is the real boundary; the allowlist is an
	// opt-in hardening against DNS rebinding for users who want it.
	for _, h := range strings.Split(os.Getenv("SONGLIB_ALLOWED_HOSTS"), ",") {
		if h != "" && h == host {
			return true
		}
	}
	return os.Getenv("SONGLIB_ALLOWED_HOSTS") == ""
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
		// Public brand assets referenced by the login page itself (favicon and
		// logo): no secrets, safe to serve before authentication. Everything
		// else stays behind the session.
		if c.Method() == "GET" {
			switch c.Path() {
			case "/songlib-logo.svg", "/songlib-mark.png":
				return c.Next()
			}
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
		if !managementPasswordMatches(a.hash, body.Password) {
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
		c.Cookie(&fiber.Cookie{Name: "sl_management", Value: token, HTTPOnly: true, Secure: requestProtocol(c) == "https", SameSite: "Strict", Path: "/", MaxAge: 3600})
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

// The auth-card language (pages.css .auth/.auth-card) rebuilt inline with
// the SongLib brand palette (gold #E3B459 sampled from the official logo).
// The form logic and session behaviour are unchanged.
const managementLoginHTML = `<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>音屿-Apple TV音乐播放器安装助手</title>
<link rel="icon" type="image/png" href="/songlib-mark.png">
<style>
*{box-sizing:border-box}
html{background:#0D0F11}
body{min-height:100vh;margin:0;display:grid;place-items:center;padding:32px 20px;color:#EAE6DE;font:14px/1.6 -apple-system,BlinkMacSystemFont,'PingFang SC','Microsoft YaHei','Segoe UI',sans-serif;-webkit-font-smoothing:antialiased;background:radial-gradient(90vmax 90vmax at 12% -8%,rgba(227,180,89,.10),transparent 62%),radial-gradient(80vmax 80vmax at 96% 110%,rgba(122,152,124,.07),transparent 60%),#0D0F11}
.auth-card{width:min(420px,100%);padding:36px 34px 30px;border-radius:22px;background:rgba(30,29,27,.94);box-shadow:inset 0 0 0 1px rgba(255,255,255,.10),inset 0 1px 0 rgba(255,255,255,.06),0 40px 90px -40px rgba(0,0,0,.75);animation:pop .7s cubic-bezier(.16,1,.3,1) both}
@keyframes pop{from{opacity:0;transform:translateY(10px) scale(.97)}to{opacity:1;transform:none}}
.auth-logo{display:block;width:208px;height:auto;margin:6px auto 22px}
.auth-card h1{margin:0;font-size:18px;font-weight:800;text-align:center;letter-spacing:.2px}
.auth-card .desc{margin:8px auto 22px;text-align:center;color:#A39C90;font-size:13.5px}
.field{display:flex;flex-direction:column;gap:6px;margin-bottom:14px}
.field label{font-size:12.5px;font-weight:650;color:#A39C90}
.ifield{position:relative}
.ifield svg{position:absolute;left:13px;top:50%;width:16px;height:16px;transform:translateY(-50%);color:#79736A;pointer-events:none}
.ifield:focus-within svg{color:#E3B459}
input{width:100%;height:44px;padding:0 12px 0 40px;border-radius:12px;border:0;outline:0;background:rgba(255,255,255,.035);box-shadow:inset 0 0 0 1px rgba(255,255,255,.15);color:#EAE6DE;font:inherit;transition:box-shadow .2s}
input:focus{box-shadow:inset 0 0 0 1px rgba(227,180,89,.65),0 0 0 3px rgba(227,180,89,.10)}
input::placeholder{color:#79736A}
.go{display:block;width:100%;height:46px;margin-top:6px;border:0;border-radius:13px;font-weight:700;font-size:15px;font-family:inherit;color:#241703;background:linear-gradient(180deg,#F0CB84,#D9A95A);box-shadow:0 8px 24px -10px rgba(227,180,89,.55),inset 0 1px 0 rgba(255,255,255,.4);cursor:pointer;transition:transform .2s cubic-bezier(.16,1,.3,1)}
.go:hover{transform:translateY(-1px)}
.go:active{transform:scale(.98)}
.status{min-height:18px;margin:12px 0 0;text-align:center;font-size:13px;color:#EF8B8B}
.hintline{margin:16px 0 0;text-align:center;font-size:12px;color:#79736A}
</style></head>
<body><form class="auth-card" novalidate>
<img class="auth-logo" src="/songlib-logo.svg" alt="音屿 SongLib Amp">
<h1>音屿-Apple TV音乐播放器安装助手</h1>
<p class="desc">登录 NAS 管理服务。此密码与 Apple 账户、音屿激活码分别管理。</p>
<div class="field"><label for="mg-password">管理密码</label><div class="ifield"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="4" y="10.5" width="16" height="10" rx="2.5"/><path d="M8 10.5V7.5a4 4 0 0 1 8 0v3"/></svg><input id="mg-password" type="password" autocomplete="current-password" required maxlength="200" placeholder="8–64 位管理密码" autofocus></div></div>
<button class="go" type="submit">登录</button>
<p role="status" id="status" class="status"></p>
<p class="hintline">音屿 SongLib Amp</p>
</form>
<script>document.querySelector('form').onsubmit=async e=>{e.preventDefault();const r=await fetch('/management/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({password:document.querySelector('input').value})});if(r.ok)location.href='/';else document.querySelector('#status').textContent=r.status===429?'尝试过多，请稍后重试':'登录失败，请检查管理密码';};</script></body></html>`
