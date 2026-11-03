<template>
  <section class="sl-hero" aria-labelledby="sl-hero-title">
    <div class="sl-hero-inner">
      <div class="sl-hero-copy">
        <p class="sl-hero-kicker">{{ $t("home.hero.kicker") }}</p>
        <h1 id="sl-hero-title" class="sl-h1 sl-hero-title">
          <span>{{ $t("home.hero.slogan_line1") }}</span>
          <span>{{ $t("home.hero.slogan_line2") }}</span>
        </h1>
        <p class="sl-hero-desc">{{ $t("home.hero.description") }}</p>

        <ul class="sl-hero-features">
          <li>{{ $t("home.hero.feature_local") }}</li>
          <li>{{ $t("home.hero.feature_tv") }}</li>
          <li>
            {{ $t("home.hero.feature_multi") }}
            <span class="sl-chip sl-chip--pro sl-hero-pro">Pro</span>
          </li>
        </ul>

        <div class="sl-hero-ctas">
          <!-- Primary action opens the real in-app install wizard, not an
               external page (docs/VISUAL_SYSTEM_AND_CODEX_HANDOFF §6). -->
          <router-link class="sl-btn sl-pri" :to="{ name: 'add-device' }">
            {{ $t("home.hero.cta_install") }}
          </router-link>
          <router-link class="sl-btn" :to="{ name: 'explore' }">
            {{ $t("home.hero.cta_learn") }}
          </router-link>
        </div>

        <!-- Other product entry points: iOS is a product link to the official
             site only; unreleased Mac stays a plain note, never a fake button. -->
        <div class="sl-hero-others">
          <a class="sl-btn sl-ghost" :href="links.ios" target="_blank" rel="noopener noreferrer">{{ $t("home.hero.cta_ios") }}</a>
          <a v-if="links.mac" class="sl-btn sl-ghost" :href="links.mac" target="_blank" rel="noopener noreferrer">Mac</a>
          <span v-else class="sl-caption">{{ $t("home.hero.mac_pending") }}</span>
        </div>

        <!-- Purchase links only appear when the backend config returned a
             verified URL; there is deliberately no "not configured" stub. -->
        <div class="sl-hero-purchase" v-if="links.basic || links.pro">
          <span class="sl-caption">{{ $t("home.hero.purchase_title") }}</span>
          <a v-if="links.basic" class="sl-btn sl-pri sl-hero-purchase-btn" :href="links.basic" target="_blank" rel="noopener noreferrer">{{ $t("home.hero.purchase_basic") }}</a>
          <a v-if="links.pro" class="sl-btn sl-pri sl-hero-purchase-btn" :href="links.pro" target="_blank" rel="noopener noreferrer">{{ $t("home.hero.purchase_pro") }}</a>
        </div>
      </div>

      <TVPreview :shots="shots" eager :interval="7" />
    </div>
  </section>
</template>

<script setup>
import { onMounted, reactive } from "vue";
import TVPreview from "@/components/TVPreview.vue";
import { songlibConfig, SONGLIB_DEFAULT_LINKS } from "@/utils/songlib-config.mjs";

// Official SongLib TV screenshots (marketing/tv/PROVENANCE.md): now playing,
// full-screen lyrics, playlists. Real player frames — never a mocked-up UI.
const shots = [
  { src: "/tv/playing.jpg", captionKey: "home.hero.gallery_playing" },
  { src: "/tv/lyrics.jpg", captionKey: "home.hero.gallery_lyrics" },
  { src: "/tv/playlist.jpg", captionKey: "home.hero.gallery_playlist" },
];

const links = reactive({ ...SONGLIB_DEFAULT_LINKS });

onMounted(async () => {
  Object.assign(links, await songlibConfig());
});
</script>

<style lang="postcss" scoped>
.sl-hero {
  border-radius: var(--sl-radius-xl);
  padding: 2px;
  background: linear-gradient(135deg, var(--sl-gold-soft), transparent 38%, var(--sl-gold-soft));
}

.sl-hero-inner {
  display: grid;
  grid-template-columns: minmax(0, 6fr) minmax(0, 6fr);
  gap: clamp(20px, 2.6vw, 40px);
  align-items: center;
  padding: clamp(24px, 2.8vw, 40px);
  border: 1px solid var(--sl-border);
  border-radius: calc(var(--sl-radius-xl) - 2px);
  background: var(--sl-surface);
  box-shadow: var(--sl-shadow);
}

/* ---- copy column ---- */

.sl-hero-kicker {
  margin: 0 0 14px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  border: 1px solid var(--sl-border);
  border-radius: 999px;
  background: var(--sl-surface-2);
  color: var(--sl-muted);
  font-size: 0.72rem;
  font-weight: 750;
  letter-spacing: 0.09em;
  width: fit-content;
}

.sl-hero-title {
  margin: 0 0 12px;
}

.sl-hero-title span {
  display: block;
}

/* Brand gold on the second slogan line — large text only, never small copy. */
.sl-hero-title span:last-child {
  color: var(--sl-gold);
}

.sl-hero-desc {
  margin: 0 0 16px;
  max-width: 32em;
  color: var(--sl-muted);
  font-size: 14.5px;
  line-height: 1.75;
}

.sl-hero-features {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 0 0 20px;
  padding: 0;
  list-style: none;
}

.sl-hero-features li {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  border: 1px solid var(--sl-border);
  border-radius: 999px;
  background: var(--sl-surface-2);
  color: var(--sl-text);
  font-size: 12.5px;
  font-weight: 650;
}

.sl-hero-pro {
  padding: 1px 7px;
}

.sl-hero-ctas {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.sl-hero-others {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
}

.sl-hero-purchase {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px dashed var(--sl-border);
}

.sl-hero-purchase-btn {
  min-height: 38px;
  padding: 7px 16px;
  font-size: 13px;
}

/* ---- responsive ---- */

@media (max-width: 1023px) {
  .sl-hero-inner {
    grid-template-columns: 1fr;
  }
}
</style>
