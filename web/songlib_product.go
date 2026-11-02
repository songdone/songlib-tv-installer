// SongLib modification 2026-11-02. Trusted fixed-origin public configuration.
package web

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"io"
	"net/http"
	"net/url"
	"time"
)

func validBuyURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && (u.Host == "www.16688.com.cn" || u.Host == "16688.com.cn") && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Path != "" && u.Path != "/"
}
func mountSongLib(server *fiber.App) {
	server.Get("/songlib/config", func(c *fiber.Ctx) error {
		result := fiber.Map{"website": "https://songlib.playsong.cn/", "download": "https://songlib.playsong.cn/#download-tv", "ios": "https://songlib.playsong.cn/#download-ios", "mac": nil, "trial": "https://songlib.playsong.cn/#trial", "basic": nil, "pro": nil, "source": "/source", "license": "/license"}
		client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		r, e := client.Get("https://songlib.playsong.cn/api/v1/sales-config")
		if e == nil {
			defer r.Body.Close()
			if r.StatusCode == 200 {
				var data struct {
					Plans map[string]struct {
						BuyURL   string `json:"buy_url"`
						Verified bool   `json:"verified"`
					} `json:"plans"`
				}
				if json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&data) == nil {
					for _, p := range []string{"basic", "pro"} {
						m := data.Plans[p]
						if m.Verified && validBuyURL(m.BuyURL) {
							result[p] = m.BuyURL
						}
					}
				}
			}
		}
		return c.JSON(result)
	})
}
