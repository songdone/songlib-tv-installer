import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import test from "node:test";
import vm from "node:vm";

const read = (rel) => readFileSync(new URL(rel, import.meta.url), "utf8");

// ---- AppShell / navigation ---------------------------------------------

test("AppShell sidebar carries the destinations and no upstream chrome", () => {
  const layout = read("../src/page/layout.vue");
  for (const route of ['home', 'devices', 'apps', 'account', 'explore', 'purchase']) {
    assert.ok(
      layout.includes(`name: "${route}"`),
      `navigation must link to ${route}`,
    );
  }
  // Sidebar navigation frame (kept from the previous iteration): desktop
  // sidebar + mobile drawer, purchase reachable as a first-level entry.
  assert.ok(layout.includes("sl-sidebar"), "sidebar navigation must exist");
  assert.ok(layout.includes("sl-sidebar--open"), "mobile drawer state must exist");
  assert.ok(layout.includes("sl-sidebar-scrim"), "drawer scrim must exist");
  assert.ok(layout.includes(`name: "settings"`), "settings stays in the sidebar");
  // The topbar GitHub entry points at the public installer repo.
  assert.ok(layout.includes("https://github.com/songdone/songlib-tv-installer"));
  // The old donate modal is gone; upstream support links survive only inside
  // the About credits, clearly attributed.
  assert.ok(!layout.includes("showDonateModal"), "donate modal must be removed");
  assert.ok(layout.includes("sponsors/bitxeno"), "upstream support links must stay in About credits");
  // Brand naming rules.
  assert.ok(layout.includes("音屿-Apple TV音乐播放器安装助手"));
  assert.ok(layout.includes("音屿 TV 安装助手"));
});

test("legacy routes and deep links stay alive", () => {
  const router = read("../src/router/index.js");
  for (const name of ['home', 'devices', 'apps', 'explore', 'add-device', 'pair', 'pair-manual', 'install', 'tools', 'laboratory', 'settings', 'account']) {
    assert.ok(
      new RegExp(`name: "${name}"`).test(router),
      `router must keep the ${name} route`,
    );
  }
});

// ---- Brand red lines ----------------------------------------------------

test("official slogan is exact in every locale", () => {
  const zh = JSON.parse(read("../../../locales/zh_cn.json"));
  assert.equal(zh.home.hero.slogan_line1 + " " + zh.home.hero.slogan_line2, "你的音乐 自成一屿");
  const en = JSON.parse(read("../../../locales/en.json"));
  assert.ok(en.home.hero.kicker.includes("SONGLIB AMP"));
});

test("no unverified commercial or capability claims in user-facing sources", () => {
  const zh = JSON.parse(read("../../../locales/zh_cn.json"));
  const en = JSON.parse(read("../../../locales/en.json"));
  const texts = JSON.stringify([zh, en]);
  // 6-device licensing is a private-backend requirement, not a live feature.
  assert.ok(!texts.includes("最多激活 6 台"), "6-device claim must not appear");
  assert.ok(!texts.includes("6 台任意"), "6-device claim must not appear");
  // The 30-day Pro trial is not live yet.
  assert.ok(!texts.includes("30 天 Pro"), "30-day Pro trial claim must not appear");
  // No "purchase channel not configured" stubs for end users.
  assert.ok(!texts.includes("购买通道尚未配置"), "config stub must not appear");
  const hero = read("../src/components/SongLibHero.vue");
  assert.ok(!hero.includes("最多激活 6 台"));
  assert.ok(!hero.includes("购买通道尚未配置"));
});

test("iPhone pairing stays behind the capability flag", () => {
  const source = read("../src/page/add-device/index.vue");
  assert.match(source, /showIphoneTab: false/, "capability flag must default to hidden");
  // Tab button and panel are gated on the flag.
  assert.ok(source.includes('v-if="showIphoneTab"'));
  assert.ok(source.includes("showIphoneTab && activeTab === 'iphone'"));
  // The underlying logic and its tests keep working: methods stay present.
  for (const fn of ["startConnect", "connectIPhone", "submitImport", "discoverTunnelPort"]) {
    assert.ok(source.includes(fn), `${fn} must remain implemented`);
  }
});

