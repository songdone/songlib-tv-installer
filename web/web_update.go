package web

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"html"
	"html/template"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/bitxeno/atvloadly/internal/app"
	"github.com/bitxeno/atvloadly/internal/app/build"
	"github.com/bitxeno/atvloadly/internal/log"
	"github.com/bitxeno/atvloadly/internal/release"
	"github.com/bitxeno/atvloadly/internal/service"
	"github.com/bitxeno/atvloadly/internal/task"
	"github.com/bitxeno/atvloadly/internal/webupdate"
	"github.com/gofiber/fiber/v2"
)

const webUpdateNoSource = "暂未配置在线界面更新源，当前使用安装包自带界面。"

type webUpdateController struct {
	manager        *webupdate.Manager
	packagedFS     fs.FS
	deployMode     string
	initErr        string
	hasActiveTasks func() bool

	opMu     sync.Mutex
	updating bool
}

type webUpdateStatusResponse struct {
	webupdate.Status
	DeployMode   string       `json:"deploy_mode"`
	CoreRelease  release.Info `json:"core_release"`
	RecoveryPath string       `json:"recovery_path"`
}

func newWebUpdateController(dataDir string, packagedFS fs.FS) *webUpdateController {
	ctrl := &webUpdateController{
		packagedFS:     packagedFS,
		deployMode:     build.DeployMode(),
		hasActiveTasks: task.HasActiveWork,
	}
	if dataDir == "" {
		return ctrl
	}
	feed := strings.TrimSpace(os.Getenv("SONGLIB_WEB_UPDATE_FEED"))
	if feed == "" {
		feed = build.WebUpdateFeed
	}
	keyID := strings.TrimSpace(os.Getenv("SONGLIB_WEB_UPDATE_KEY_ID"))
	if keyID == "" {
		keyID = build.WebUpdateKeyID
	}
	pubB64 := strings.TrimSpace(os.Getenv("SONGLIB_WEB_UPDATE_PUBLIC_KEY"))
	if pubB64 == "" {
		pubB64 = build.WebUpdatePublicKey
	}
	var pubKey ed25519.PublicKey
	if pubB64 != "" {
		decoded, err := base64.StdEncoding.Strict().DecodeString(pubB64)
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			ctrl.initErr = "界面更新公钥格式无效，已保持使用随包界面。"
			log.Warn("invalid SONGLIB_WEB_UPDATE_PUBLIC_KEY; web hot update disabled")
			return ctrl
		}
		pubKey = ed25519.PublicKey(decoded)
	}
	allowHTTP := strings.EqualFold(strings.TrimSpace(os.Getenv("SONGLIB_WEB_UPDATE_ALLOW_HTTP")), "true") ||
		strings.TrimSpace(os.Getenv("SONGLIB_WEB_UPDATE_ALLOW_HTTP")) == "1"

	mgr, err := webupdate.New(webupdate.Config{
		DataDir:        dataDir,
		PackagedFS:     packagedFS,
		BackendVersion: build.Version,
		BackendID:      build.BackendID(),
		FeedURL:        feed,
		KeyID:          keyID,
		PublicKey:      pubKey,
		AllowHTTP:      allowHTTP,
	})
	if err != nil {
		ctrl.initErr = "界面热更新初始化失败，已保持使用随包界面。"
		log.Err(err).Msg("web hot update manager init failed")
		return ctrl
	}
	ctrl.manager = mgr
	return ctrl
}

