package web

import (
	"bytes"
	"image"
	_ "image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
)

// The browser tab and Apple touch icon use the official SongLib mark, not the
// upstream ATVLoadly triangle (docs/UI_PRODUCT_REDESIGN_HANDOFF_20261104.md §1).
const songlibMarkPath = "/songlib-mark.png"

func TestBrandIconSource(t *testing.T) {
	mark, err := os.ReadFile(filepath.Join("static", "public", "songlib-mark.png"))
	if err != nil {
		t.Fatalf("read source mark: %v", err)
	}
	assertSongLibMark(t, mark)

	index, err := os.ReadFile(filepath.Join("static", "index.html"))
	if err != nil {
		t.Fatalf("read source index: %v", err)
	}
	wantIcon := `<link rel="icon" type="image/png" sizes="512x512" href="/songlib-mark.png" />`
	if !strings.Contains(string(index), wantIcon) {
		t.Fatalf("source index does not contain %q", wantIcon)
	}
	wantTouch := `<link rel="apple-touch-icon" type="image/png" href="/songlib-mark.png" />`
	if !strings.Contains(string(index), wantTouch) {
		t.Fatalf("source index does not contain %q", wantTouch)
	}
	if strings.Contains(string(index), "/img/app.svg") {
		t.Fatal("source index still references the upstream SVG favicon")
	}
}

func TestBrandIconProductionAsset(t *testing.T) {
	assets := StaticAssets()
	index, err := assets.Open("index.html")
	if err != nil {
		t.Fatalf("open built index: %v", err)
	}
	defer func() { _ = index.Close() }()

	indexData, err := io.ReadAll(index)
	if err != nil {
		t.Fatalf("read built index: %v", err)
	}
	if !strings.Contains(string(indexData), songlibMarkPath) {
		t.Fatal("built index does not reference the SongLib mark")
	}
	if strings.Contains(string(indexData), "/img/app.svg") {
		t.Fatal("built index still references the upstream SVG favicon")
	}

	mark, err := assets.Open("songlib-mark.png")
	if err != nil {
		t.Fatalf("open built mark: %v", err)
	}
	defer func() { _ = mark.Close() }()
	markData, err := io.ReadAll(mark)
	if err != nil {
		t.Fatalf("read built mark: %v", err)
	}
	assertSongLibMark(t, markData)

	app := fiber.New()
	app.Use("/", filesystem.New(filesystem.Config{Root: http.FS(assets)}))
	response, err := app.Test(httptest.NewRequest(http.MethodGet, songlibMarkPath, nil))
	if err != nil {
		t.Fatalf("request mark: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("mark status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "image/png") {
		t.Fatalf("mark content type = %q, want image/png", response.Header.Get("Content-Type"))
	}
}

func assertSongLibMark(t *testing.T, data []byte) {
	t.Helper()
	mark, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode mark: %v", err)
	}
	if format != "png" {
		t.Fatalf("mark format = %q, want png", format)
	}
	// The shipped web mark is the 512x512 master from branding/ICON_512.PNG.
	if got := mark.Bounds().Size(); got.X != 512 || got.Y != 512 {
		t.Fatalf("mark dimensions = %dx%d, want 512x512", got.X, got.Y)
	}
}
