<template>
  <!-- LegalCredits (docs/VISUAL_SYSTEM_AND_CODEX_HANDOFF §5): version, source,
       license and upstream attribution — never delete the upstream credits. -->
  <footer class="sl-footer">
    <div class="sl-footer-line">
      <span class="sl-footer-version">{{ $t("home.footer.version_label") }} {{ version }}</span>
      <span class="sl-footer-sep" aria-hidden="true">·</span>
      <a href="/source">{{ $t("home.footer.source") }}</a>
      <span class="sl-footer-sep" aria-hidden="true">·</span>
      <a href="/license">{{ $t("home.footer.license") }}</a>
      <span class="sl-footer-sep" aria-hidden="true">·</span>
      <a href="https://github.com/bitxeno/atvloadly" target="_blank" rel="noopener noreferrer">{{ $t("home.footer.upstream") }}</a>
    </div>
    <p class="sl-caption sl-footer-note">{{ $t("home.footer.upstream_note") }}</p>
  </footer>
</template>

<script>
import api from "@/api/api";

export default {
  name: "HomeFooter",
  data() {
    return {
      version: "…",
    };
  },
  mounted() {
    api.getVersion().then((res) => {
      this.version = res.data?.version || "unknown";
    }).catch(() => {
      this.version = "unknown";
    });
  },
};
</script>

<style lang="postcss" scoped>
.sl-footer {
  margin-top: 40px;
  padding-top: 18px;
  border-top: 1px solid var(--sl-border);
}

.sl-footer-line {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  color: var(--sl-text-muted);
  font-size: 13px;
  font-weight: 600;
}

.sl-footer-line a {
  color: var(--sl-text-muted);
  text-decoration: none;
  transition: color var(--sl-motion-fast) ease;
}

.sl-footer-line a:hover,
.sl-footer-line a:focus-visible {
  color: var(--sl-text);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.sl-footer-sep {
  color: var(--sl-border);
}

.sl-footer-version {
  color: var(--sl-text);
}

.sl-footer-note {
  margin-top: 6px;
}
</style>
