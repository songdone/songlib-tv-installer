// SongLib modification 2026-11-02: authenticated management boundary.
package web

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
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
	dataDir  string
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

func hashManagementPassword(password string) ([]byte, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(managementPasswordMaterial(password)), 12)
	if err != nil {
		return nil, err
	}
	return append([]byte("sha256$"), hash...), nil
}

func managementPasswordMatches(hash []byte, password string) bool {
	if strings.HasPrefix(string(hash), "sha256$") {
		return bcrypt.CompareHashAndPassword(hash[len("sha256$"):], []byte(managementPasswordMaterial(password))) == nil
	}
	return bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
}

func writePasswordHashAtomic(dataDir string, hash []byte) error {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}
	tmpPath := filepath.Join(dataDir, ".management-password.hash."+randomToken()[:16])
	f, err := os.OpenFile(tmpPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(hash)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil {
		_ = os.Remove(tmpPath)
		return writeErr
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	target := filepath.Join(dataDir, "management-password.hash")
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
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
		hash, err = hashManagementPassword(password)
		if err != nil {
			return nil, err
		}
		if err = writePasswordHashAtomic(dataDir, hash); err != nil {
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
	return &managementAuth{
		dataDir:  dataDir,
		hash:     hash,
		sessions: map[[32]byte]managementSession{},
		attempts: map[string]attempt{},
	}, nil
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
	if site := c.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" && site != "none" {
		return false
	}
	expectedScheme := requestProtocol(c)
	expectedHost := string(c.Request().Host())
	origin := c.Get("Origin")
	if origin != "" {
		return origin == expectedScheme+"://"+expectedHost
	}
	// When Origin is omitted (for example certain same-origin browser form POSTs),
	// require a valid same-origin Referer URL. Never accept missing both.
	referer := c.Get("Referer")
	if referer == "" {
		return false
	}
	u, err := url.Parse(referer)
	if err != nil || u.User != nil || u.Opaque != "" || u.Host == "" {
		return false
	}
	return u.Scheme == expectedScheme && strings.EqualFold(u.Host, expectedHost)
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

func (a *managementAuth) recordAttempt(ip string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	at := a.attempts[ip]
	if time.Now().After(at.until) {
		at = attempt{until: time.Now().Add(10 * time.Minute)}
	}
	at.count++
	if len(a.attempts) > 4096 {
		for k, v := range a.attempts {
			if time.Now().After(v.until) {
				delete(a.attempts, k)
			}
		}
	}
	a.attempts[ip] = at
	return at.count
}

func (a *managementAuth) clearAttempts(ip string) {
	a.mu.Lock()
	delete(a.attempts, ip)
	a.mu.Unlock()
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
			csrfToken := c.Get("X-CSRF-Token")
			if csrfToken == "" && c.Path() == "/management/web-update/recovery" {
				csrfToken = c.FormValue("csrf")
			}
			if !sameOrigin(c) || subtle.ConstantTimeCompare([]byte(csrfToken), []byte(session.csrf)) != 1 {
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
		if a.recordAttempt(c.IP()) > 10 {
			return c.SendStatus(429)
		}
		var body struct {
			Password string `json:"password"`
		}
		if len(c.Body()) > 1024 || json.Unmarshal(c.Body(), &body) != nil {
			return c.SendStatus(400)
		}
		a.mu.Lock()
		currentHash := append([]byte(nil), a.hash...)
		a.mu.Unlock()
		if !managementPasswordMatches(currentHash, body.Password) {
			return c.SendStatus(401)
		}
		a.clearAttempts(c.IP())
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
	server.Post("/management/password", func(c *fiber.Ctx) error {
		if a.recordAttempt(c.IP()) > 10 {
			return c.Status(429).JSON(fiber.Map{"error": "尝试过多，请稍后重试"})
		}
		var body struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if len(c.Body()) > 1024 || json.Unmarshal(c.Body(), &body) != nil {
			return c.Status(400).JSON(fiber.Map{"error": "请求格式无效"})
		}
		if !validManagementPassword(body.NewPassword) {
			return c.Status(400).JSON(fiber.Map{"error": "新密码须为 8 至 64 位，且不能包含换行符"})
		}
		a.mu.Lock()
		currentHash := append([]byte(nil), a.hash...)
		a.mu.Unlock()
		if !managementPasswordMatches(currentHash, body.CurrentPassword) {
			return c.Status(401).JSON(fiber.Map{"error": "当前管理密码不正确"})
		}
		a.clearAttempts(c.IP())
		newHash, err := hashManagementPassword(body.NewPassword)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "保存管理密码失败"})
		}
		if err := writePasswordHashAtomic(a.dataDir, newHash); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "写入管理密码文件失败"})
		}
		// Rotate current session and invalidate all other active sessions.
		token, csrf := randomToken(), randomToken()
		a.mu.Lock()
		a.hash = newHash
		a.sessions = map[[32]byte]managementSession{
			sha256.Sum256([]byte(token)): {csrf: csrf, expires: time.Now().Add(time.Hour)},
		}
		a.mu.Unlock()
		c.Cookie(&fiber.Cookie{Name: "sl_management", Value: token, HTTPOnly: true, Secure: requestProtocol(c) == "https", SameSite: "Strict", Path: "/", MaxAge: 3600})
		return c.JSON(fiber.Map{"ok": true, "csrf": csrf})
	})
	server.Post("/management/logout", func(c *fiber.Ctx) error {
		a.mu.Lock()
		delete(a.sessions, sha256.Sum256([]byte(c.Cookies("sl_management"))))
		a.mu.Unlock()
		c.Cookie(&fiber.Cookie{Name: "sl_management", Value: "", HTTPOnly: true, Secure: requestProtocol(c) == "https", SameSite: "Strict", Path: "/", MaxAge: -1, Expires: time.Unix(0, 0)})
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
.ifield .lock{position:absolute;left:13px;top:50%;width:16px;height:16px;transform:translateY(-50%);color:#79736A;pointer-events:none}
.ifield:focus-within .lock{color:#E3B459}
input{width:100%;height:44px;padding:0 58px 0 40px;border-radius:12px;border:0;outline:0;background:rgba(255,255,255,.035);box-shadow:inset 0 0 0 1px rgba(255,255,255,.15);color:#EAE6DE;font-family:inherit;font-size:max(1rem,1em);transition:box-shadow .2s}
input:focus{box-shadow:inset 0 0 0 1px rgba(227,180,89,.65),0 0 0 3px rgba(227,180,89,.10)}
input::placeholder{color:#79736A;font-size:14px}
.toggle{position:absolute;right:8px;top:50%;transform:translateY(-50%);border:0;border-radius:8px;padding:5px 8px;background:transparent;color:#A39C90;font:600 12px/1 inherit;cursor:pointer}
.toggle:hover,.toggle:focus-visible{color:#EAE6DE;background:rgba(255,255,255,.08)}
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
<div class="field"><label for="mg-password">管理密码</label><div class="ifield"><svg class="lock" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><rect x="4" y="10.5" width="16" height="10" rx="2.5"/><path d="M8 10.5V7.5a4 4 0 0 1 8 0v3"/></svg><input id="mg-password" type="password" autocomplete="current-password" required maxlength="200" placeholder="8–64 位管理密码" autofocus><button id="mg-toggle" class="toggle" type="button" aria-pressed="false" aria-label="显示密码">显示</button></div></div>
<button class="go" type="submit">登录</button>
<p role="status" id="status" class="status"></p>
<p class="hintline">音屿 SongLib Amp</p>
</form>
<script>const pw=document.querySelector('#mg-password'),tg=document.querySelector('#mg-toggle'),st=document.querySelector('#status');tg.onclick=()=>{const show=pw.type==='password';pw.type=show?'text':'password';tg.setAttribute('aria-pressed',String(show));tg.textContent=show?'隐藏':'显示';tg.setAttribute('aria-label',show?'隐藏密码':'显示密码');};pw.oninput=()=>{st.textContent='';};document.querySelector('form').onsubmit=async e=>{e.preventDefault();const r=await fetch('/management/login',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({password:pw.value})});if(r.ok)location.href='/';else{st.textContent=r.status===429?'尝试过多，请稍后重试':'登录失败，请检查管理密码';pw.focus();}};</script></body></html>`
