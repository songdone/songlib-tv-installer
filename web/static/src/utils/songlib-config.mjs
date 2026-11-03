// Shared public product configuration (single fetch, cached per session).
//
// The Go backend serves fixed trusted origins at /songlib/config and only
// forwards buy URLs that came back verified from the official site. UI
// components must not invent or hardcode extra commercial links: anything the
// backend does not send stays hidden (docs/UI_PRODUCT_REDESIGN_HANDOFF_20261104.md §6).
export const SONGLIB_DEFAULT_LINKS = {
  website: "https://songlib.playsong.cn/",
  download: "https://songlib.playsong.cn/#download-tv",
  ios: "https://songlib.playsong.cn/#download-ios",
  mac: null,
  trial: null,
  basic: null,
  pro: null,
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
