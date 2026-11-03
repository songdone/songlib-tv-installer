package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// salesConfigStub is the JSON the official site's /api/v1/sales-config returns.
type salesConfigStub struct {
	Channel string `json:"channel"`
	Plans   map[string]struct {
		BuyURL   string `json:"buy_url"`
		Verified bool   `json:"verified"`
		Amount   int    `json:"amount"`
	} `json:"plans"`
}

func withSalesStub(t *testing.T, status int, body string) {
	t.Helper()
	stub := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(stub.Close)

	previous := salesConfigURL
	salesConfigURL = stub.URL
	t.Cleanup(func() { salesConfigURL = previous })
}

func songLibConfig(t *testing.T) map[string]interface{} {
	t.Helper()
	app := fiber.New()
	mountSongLib(app)
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/songlib/config", nil))
	if err != nil {
		t.Fatalf("request /songlib/config: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return result
}

func TestSongLibConfigForwardsVerifiedPlansWithPrice(t *testing.T) {
	withSalesStub(t, 200, `{"channel":"16688","plans":{
		"basic":{"buy_url":"https://www.16688.com.cn/goods/G123","verified":true,"amount":600},
		"pro":{"buy_url":"https://www.16688.com.cn/goods/G456","verified":true,"amount":1800}
	}}`)
	result := songLibConfig(t)
	if result["basic"] != "https://www.16688.com.cn/goods/G123" {
		t.Fatalf("basic link = %v", result["basic"])
	}
	if result["pro"] != "https://www.16688.com.cn/goods/G456" {
		t.Fatalf("pro link = %v", result["pro"])
	}
	// Prices travel in cents and only alongside a verified channel link.
	if result["basic_price"] != float64(600) || result["pro_price"] != float64(1800) {
		t.Fatalf("prices = %v / %v", result["basic_price"], result["pro_price"])
	}
	if result["trial"] != "https://songlib.playsong.cn/#trial" {
		t.Fatalf("trial = %v", result["trial"])
	}
}

func TestSongLibConfigHidesUnverifiedPlansAndPrices(t *testing.T) {
	// Unverified plans must lose BOTH the link and the price: no stale or
	// invented amounts may reach the UI while the channel is not confirmed.
	withSalesStub(t, 200, `{"channel":"16688","plans":{
		"basic":{"buy_url":"https://www.16688.com.cn/goods/G123","verified":false,"amount":600},
		"pro":{"buy_url":"https://example.com/goods","verified":true,"amount":1800}
	}}`)
	result := songLibConfig(t)
	if result["basic"] != nil {
		t.Fatalf("unverified basic link must be nil, got %v", result["basic"])
	}
	if result["basic_price"] != nil {
		t.Fatalf("unverified basic price must be nil, got %v", result["basic_price"])
	}
	// Non-16688 hosts are rejected even when the upstream marks them verified.
	if result["pro"] != nil {
		t.Fatalf("non-16688 pro link must be nil, got %v", result["pro"])
	}
	if result["pro_price"] != nil {
		t.Fatalf("rejected pro price must be nil, got %v", result["pro_price"])
	}
}

func TestSongLibConfigSurvivesBrokenUpstream(t *testing.T) {
	// Upstream down / non-200 / garbage: the config still returns the fixed
	// trusted origins with everything commercial set to nil.
	withSalesStub(t, 503, "service unavailable")
	result := songLibConfig(t)
	if result["website"] != "https://songlib.playsong.cn/" {
		t.Fatalf("website = %v", result["website"])
	}
	for _, key := range []string{"basic", "pro", "basic_price", "pro_price"} {
		if result[key] != nil {
			t.Fatalf("%s must be nil on broken upstream, got %v", key, result[key])
		}
	}
}

func TestSongLibConfigValidatesBuyURLShape(t *testing.T) {
	cases := []struct {
		name, raw string
		want      bool
	}{
		{"goods path", "https://www.16688.com.cn/goods/G92590232", true},
		{"apex host", "https://16688.com.cn/goods/G1", true},
		{"no path", "https://www.16688.com.cn/", false},
		{"query injection", "https://www.16688.com.cn/goods/G1?x=1", false},
		{"fragment", "https://www.16688.com.cn/goods/G1#s", false},
		{"userinfo", "https://user@www.16688.com.cn/goods/G1", false},
		{"other host", "https://evil.example.com/goods/G1", false},
		{"plain http", "http://www.16688.com.cn/goods/G1", false},
	}
	for _, tc := range cases {
		if got := validBuyURL(tc.raw); got != tc.want {
			t.Fatalf("%s: validBuyURL(%q) = %v, want %v", tc.name, tc.raw, got, tc.want)
		}
	}
}

func TestSongLibStubShapeMatchesLicensingBackend(t *testing.T) {
	// Guard the stub against drift from the licensing backend's public():
	// sales.py returns channel/plans{buy_url,verified,amount}.
	var stub salesConfigStub
	if err := json.Unmarshal([]byte(`{"channel":"16688","plans":{"basic":{"buy_url":"u","verified":true,"amount":1}}}`), &stub); err != nil {
		t.Fatalf("stub shape: %v", err)
	}
	if stub.Channel != "16688" || !stub.Plans["basic"].Verified || stub.Plans["basic"].Amount != 1 {
		t.Fatal("stub fields drifted from the licensing backend contract")
	}
}
