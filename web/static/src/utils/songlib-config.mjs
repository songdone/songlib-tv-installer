// Shared public product configuration (single fetch, cached per session).
//
// The Go backend serves fixed trusted origins at /songlib/config and only
// forwards buy URLs that came back verified from the official site. UI
// components must not invent or hardcode extra commercial links: anything the
// backend does not send stays hidden (docs/UI_PRODUCT_REDESIGN_HANDOFF_20261104.md §6).
// The owner's 16688 storefront (same account as the FrpDeck catalog). It is
// the default purchase target; /songlib/config replaces it with the specific
// verified goods-page URL + price whenever the channel reports them.
export const SONGLIB_16688_SHOP = "https://www.16688.com.cn/shop/SONGDONE";

export const SONGLIB_DEFAULT_LINKS = {
  website: "https://songlib.playsong.cn/",
  download: "https://songlib.playsong.cn/#download-tv",
  ios: "https://songlib.playsong.cn/#download-ios",
  mac: null,
  trial: null,
  trial_days: 30,
  trial_open: false,
  basic: SONGLIB_16688_SHOP,
  pro: SONGLIB_16688_SHOP,
  // Plan prices in cents (分) — only ever filled from a verified channel
  // entry by /songlib/config; never hardcoded here.
  basic_price: null,
  pro_price: null,
  source: "/source",
  license: "/license",
};

let cache = null;

export function songlibConfig() {
  if (!cache) {
    cache = fetch("/songlib/config")
      .then((r) => (r.ok ? r.json() : {}))
      .catch(() => ({}));
  }
  return cache;
}
