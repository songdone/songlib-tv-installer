// Shared public product configuration (single fetch, cached per session).
//
// The Go backend serves fixed trusted origins at /songlib/config and only
// forwards buy URLs that came back verified from the official site. UI
// components must not invent or hardcode extra commercial links: anything the
// backend does not send stays hidden (docs/UI_PRODUCT_REDESIGN_HANDOFF_20261104.md §6).
//
// Owner-confirmed goods pages on the 16688 storefront (2026-11-04). Each plan
// points at its own product page — the storefront is never used as a stand-in
// for "the plan page", because that would land buyers on a catalogue instead
// of the product they clicked.
export const SONGLIB_PLAN_BASIC = "https://www.16688.com.cn/goods/G41709534";
export const SONGLIB_PLAN_PRO = "https://www.16688.com.cn/goods/G49510548";

// The official site's trial section: informational only. Whether the trial can
// actually be claimed is decided by trial_open + trial_url, not by this anchor.
export const SONGLIB_TRIAL_ANCHOR = "https://songlib.playsong.cn/#trial";

export const SONGLIB_DEFAULT_LINKS = {
  website: "https://songlib.playsong.cn/",
  download: "https://songlib.playsong.cn/#download-tv",
  ios: "https://songlib.playsong.cn/#download-ios",
  mac: null,
  trial: SONGLIB_TRIAL_ANCHOR,
  trial_url: null,
  trial_days: 30,
  trial_open: false,
  basic: SONGLIB_PLAN_BASIC,
  pro: SONGLIB_PLAN_PRO,
  // Plan prices in cents (分) — only ever filled from a verified channel
  // entry by /songlib/config; never hardcoded here.
  basic_price: null,
  pro_price: null,
  source: "/source",
  license: "/license",
};

const BUY_HOSTS = new Set(["www.16688.com.cn", "16688.com.cn"]);
const OFFICIAL_HOSTS = new Set(["songlib.playsong.cn", "www.songlib.playsong.cn"]);

// httpsURL returns the URL only when it is plain https on an allowed host:
// no credentials, no downgrade, no lookalike domain.
export function httpsURL(raw, hosts) {
  if (typeof raw !== "string" || raw === "") return null;
  let url;
  try {
    url = new URL(raw);
  } catch {
    return null;
  }
  if (url.protocol !== "https:") return null;
  if (url.username || url.password) return null;
  return hosts.has(url.hostname) ? url.toString() : null;
}

// formatPlanPrice renders a cents amount as a yuan string without losing the
// 角/分 that the old Math.floor(amount / 100) dropped (6800 → "68",
// 6850 → "68.50"). Anything that is not a positive amount returns null so the
// card can hide the price instead of showing "¥0".
export function formatPlanPrice(cents) {
  if (typeof cents !== "number" || !Number.isFinite(cents)) return null;
  const value = Math.trunc(cents);
  if (value <= 0) return null;
  const yuan = Math.trunc(value / 100);
  const fen = value % 100;
  return fen === 0 ? String(yuan) : `${yuan}.${String(fen).padStart(2, "0")}`;
}

// trialIsClaimable is the single gate for the trial CTA: the backend must have
// validated an official claim URL AND flagged the campaign as open.
export function trialIsClaimable(links) {
  return Boolean(links && links.trial_open === true && httpsURL(links.trial_url, OFFICIAL_HOSTS));
}

// trialClaimURL returns the validated claim link, or null when closed/invalid.
export function trialClaimURL(links) {
  return trialIsClaimable(links) ? httpsURL(links.trial_url, OFFICIAL_HOSTS) : null;
}

// trialDays keeps the documented length unless the backend sent a sane one.
export function trialDays(links) {
  const days = links && links.trial_days;
  return typeof days === "number" && Number.isInteger(days) && days > 0 && days <= 365
    ? days
    : SONGLIB_DEFAULT_LINKS.trial_days;
}

// normalizeSonglibConfig sanitises the /songlib/config payload before any
// component sees it. Missing or rejected values fall back to the shipped
// defaults, so a broken backend can never produce a dead or foreign link.
export function normalizeSonglibConfig(payload) {
  const raw = payload && typeof payload === "object" ? payload : {};
  const next = { ...SONGLIB_DEFAULT_LINKS };
  if (httpsURL(raw.website, OFFICIAL_HOSTS)) next.website = httpsURL(raw.website, OFFICIAL_HOSTS);
  if (httpsURL(raw.download, OFFICIAL_HOSTS)) next.download = httpsURL(raw.download, OFFICIAL_HOSTS);
  if (httpsURL(raw.ios, OFFICIAL_HOSTS)) next.ios = httpsURL(raw.ios, OFFICIAL_HOSTS);
  if (httpsURL(raw.mac, OFFICIAL_HOSTS)) next.mac = httpsURL(raw.mac, OFFICIAL_HOSTS);
  if (httpsURL(raw.trial, OFFICIAL_HOSTS)) next.trial = httpsURL(raw.trial, OFFICIAL_HOSTS);
  const claim = httpsURL(raw.trial_url, OFFICIAL_HOSTS);
  next.trial_url = claim;
  next.trial_open = claim !== null && raw.trial_open === true;
  next.trial_days = trialDays(raw);
  if (httpsURL(raw.basic, BUY_HOSTS)) next.basic = httpsURL(raw.basic, BUY_HOSTS);
  if (httpsURL(raw.pro, BUY_HOSTS)) next.pro = httpsURL(raw.pro, BUY_HOSTS);
  next.basic_price = formatPlanPrice(raw.basic_price) === null ? null : raw.basic_price;
  next.pro_price = formatPlanPrice(raw.pro_price) === null ? null : raw.pro_price;
  return next;
}

let cache = null;

export function songlibConfig() {
  if (!cache) {
    cache = fetch("/songlib/config")
      .then((r) => (r.ok ? r.json() : {}))
      .then(normalizeSonglibConfig)
      .catch(() => ({ ...SONGLIB_DEFAULT_LINKS }));
  }
  return cache;
}
