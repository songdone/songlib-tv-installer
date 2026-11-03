<template>
  <div class="sl-page sl-explore">
    <header class="sl-explore-head">
      <p class="sl-explore-eyebrow">{{ $t("explore.eyebrow") }}</p>
      <h1 class="sl-h1">{{ $t("explore.title") }}</h1>
      <p class="sl-body sl-explore-intro">{{ $t("explore.intro") }}</p>
      <div class="sl-explore-sources" :aria-label="$t('explore.sources_label')">
        <span v-for="s in sources" :key="s" class="sl-chip">{{ s }}</span>
      </div>
    </header>

    <!-- Chapter 1 · Now playing (official screenshot) -->
    <section class="sl-explore-chapter">
      <div class="sl-explore-media">
        <TVPreview :shots="[{ src: '/tv/playing.jpg', captionKey: 'explore.ch1_caption' }]" :interval="0" />
      </div>
      <div class="sl-explore-copy">
        <h2 class="sl-h2">{{ $t("explore.ch1_title") }}</h2>
        <p class="sl-body">{{ $t("explore.ch1_desc") }}</p>
        <ul class="sl-explore-list">
          <li>{{ $t("explore.ch1_point1") }}</li>
          <li>{{ $t("explore.ch1_point2") }}</li>
        </ul>
      </div>
    </section>

    <!-- Chapter 2 · Synced lyrics (official screenshot) -->
    <section class="sl-explore-chapter sl-explore-chapter--flip">
      <div class="sl-explore-media">
        <TVPreview :shots="[{ src: '/tv/lyrics.jpg', captionKey: 'explore.ch2_caption' }]" :interval="0" />
      </div>
      <div class="sl-explore-copy">
        <h2 class="sl-h2">{{ $t("explore.ch2_title") }}</h2>
        <p class="sl-body">{{ $t("explore.ch2_desc") }}</p>
        <ul class="sl-explore-list">
          <li>{{ $t("explore.ch2_point1") }}</li>
          <li>{{ $t("explore.ch2_point2") }}</li>
        </ul>
      </div>
    </section>

    <!-- Chapter 3 · Library across sources (official screenshot) -->
    <section class="sl-explore-chapter">
      <div class="sl-explore-media">
        <TVPreview :shots="[{ src: '/tv/library.jpg', captionKey: 'explore.ch3_caption' }]" :interval="0" />
      </div>
      <div class="sl-explore-copy">
        <h2 class="sl-h2">{{ $t("explore.ch3_title") }}</h2>
        <p class="sl-body">{{ $t("explore.ch3_desc") }}</p>
        <ul class="sl-explore-list">
          <li>{{ $t("explore.ch3_point1") }}</li>
          <li>
            {{ $t("explore.ch3_point2") }}
            <span class="sl-chip sl-chip--pro">Pro</span>
          </li>
        </ul>
      </div>
    </section>

    <!-- Feature summary: three modules, Pro clearly marked. -->
    <section class="sl-section">
      <h2 class="sl-section-title">{{ $t("explore.features_title") }}</h2>
      <div class="sl-feature-grid">
        <div class="sl-card sl-card-pad sl-feature">
          <h3 class="sl-h3">{{ $t("explore.feature1_title") }}</h3>
          <p class="sl-body">{{ $t("explore.feature1_desc") }}</p>
        </div>
        <div class="sl-card sl-card-pad sl-feature">
          <h3 class="sl-h3">{{ $t("explore.feature2_title") }}</h3>
          <p class="sl-body">{{ $t("explore.feature2_desc") }}</p>
        </div>
        <div class="sl-card sl-card-pad sl-feature">
          <h3 class="sl-h3">
            {{ $t("explore.feature3_title") }}
            <span class="sl-chip sl-chip--pro">Pro</span>
          </h3>
          <p class="sl-body">{{ $t("explore.feature3_desc") }}</p>
        </div>
      </div>
    </section>

    <!-- Product entry points: real links only; unreleased Mac is plain text. -->
    <section class="sl-section">
      <div class="sl-card sl-card-pad sl-explore-cta">
        <div class="sl-explore-cta-text">
          <h3 class="sl-h3">{{ $t("explore.get_title") }}</h3>
          <p class="sl-body">{{ $t("explore.get_desc") }}</p>
        </div>
        <div class="sl-explore-cta-actions">
          <a class="sl-btn sl-btn--primary" :href="links.download" target="_blank" rel="noopener noreferrer">
            {{ $t("explore.get_tv") }}
          </a>
          <a class="sl-btn sl-btn--secondary" :href="links.ios" target="_blank" rel="noopener noreferrer">
            {{ $t("home.hero.cta_ios") }}
          </a>
          <a v-if="links.mac" class="sl-btn sl-btn--secondary" :href="links.mac" target="_blank" rel="noopener noreferrer">Mac</a>
          <span v-else class="sl-caption">{{ $t("home.hero.mac_pending") }}</span>
        </div>
      </div>
    </section>

    <!-- Purchase module: only rendered with verified URLs from config. -->
    <section class="sl-section" v-if="links.basic || links.pro">
      <div class="sl-card sl-card-pad sl-explore-cta">
        <div class="sl-explore-cta-text">
          <h3 class="sl-h3">{{ $t("home.hero.purchase_title") }}</h3>
        </div>
        <div class="sl-explore-cta-actions">
          <a v-if="links.basic" class="sl-btn sl-btn--gold" :href="links.basic" target="_blank" rel="noopener noreferrer">{{ $t("home.hero.purchase_basic") }}</a>
          <a v-if="links.pro" class="sl-btn sl-btn--gold" :href="links.pro" target="_blank" rel="noopener noreferrer">{{ $t("home.hero.purchase_pro") }}</a>
        </div>
      </div>
    </section>

    <HomeFooter />
  </div>