func (w *webUpdateController) localStatus() webUpdateStatusResponse {
	var st webupdate.Status
	if w.manager != nil {
		st = w.manager.Status()
		switch {
		case strings.HasPrefix(st.Reason, "using packaged frontend; no saved update state"):
			st.Reason = ""
		case strings.Contains(st.Reason, "backend build changed"):
			st.Reason = "已换用新核心安装包自带的界面。"
		case strings.HasPrefix(st.Reason, "active frontend failed validation"):
			st.Reason = "上次激活的界面校验异常，已自动回退到上一版。"
		case strings.HasPrefix(st.Reason, "previous frontend failed validation"):
			st.Reason = "上一版界面快照校验异常，暂时无法回退。"
		case strings.HasPrefix(st.Reason, "using packaged frontend;"):
			st.Reason = "保存的界面快照不可用，已自动恢复原版界面。"
		case st.Reason != "":
			st.Reason = "上次界面更新操作有异常，请核对当前界面版本。"
		}
		if st.Available != nil && !st.Available.Compatible {
			st.Available.Reason = "需要升级核心镜像或飞牛 FPK（最低要求版本 " + st.Available.MinBackendVersion + "）。"
		}
		if !st.Configured {
			if st.Reason == "" {
				st.Reason = webUpdateNoSource
			} else {
				st.Reason = webUpdateNoSource + " " + st.Reason
			}
		}
	} else {
		reason := webUpdateNoSource
		if w.initErr != "" {
			reason = w.initErr
		}
		st = webupdate.Status{
			InstalledBackendVersion: build.Version,
			BackendID:               build.BackendID(),
			PackagedVersion:         build.Version,
			ActiveVersion:           build.Version,
			Reason:                  reason,
		}
	}

	w.opMu.Lock()
	activeTasks := w.hasActiveTasks != nil && w.hasActiveTasks()
	st.Busy = st.Busy || w.updating || activeTasks || service.ActiveInteractiveSessions() > 0
	w.opMu.Unlock()

	return webUpdateStatusResponse{
		Status:       st,
		DeployMode:   w.deployMode,
		CoreRelease:  release.Latest(),
		RecoveryPath: "/management/web-update/recovery",
	}
}

func (w *webUpdateController) reserveExecution() (func(), error) {
	w.opMu.Lock()
	if w.updating {
		w.opMu.Unlock()
		return nil, webupdate.ErrBusy
	}
	w.updating = true
	w.opMu.Unlock()

	releaseFlag := func() {
		w.opMu.Lock()
		w.updating = false
		w.opMu.Unlock()
	}

	releaseSessions, ok := service.TryReserveMaintenance()
	if !ok {
		releaseFlag()
		return nil, webupdate.ErrBusy
	}
	if w.hasActiveTasks != nil && w.hasActiveTasks() {
		releaseSessions()
		releaseFlag()
		return nil, webupdate.ErrBusy
	}
	return func() {
		releaseSessions()
		releaseFlag()
	}, nil
}

func (w *webUpdateController) perform(ctx context.Context, action, revision string, expected, previous *string) error {
	if w.manager == nil {
		return webupdate.ErrUnavailable
	}
	release, err := w.reserveExecution()
	if err != nil {
		return err
	}
	defer release()

	status := w.manager.Status()
	if action != "check" && (expected == nil || *expected != status.ActiveRevision) {
		return webupdate.ErrStale
	}
	if action == "rollback" && (previous == nil || !status.CanRollback || *previous != status.PreviousRevision) {
		return webupdate.ErrStale
	}

	switch action {
	case "check":
		_, err = w.manager.Check(ctx)
	case "apply":
		_, err = w.manager.Apply(ctx, revision)
	case "rollback":
		_, err = w.manager.Rollback(ctx)
	case "restore":
		_, err = w.manager.RestorePackaged(ctx)
	default:
		return webupdate.ErrUnavailable
	}
	return err
}

