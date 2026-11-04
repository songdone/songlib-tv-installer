// SongLib modification 2026-11-02, extended 2026-11-04 with plan pricing and
// trial gating.
// Trusted fixed-origin public configuration: the installer never talks to
// 16688 itself and never activates licenses — it only mirrors the plan links
// (and, once the official channel marks them verified for the day, the
// current price) published by the official site's sales-config endpoint.
package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bitxeno/atvloadly/internal/app/build"
	"github.com/gofiber/fiber/v2"
)

// salesConfigURL is a package variable only so tests can point it at a stub
// server. Production builds always use the official fixed origin.
var salesConfigURL = "https://songlib.playsong.cn/api/v1/sales-config"

// trialFallbackDays is the documented trial length. The upstream value only
// replaces it when it is a sane number of days.
const trialFallbackDays = 30

// defaultTrialAnchor is the official site's trial section. It is an
// informational link ("read about the trial"), never a claim that the trial is
// open — that is what trial_open/trial_url are for.
const defaultTrialAnchor = "https://songlib.playsong.cn/#trial"

func validBuyURL(raw string) bool {
	u, e := url.Parse(raw)
	return e == nil && u.Scheme == "https" && (u.Host == "www.16688.com.cn" || u.Host == "16688.com.cn") && u.User == nil && u.RawQuery == "" && u.Fragment == "" && u.Path != "" && u.Path != "/"
}

// validClaimURL keeps the trial claim link on the official site: a sales
// config must not be able to send users anywhere else (or downgrade to http).
func validClaimURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	return u.Host == "songlib.playsong.cn" || u.Host == "www.songlib.playsong.cn"
}

// validPlanAmount accepts a positive integer price in cents (分) as published
// by the licensing backend. Zero, negative and absent amounts all mean "no
// price shown": a plan is never advertised as free by accident.
func validPlanAmount(amount *int) bool { return amount != nil && *amount > 0 }

func validTrialDays(days int) bool { return days > 0 && days <= 365 }

func mountSongLib(server *fiber.App) {
	server.Get("/songlib/diagnostics", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"product": "音屿-Apple TV音乐播放器安装助手", "version": build.Version, "process": "running", "privacy": "no account/device identifiers or logs included", "device_acceptance": "not_verified", "resign_long_term": "not_observed"})
	})
	server.Get("/songlib/config", func(c *fiber.Ctx) error {
		result := fiber.Map{"website": "https://songlib.playsong.cn/", "download": "https://songlib.playsong.cn/#download-tv", "ios": "https://songlib.playsong.cn/#download-ios", "mac": nil, "trial": defaultTrialAnchor, "trial_url": nil, "trial_days": trialFallbackDays, "trial_open": false, "basic": nil, "pro": nil, "basic_price": nil, "pro_price": nil, "source": "/source", "license": "/license"}
		client := &http.Client{Timeout: 3 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
		r, e := client.Get(salesConfigURL)
		if e == nil {
			defer func() { _ = r.Body.Close() }()
			if r.StatusCode == 200 {
				var data struct {
					Plans map[string]struct {
						BuyURL   string `json:"buy_url"`
						Verified bool   `json:"verified"`
						Amount   *int   `json:"amount"`
					} `json:"plans"`
					Trial struct {
						Days     int    `json:"days"`
						Open     bool   `json:"open"`
						ClaimURL string `json:"claim_url"`
					} `json:"trial"`
				}
				if json.NewDecoder(io.LimitReader(r.Body, 65536)).Decode(&data) == nil {
					// 30-day Pro experience card state: rendered by the purchase
					// page as a link to the official claim flow, never as a
					// local activation box. Both the link and the "open" flag
					// must survive validation before the UI offers a claim.
					if validClaimURL(data.Trial.ClaimURL) {
						if validTrialDays(data.Trial.Days) {
							result["trial_days"] = data.Trial.Days
						}
						result["trial_url"] = data.Trial.ClaimURL
						result["trial_open"] = data.Trial.Open
					}
					for _, p := range []string{"basic", "pro"} {
						m := data.Plans[p]
						if m.Verified && validBuyURL(m.BuyURL) {
							result[p] = m.BuyURL
							// The price travels with a verified channel link
							// only; unverified plans stay linkless AND
							// priceless so no stale or invented amounts can
							// reach the UI.
							if validPlanAmount(m.Amount) {
								result[p+"_price"] = *m.Amount
							}
						}
					}
				}
			}
		}
		return c.JSON(result)
	})
}
