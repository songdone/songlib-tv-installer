import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";

const read = (rel) => readFileSync(new URL(rel, import.meta.url), "utf8");

test("vite build uses relative base for _web/<revision> asset resolution", () => {
  const viteConfig = read("../../vite.config.js");
  assert.match(viteConfig, /base:\s*["']\.\/["']/);
});

test("WebUpdateSection separates core upgrade from signed frontend hot update", () => {
  const src = read("../src/page/settings/components/WebUpdateSection.vue");
  assert.ok(src.includes("deploy_mode"), "must display deploy mode (fnOS vs Docker)");
  assert.ok(src.includes("core_release"), "must display core release state separately");
  assert.ok(src.includes("<dialog"), "must use native <dialog> for switch confirmation");
  assert.ok(!src.includes('role="dialog"'), "native modal <dialog> must not carry redundant role=dialog");
  assert.ok(!src.includes("aria-modal"), "native modal <dialog> must not carry redundant aria-modal");
  assert.ok(src.includes("overscroll-behavior: contain"), "modal dialog must contain scroll");
  assert.ok(src.includes("expected_active_revision"), "switch must verify expected active revision");
  assert.ok(src.includes("expected_previous_revision"), "rollback must verify expected previous revision");
  assert.ok(src.includes("/management/web-update/recovery"), "must link standalone recovery page");
});

test("ManagementPasswordSection follows accessible form and autofill contract", () => {
  const src = read("../src/page/settings/components/ManagementPasswordSection.vue");
  assert.ok(src.includes('autocomplete="current-password"'));
  assert.ok(src.includes('autocomplete="new-password"'));
  assert.ok(src.includes('aria-pressed'), "must expose accessible show/hide password toggle");
  assert.ok(src.includes("/management/password"));
  assert.ok(src.includes("updateManagementCSRF"), "must rotate cached CSRF token after password change");
  assert.ok(src.includes("font-size: max(1rem, 1em)"), "must prevent iOS zoom on form inputs");
});

test("UpdateBanner targets SongLib core updates without upstream branding", () => {
  const src = read("../src/components/UpdateBanner.vue");
  assert.ok(src.includes("songlib-dismissed-release"));
  assert.ok(src.includes("sl-update-banner"));
  assert.ok(!src.includes("atvloadly-dismissed-release"));
});