func writeWebUpdateAPIError(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, webupdate.ErrBusy):
		return c.Status(http.StatusConflict).JSON(apiError("当前有安装、配对、登录、续签或其它更新任务正在执行，请完成后重试。"))
	case errors.Is(err, webupdate.ErrStale):
		return c.Status(http.StatusConflict).JSON(apiError("当前界面版本或发布清单已变化，请刷新本机状态并重新检查更新。"))
	case errors.Is(err, webupdate.ErrIncompatible):
		return c.Status(http.StatusConflict).JSON(apiError("该界面更新需要更新核心容器镜像或飞牛 FPK 后再使用。"))
	case errors.Is(err, webupdate.ErrUnavailable):
		return c.Status(http.StatusServiceUnavailable).JSON(apiError("当前没有可用的在线更新源或可回退快照。"))
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return c.Status(http.StatusGatewayTimeout).JSON(apiError("等待操作超时，请先刷新本机状态确认是否已生效，不要盲目重发。"))
	default:
		log.Err(err).Msg("web hot update action failed")
		return c.Status(http.StatusBadGateway).JSON(apiError("下载或签名/文件校验未通过，未确认界面切换。请刷新状态核对当前版本。"))
	}
}

func (w *webUpdateController) registerAPI(api fiber.Router) {
	api.Get("/web-update", func(c *fiber.Ctx) error {
		return c.Status(http.StatusOK).JSON(apiSuccess(w.localStatus()))
	})
	api.Post("/web-update/:action", func(c *fiber.Ctx) error {
		action := c.Params("action")
		if action != "check" && action != "apply" && action != "rollback" && action != "restore" {
			return c.Status(http.StatusNotFound).JSON(apiError("不支持的更新操作"))
		}
		if len(c.Body()) > 4096 {
			return c.Status(http.StatusBadRequest).JSON(apiError("请求体过大"))
		}
		var body struct {
			Revision string  `json:"revision,omitempty"`
			Expected *string `json:"expected_active_revision,omitempty"`
			Previous *string `json:"expected_previous_revision,omitempty"`
		}
		if len(bytes.TrimSpace(c.Body())) > 0 {
			dec := json.NewDecoder(bytes.NewReader(c.Body()))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&body); err != nil {
				return c.Status(http.StatusBadRequest).JSON(apiError("无效的请求参数"))
			}
		}
		timeout := 20 * time.Second
		if action == "apply" {
			timeout = 100 * time.Second
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), timeout)
		defer cancel()
		if err := w.perform(ctx, action, body.Revision, body.Expected, body.Previous); err != nil {
			return writeWebUpdateAPIError(c, err)
		}
		return c.Status(http.StatusOK).JSON(apiSuccess(w.localStatus()))
	})
}