</template>

<script>
import TVPreview from "@/components/TVPreview.vue";
import HomeFooter from "@/components/HomeFooter.vue";
import { songlibConfig, SONGLIB_DEFAULT_LINKS } from "@/utils/songlib-config.mjs";

export default {
  name: "Explore",
  components: { TVPreview, HomeFooter },
  data() {
    return {
      // Server types verified in released builds (official site copy).
      sources: ["Plex", "Emby", "飞牛音乐", "极音乐", "Navidrome / Subsonic", "WebDAV"],
      links: { ...SONGLIB_DEFAULT_LINKS },
    };
  },
  mounted() {
    songlibConfig().then((config) => Object.assign(this.links, config));
  },
};
</script>

<style lang="postcss" scoped>
.sl-explore-head {
  max-width: 640px;
  margin-bottom: 32px;
}

.sl-explore-eyebrow {
  margin: 0 0 10px;
  color: var(--sl-gold);
  font-size: 12.5px;
  font-weight: 750;
  letter-spacing: 0.1em;
  text-transform: uppercase;
}

.sl-explore-intro {
  margin-top: 10px;
}

.sl-explore-sources {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}

.sl-explore-chapter {
  display: grid;
  grid-template-columns: minmax(0, 7fr) minmax(0, 5fr);
  gap: clamp(20px, 3vw, 44px);
  align-items: center;
  margin-top: 40px;
}

.sl-explore-chapter--flip .sl-explore-media {
  order: 2;
}

.sl-explore-copy h2 {
  margin-bottom: 10px;
}

.sl-explore-list {
  margin: 12px 0 0;
  padding-left: 18px;
  list-style: disc;
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--sl-text-muted);
  font-size: 14px;
  line-height: 1.65;
}

.sl-feature-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
}

.sl-feature .sl-body {
  margin-top: 8px;
}

.sl-explore-cta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.sl-explore-cta-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

@media (max-width: 1023px) {
  .sl-explore-chapter {
    grid-template-columns: 1fr;
  }

  .sl-explore-chapter--flip .sl-explore-media {
    order: 0;
  }

  .sl-feature-grid {
    grid-template-columns: 1fr;
  }
}
</style>