test("workbench has no large empty management table", () => {
  const home = read("../src/page/home/index.vue");
  assert.ok(!home.includes("<table"), "workbench must not render the app table");
  assert.ok(home.includes("QuickStartStepper"), "workbench must include the quick start");
  assert.ok(home.includes("HomeFooter"), "workbench must include the credits footer");
  assert.ok(home.includes("home.safety.title"), "workbench must show the safety notes");
});

test("apps page keeps card/table dual mode and real actions", () => {
  const apps = read("../src/page/apps/index.vue");
  assert.match(apps, /CARD_LIMIT: 8/);
  assert.ok(apps.includes("<table"), "apps page keeps the sortable table");
  assert.ok(apps.includes("sl-app-card"), "apps page renders row cards");
  assert.ok(apps.includes("SourceDialog"));
  for (const fn of ["refreshApp", "reinstallApp", "updateApp", "deleteApp", "checkUpdates"]) {
    assert.ok(apps.includes(fn), `${fn} must stay wired`);
  }
});

test("devices page filters and manual pairing exist", () => {
  const devices = read("../src/page/devices/index.vue");
  assert.ok(devices.includes('activeFilter === "paired"') || devices.includes('"paired"'));
  assert.ok(devices.includes("pair-manual"), "manual IP must route to the real pair flow");
  assert.ok(devices.includes("deleteManualDevice"), "forget must call the real API");
});

test("hero primary CTA opens the in-app wizard, not an external page", () => {
  const hero = read("../src/components/SongLibHero.vue");
  assert.ok(hero.includes(`:to="{ name: 'add-device' }"`));
  assert.ok(hero.includes(`:to="{ name: 'explore' }"`));
});

test("design tokens exist and are theme-complete", () => {
  const tokens = read("../src/styles/tokens.css");
  for (const token of ["--sl-gold", "--sl-accent", "--sl-bg", "--sl-surface", "--sl-border", "--sl-danger", "--sl-topbar", "--sl-heading"]) {
    assert.ok(tokens.includes(token), `tokens.css must define ${token}`);
  }
  // Brand gold is calibrated from the logo (#E3B459), not invented — and no
  // FrpDeck palette value (#E6BE7A) is copied.
  assert.ok(tokens.includes("#E3B459"), "raw logo gold must be present");
  assert.ok(!tokens.includes("#E6BE7A"), "FrpDeck gold must not be copied");
  assert.ok(existsSync(new URL("../src/styles/layout.css", import.meta.url)));
  assert.ok(existsSync(new URL("../src/styles/components.css", import.meta.url)));
});

test("purchase page keeps the FrpDeck plan language with gated commerce", () => {
  const purchase = read("../src/page/purchase/index.vue");
  assert.ok(purchase.includes("sl-plan"), "plan cards must use the FrpDeck plan pattern");
  assert.ok(purchase.includes("sl-cmp"), "comparison table must exist");
  assert.ok(purchase.includes("sl-steps"), "redemption steps must exist");
  assert.ok(purchase.includes("sl-faq"), "FAQ must exist");
  // Commerce red lines: no invented prices, no machine-code activation box.
  for (const banned of ["¥8.8", "¥18", "¥6", "机器码", "兑换码激活", "FD2.", "FDV"]) {
    assert.ok(!purchase.includes(banned), `purchase page must not contain ${banned}`);
  }
  // Purchase CTAs render only from verified config links.
  assert.match(purchase, /v-if="links\.basic"/);
  assert.match(purchase, /v-if="links\.pro"/);
  assert.ok(purchase.includes("songlib-config"), "links come from the verified config source");
});

test("index.html favicon and title follow the brand", () => {
  const html = read("../../../web/static/index.html");
  assert.ok(html.includes("/songlib-mark.png"));
  assert.ok(html.includes("音屿-Apple TV音乐播放器安装助手"));
  assert.ok(!html.includes("/img/app.svg"), "upstream favicon must be gone");
});

test("official TV screenshots are committed with provenance", () => {
  for (const f of ["playing.jpg", "lyrics.jpg", "playlist.jpg", "library.jpg", "artist.jpg"]) {
    assert.ok(
      existsSync(new URL(`../../../web/static/public/tv/${f}`, import.meta.url)),
      `web asset tv/${f} missing`,
    );
  }
  assert.ok(existsSync(new URL("../../../marketing/tv/PROVENANCE.md", import.meta.url)));
  const provenance = read("../../../marketing/tv/PROVENANCE.md");
  assert.ok(provenance.includes("songlib.playsong.cn"));
});