var webRecoveryTemplate = template.Must(template.New("songlib-web-recovery").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>音屿安装助手 · 界面恢复</title>
<style>
*{box-sizing:border-box}
body{margin:0;padding:40px 20px;font:14px/1.7 -apple-system,BlinkMacSystemFont,'PingFang SC','Microsoft YaHei',sans-serif;background:#0D0F11;color:#EAE6DE}
main{max-width:620px;margin:0 auto;padding:28px;border-radius:18px;background:#1A1917;border:1px solid rgba(255,255,255,.1)}
h1{margin:0 0 8px;font-size:20px}
p{margin:8px 0;color:#B8B2A6}
code{padding:2px 6px;border-radius:6px;background:rgba(255,255,255,.08);color:#F0CB84;overflow-wrap:anywhere}
.actions{display:flex;flex-wrap:wrap;gap:12px;margin:20px 0}
button{padding:10px 18px;border-radius:10px;border:1px solid rgba(227,180,89,.45);background:#E3B459;color:#241703;font-weight:700;cursor:pointer;font-family:inherit;font-size:14px}
button.secondary{background:transparent;color:#EAE6DE;border-color:rgba(255,255,255,.22)}
a{color:#E3B459}
.note{font-size:13px;color:#9A9489}
</style>
</head>
<body>
<main>
<h1>音屿安装助手 · 独立界面恢复</h1>
<p>本页面由后端直接输出，不依赖热更新前端脚本。恢复界面不会影响你的 Apple 账户、配对记录、已安装应用或管理密码。</p>
<p>当前界面版本：<code>{{.Status.ActiveVersion}}</code>{{if .Status.CanRestore}} · 随包原版：<code>{{.Status.PackagedVersion}}</code>{{else}}（当前已是随包原版界面）{{end}}</p>
{{if .Status.Reason}}<p class="note">状态说明：{{.Status.Reason}}</p>{{end}}
<div class="actions">
{{if .Status.CanRollback}}
<form method="post" action="/management/web-update/recovery">
  <input type="hidden" name="csrf" value="{{.CSRF}}">
  <input type="hidden" name="action" value="rollback">
  <input type="hidden" name="expected" value="{{.Status.ActiveRevision}}">
  <input type="hidden" name="previous" value="{{.Status.PreviousRevision}}">
  <button type="submit">回退到上一版 ({{.Status.PreviousVersion}})</button>
</form>
{{end}}
{{if .Status.CanRestore}}
<form method="post" action="/management/web-update/recovery">
  <input type="hidden" name="csrf" value="{{.CSRF}}">
  <input type="hidden" name="action" value="restore">
  <input type="hidden" name="expected" value="{{.Status.ActiveRevision}}">
  <button type="submit" class="secondary">恢复安装包原版界面</button>
</form>
{{end}}
</div>
<p><a href="/">返回音屿安装助手</a></p>
</main>
</body>
</html>`))

func (w *webUpdateController) mountRecoveryAndAssets(server *fiber.App) {
	server.Get("/management/web-update/recovery", func(c *fiber.Ctx) error {
		csrf, _ := c.Locals("management_csrf").(string)
		c.Set("Cache-Control", documentCacheControl)
		// Same-origin referrer ensures browser form POSTs include Origin/Referer
		// for sameOrigin verification without leaking cross-origin URLs.
		c.Set("Referrer-Policy", "same-origin")
		c.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'self'")
		c.Type("html", "utf-8")
		var buf bytes.Buffer
		_ = webRecoveryTemplate.Execute(&buf, struct {
			Status webUpdateStatusResponse
			CSRF   string
		}{w.localStatus(), csrf})
		return c.Send(buf.Bytes())
	})
	server.Post("/management/web-update/recovery", func(c *fiber.Ctx) error {
		action := c.FormValue("action")
		if action != "restore" && action != "rollback" {
			return c.Status(http.StatusBadRequest).SendString("无效的恢复操作")
		}
		expected := c.FormValue("expected")
		var previous *string
		if action == "rollback" {
			prev := c.FormValue("previous")
			previous = &prev
		}
		ctx, cancel := context.WithTimeout(c.UserContext(), 15*time.Second)
		defer cancel()
		if err := w.perform(ctx, action, "", &expected, previous); err != nil {
			return writeWebUpdateAPIError(c, err)
		}
		return c.Redirect("/management/web-update/recovery", http.StatusSeeOther)
	})
	server.Get("/_web/:revision/*", func(c *fiber.Ctx) error {
		rev := c.Params("revision")
		rel := strings.TrimPrefix(path.Clean("/"+c.Params("*")), "/")
		if rel == "" || rel == "index.html" || !webupdate.ValidPath(rel) {
			return c.SendStatus(http.StatusNotFound)
		}
		data, err := w.readRevision(rev, rel)
		if err != nil {
			return c.SendStatus(http.StatusNotFound)
		}
		c.Set("Cache-Control", "private, max-age=31536000, immutable")
		return sendStaticBytes(c, rel, data)
	})
}

func (w *webUpdateController) readRevision(rev, rel string) ([]byte, error) {
	if w.manager != nil {
		return w.manager.ReadRevisionFile(rev, rel)
	}
	if rev == "packaged" || rev == "packaged-"+build.BackendID() {
		if !webupdate.ValidPath(rel) {
			return nil, os.ErrNotExist
		}
		return fs.ReadFile(w.packagedFS, rel)
	}
	return nil, os.ErrNotExist
}

func (w *webUpdateController) activeRevisionToken() (revToken, revKey, version string) {
	if w.manager != nil {
		st := w.manager.Status()
		if st.ActiveRevision != "" {
			return st.ActiveRevision, st.ActiveRevision, st.ActiveVersion
		}
		return w.manager.PackagedRevisionToken(), "packaged", st.PackagedVersion
	}
	return "packaged-" + build.BackendID(), "packaged", build.Version
}

func injectAssetBase(indexHTML []byte, revToken, version string) []byte {
	injection := `<base href="/_web/` + html.EscapeString(revToken) + `/" />` +
		`<meta name="songlib-web-revision" content="` + html.EscapeString(revToken) + `" />` +
		`<meta name="songlib-web-version" content="` + html.EscapeString(version) + `" />`
	s := string(indexHTML)
	if idx := strings.Index(strings.ToLower(s), "<head>"); idx >= 0 {
		insertAt := idx + len("<head>")
		return []byte(s[:insertAt] + "\n  " + injection + s[insertAt:])
	}
	return append([]byte(injection+"\n"), indexHTML...)
}

func sendStaticBytes(c *fiber.Ctx, rel string, data []byte) error {
	if ct := mime.TypeByExtension(path.Ext(rel)); ct != "" {
		c.Set(fiber.HeaderContentType, ct)
	}
	if c.Method() == fiber.MethodHead {
		c.Set(fiber.HeaderContentLength, fmtInt(len(data)))
		return c.SendStatus(http.StatusOK)
	}
	return c.Status(http.StatusOK).Send(data)
}

func fmtInt(v int) string {
	return string(json.Number(strings.TrimSpace(string(mustMarshalInt(v)))))
}

func mustMarshalInt(v int) []byte {
	b, _ := json.Marshal(v)
	return b
}

func (w *webUpdateController) frontendMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Method() != fiber.MethodGet && c.Method() != fiber.MethodHead {
			return c.Next()
		}
		reqPath := c.Path()
		// Serve index.html with injected revision base and no-store cache headers.
		if reqPath == "/" || reqPath == "/index.html" {
			c.Set(fiber.HeaderCacheControl, documentCacheControl)
			revToken, revKey, version := w.activeRevisionToken()
			data, err := w.readRevision(revKey, "index.html")
			if err != nil {
				c.Set("Content-Type", "text/html; charset=utf-8")
				return c.Status(http.StatusServiceUnavailable).SendString(
					`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><title>界面异常</title></head><body><h1>当前界面文件异常</h1><p>检测到已激活的热更新界面损坏，点击下方链接恢复安装包原版界面。</p><p><a href="/management/web-update/recovery">打开界面恢复页</a></p></body></html>`,
				)
			}
			rendered := injectAssetBase(data, revToken, version)
			c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
			if c.Method() == fiber.MethodHead {
				c.Set(fiber.HeaderContentLength, fmtInt(len(rendered)))
				return c.SendStatus(http.StatusOK)
			}
			return c.Status(http.StatusOK).Send(rendered)
		}

		rel := strings.TrimPrefix(path.Clean(reqPath), "/")
		if !webupdate.ValidPath(rel) {
			return c.Next()
		}
		// Unauthenticated login-page brand assets always come from the packaged go:embed FS.
		if c.Locals("management_csrf") == nil && (rel == "songlib-logo.svg" || rel == "songlib-mark.png") {
			data, err := fs.ReadFile(w.packagedFS, rel)
			if err != nil {
				return c.SendStatus(http.StatusNotFound)
			}
			return sendStaticBytes(c, rel, data)
		}
		_, revKey, _ := w.activeRevisionToken()
		data, err := w.readRevision(revKey, rel)
		if err != nil {
			// Missing JS/CSS/media must return 404 directly, never fall through to SPA HTML.
			return c.SendStatus(http.StatusNotFound)
		}
		c.Set(fiber.HeaderCacheControl, documentCacheControl)
		return sendStaticBytes(c, rel, data)
	}
}

// initDefaultWebUpdateController initializes a controller using the configured
// app DataDir when available.
func initDefaultWebUpdateController() *webUpdateController {
	dataDir := ""
	if app.Config != nil {
		dataDir = app.Config.Server.DataDir
	}
	return newWebUpdateController(dataDir, StaticAssets())
}
