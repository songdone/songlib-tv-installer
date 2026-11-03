// SongLib modification 2026-11-02, extended 2026-11-04 with plan pricing.
// Trusted fixed-origin public configuration: the installer never talks to
// 16688 itself and never activates licenses — it only mirrors the plan links
// (and, once the official channel marks them verified for the day, the
// current price) published by the official site's sales-config endpoint.
package web

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"io"
	"net/http"
	"net/url"
	"time"
)

// salesConfigURL is a package variable only so tests can point it at a stub
// server. Production builds always use the official fixed origin.
var salesConfigURL = "https://songlib.playsong.cn/api/v1/sales-config"

func validBuyURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && (u.Host == "www.16688.com.cn" || u.Host == "16688.com.cn") && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Path != "" && u.Path != "/"
}

// validPlanAmount accepts a non-negative integer price in cents (分) as
// published by the licensing backend; anything else means "no price shown".
func validPlanAmount(amount int) bool { return amount >= 0 }

func mountSongLib(server *fiber.App) {
	server.Get("/songlib/diagnostics", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"product": "音屿-Apple TV音乐播放器安装助手", "version": "0.1.3", "process": "running", "privacy": "no account/device identifiers or logs included", "device_acceptance": "not_verified", "resign_long_term": "not_observed"})
	})
	server.Get("/songlib/config", func(c *fiber.Ctx) error {
		result := fiber.Map{"website": "https://songlib.playsong.cn/", "download": "https://songlib.playsong.cn/#download-tv", "ios": "https://songlib.playsong.cn/#download-ios", "mac": nil, "trial": "https://songlib.playsong.cn/#trial", "basic": nil, "pro": nil, "basic_price": nil, "pro_price": nil, "source": "/source", "license": "/license"}
		client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		r, e := client.Get(salesConfigURL)
		if e == nil {
			defer func() { _ = r.Body.Close() }()
			if r.StatusCode == 200 {
				var data struct {
					Plans map[string]struct {
						BuyURL   string `json:"buy_url"`
						Verified bool   `json:"verified"`
						Amount   int    `json:"amount"`
					} `json:"plans"`
				}
				if json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&data) == nil {
					for _, p := range []string{"basic", "pro"} {
						m := data.Plans[p]
						if m.Verified && validBuyURL(m.BuyURL) {
							result[p] = m.BuyURL
							// The price travels with a verified channel link
							// only; unverified plans stay linkless AND
							// priceless so no stale or invented amounts can
							// reach the UI.
							if validPlanAmount(m.Amount) {
								result[p+"_price"] = m.Amount
							}
						}
					}
				}
			}
		}
		return c.JSON(result)
	})
}
