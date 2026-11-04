import assert from "node:assert/strict";
import test from "node:test";

import {
  SONGLIB_DEFAULT_LINKS,
  SONGLIB_PLAN_BASIC,
  SONGLIB_PLAN_PRO,
  formatPlanPrice,
  httpsURL,
  normalizeSonglibConfig,
  trialClaimURL,
  trialDays,
  trialIsClaimable,
} from "../src/utils/songlib-config.mjs";

const BUY_HOSTS = new Set(["www.16688.com.cn", "16688.com.cn"]);
const OFFICIAL_HOSTS = new Set(["songlib.playsong.cn", "www.songlib.playsong.cn"]);

// ---- prices -------------------------------------------------------------

test("price keeps the 角/分 that Math.floor() used to drop", () => {
  assert.equal(formatPlanPrice(6800), "68");
  assert.equal(formatPlanPrice(6850), "68.50");
  assert.equal(formatPlanPrice(6805), "68.05");
  assert.equal(formatPlanPrice(5), "0.05");
  assert.equal(formatPlanPrice(12850), "128.50");
});

test("a zero, negative or non-numeric amount is never shown as ¥0", () => {
  for (const value of [0, -100, null, undefined, "", "6800", NaN, Infinity, {}]) {
    assert.equal(formatPlanPrice(value), null, `formatPlanPrice(${String(value)})`);
  }
});

test("fractional cents are truncated, not rounded up into a fake price", () => {
  assert.equal(formatPlanPrice(6800.9), "68");
});

// ---- link validation ----------------------------------------------------

test("httpsURL accepts only plain https on an allowed host", () => {
  assert.equal(
    httpsURL("https://www.16688.com.cn/goods/G41709534", BUY_HOSTS),
    "https://www.16688.com.cn/goods/G41709534",
  );
  for (const bad of [
    "http://www.16688.com.cn/goods/G1",
    "https://evil.example.com/goods/G1",
    "https://16688.com.cn.evil.example/goods/G1",
    "https://user:pass@www.16688.com.cn/goods/G1",
    "javascript:alert(1)",
    "",
    null,
    undefined,
  ]) {
    assert.equal(httpsURL(bad, BUY_HOSTS), null, `httpsURL(${String(bad)})`);
  }
});

// ---- defaults -----------------------------------------------------------

test("each plan defaults to its own 16688 goods page, not the storefront", () => {
  assert.equal(SONGLIB_DEFAULT_LINKS.basic, SONGLIB_PLAN_BASIC);
  assert.equal(SONGLIB_DEFAULT_LINKS.pro, SONGLIB_PLAN_PRO);
  assert.match(SONGLIB_PLAN_BASIC, /\/goods\/G\d+$/);
  assert.match(SONGLIB_PLAN_PRO, /\/goods\/G\d+$/);
  assert.notEqual(SONGLIB_PLAN_BASIC, SONGLIB_PLAN_PRO);
  assert.ok(!SONGLIB_DEFAULT_LINKS.basic.includes("/shop/"), "storefront is not a plan page");
});

// ---- trial gating -------------------------------------------------------

test("the trial CTA needs an open campaign AND an official claim link", () => {
  const open = {
    trial_open: true,
    trial_url: "https://songlib.playsong.cn/#trial",
    trial_days: 30,
  };
  assert.equal(trialIsClaimable(open), true);
  assert.equal(trialClaimURL(open), "https://songlib.playsong.cn/#trial");

  assert.equal(trialIsClaimable({ ...open, trial_open: false }), false);
  assert.equal(trialIsClaimable({ ...open, trial_url: null }), false);
  assert.equal(trialIsClaimable({ ...open, trial_url: "https://evil.example.com/trial" }), false);
  assert.equal(trialIsClaimable({ ...open, trial_url: "http://songlib.playsong.cn/trial" }), false);
  assert.equal(trialIsClaimable(null), false);
});

test("trial days fall back to the documented length", () => {
  assert.equal(trialDays({ trial_days: 14 }), 14);
  assert.equal(trialDays({ trial_days: 900 }), 30);
  assert.equal(trialDays({ trial_days: -1 }), 30);
  assert.equal(trialDays({}), 30);
});

// ---- payload normalisation ---------------------------------------------

test("a broken payload falls back to shipped defaults, never to a dead link", () => {
  const config = normalizeSonglibConfig({ basic: null, pro: null, basic_price: null, pro_price: null });
  assert.equal(config.basic, SONGLIB_PLAN_BASIC);
  assert.equal(config.pro, SONGLIB_PLAN_PRO);
  assert.equal(config.basic_price, null);
  assert.equal(config.pro_price, null);
  assert.equal(config.trial_open, false);
  assert.equal(config.website, "https://songlib.playsong.cn/");
});

test("a non-16688 buy link is rejected while a verified one is kept", () => {
  const config = normalizeSonglibConfig({
    basic: "https://example.com/goods/G1",
    pro: "https://www.16688.com.cn/goods/G49510548",
  });
  assert.equal(config.basic, SONGLIB_PLAN_BASIC, "foreign host must not replace the default");
  assert.equal(config.pro, "https://www.16688.com.cn/goods/G49510548");
});

test("prices survive only as positive amounts", () => {
  const config = normalizeSonglibConfig({ basic_price: 6850, pro_price: 0 });
  assert.equal(config.basic_price, 6850);
  assert.equal(config.pro_price, null);
});

test("trial_open is never trusted without a validated claim URL", () => {
  const withForeignClaim = normalizeSonglibConfig({
    trial_open: true,
    trial_url: "https://example.com/trial",
  });
  assert.equal(withForeignClaim.trial_open, false);
  assert.equal(withForeignClaim.trial_url, null);

  const official = normalizeSonglibConfig({
    trial_open: true,
    trial_url: "https://songlib.playsong.cn/#trial",
    trial_days: 30,
  });
  assert.equal(official.trial_open, true);
  assert.equal(trialClaimURL(official), "https://songlib.playsong.cn/#trial");
});

test("official-site links are validated against the official hosts", () => {
  assert.equal(
    httpsURL("https://www.songlib.playsong.cn/x", OFFICIAL_HOSTS),
    "https://www.songlib.playsong.cn/x",
  );
  assert.equal(httpsURL("https://songlib.playsong.cn.evil.com/x", OFFICIAL_HOSTS), null);
});
