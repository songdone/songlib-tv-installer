<template>
  <div v-if="visible" class="sl-card sl-pad sl-update-banner" role="status" aria-live="polite">
    <div class="sl-update-banner-text">
      <span class="sl-chip sl-chip--pro">{{ $t("settings.web_update.core_badge") }}</span>
      <span class="sl-body">
        {{ $t("home.release_banner.message", { version: release.latest, current: release.current }) }}
      </span>
    </div>
    <div class="sl-update-banner-actions">
      <router-link class="sl-btn" :to="{ name: 'settings' }">
        {{ $t("settings.web_update.upgrade_guide_btn") }}
      </router-link>
      <a class="sl-btn sl-pri" :href="release.url" target="_blank" rel="noopener noreferrer">
        {{ $t("home.release_banner.view") }}
      </a>
      <button type="button" class="sl-btn sl-ghost" @click="dismiss">
        {{ $t("home.release_banner.dismiss") }}
      </button>
    </div>
  </div>
</template>

<script>
import api from "@/api/api";

const dismissedKey = "songlib-dismissed-release";

function readDismissed() {
  try {
    return window.localStorage.getItem(dismissedKey);
  } catch {
    return null;
  }
}

export default {
  name: "UpdateBanner",
  data() {
    return {
      release: null,
      dismissed: readDismissed(),
    };
  },
  computed: {
    visible() {
      return !!this.release?.available && this.release.latest !== this.dismissed;
    },
  },
  created() {
    api.getLatestRelease().then((res) => {
      this.release = res.data;
    }).catch(() => {});
  },
  methods: {
    dismiss() {
      this.dismissed = this.release.latest;
      try {
        window.localStorage.setItem(dismissedKey, this.dismissed);
      } catch {
        // Without storage the banner stays hidden for this page load only.
      }
    },
  },
};
</script>

<style lang="postcss" scoped>
.sl-update-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  flex-wrap: wrap;
  margin-bottom: 18px;
}

.sl-update-banner-text {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;
  flex: 1;
}

.sl-update-banner-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
