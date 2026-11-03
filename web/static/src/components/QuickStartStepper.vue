<template>
  <!-- Summary card: shown on the workbench once at least one Apple TV is
       paired; the full stepper stays available on the devices page. -->
  <div v-if="compact" class="sl-card sl-card-pad sl-qs">
    <div class="sl-qs-summary">
      <div class="sl-qs-summary-text">
        <h3 class="sl-h3">{{ $t("quickstart.summary_title", { count: paired.length }) }}</h3>
        <p class="sl-body">{{ renewalLine }}</p>
      </div>
      <div class="sl-qs-summary-actions">
        <router-link class="sl-btn sl-btn--primary" :to="{ name: 'apps' }">
          {{ $t("quickstart.summary_open_apps") }}
        </router-link>
        <router-link class="sl-btn sl-btn--tertiary" :to="{ name: 'devices' }">
          {{ $t("quickstart.summary_open_devices") }}
        </router-link>
      </div>
    </div>
  </div>

  <!-- Full three-step guide: the single empty-state block for first use. -->
  <div v-else class="sl-card sl-card-pad sl-qs">
    <header class="sl-qs-head">
      <h3 class="sl-h2">{{ $t("quickstart.title") }}</h3>
      <p class="sl-body">{{ $t("quickstart.subtitle") }}</p>
    </header>

    <ol class="sl-qs-steps">
      <!-- Step 1 · Discover: live list of TVs currently in pairing mode. -->
      <li class="sl-qs-step" :class="{ 'sl-qs-step--current': currentStep === 1 }">
        <div class="sl-qs-step-head">
          <span class="sl-qs-num">1</span>
          <h4 class="sl-h3">{{ $t("quickstart.step1_title") }}</h4>
          <span v-if="currentStep === 1" class="sl-status sl-status--muted sl-qs-state">
            <span class="sl-status-dot"></span>{{ $t("quickstart.state_waiting") }}
          </span>
        </div>
        <p class="sl-body">{{ $t("quickstart.step1_desc") }}</p>

        <div class="sl-qs-found" v-if="scanning">
          <span class="loading loading-spinner loading-sm"></span>
          <span class="sl-body">{{ $t("quickstart.scanning") }}</span>
        </div>
        <template v-else>
          <div class="sl-qs-found" v-if="pairable.length > 0">
            <ul class="sl-qs-devices">
              <li v-for="device in pairable" :key="device.id">
                <span class="sl-qs-device" :title="device.ip">
                  <span class="sl-qs-device-icon"><AppleTVIcon /></span>
                  {{ device.name }}
                </span>
                <router-link class="sl-btn sl-btn--primary sl-qs-pair" :to="{ name: 'pair', params: { id: device.id } }">
                  {{ $t("quickstart.pair_now") }}
                </router-link>
              </li>
            </ul>
          </div>
          <p class="sl-caption sl-qs-hint" v-else>{{ $t("quickstart.none_found") }}</p>

          <div class="sl-qs-actions">
            <button type="button" class="sl-btn sl-btn--secondary" :disabled="scanning" @click="refresh">
              {{ $t("quickstart.rescan") }}
            </button>
            <router-link class="sl-btn sl-btn--tertiary" :to="{ name: 'add-device' }">
              {{ $t("quickstart.manual_ip") }}
            </router-link>
          </div>
        </template>
      </li>

      <!-- Step 2 · Pair: guidance for the TV side; the action lives on the
           discovered rows above and on the add-device page. -->
      <li class="sl-qs-step" :class="{ 'sl-qs-step--current': currentStep === 2 }">
        <div class="sl-qs-step-head">
          <span class="sl-qs-num">2</span>
          <h4 class="sl-h3">{{ $t("quickstart.step2_title") }}</h4>
          <span v-if="currentStep === 2" class="sl-status sl-status--muted sl-qs-state">
            <span class="sl-status-dot"></span>{{ $t("quickstart.state_pairing") }}
          </span>
        </div>
        <p class="sl-body">{{ $t("quickstart.step2_desc") }}</p>
      </li>

      <!-- Step 3 · Install: official IPA download plus the real install flow
           once a TV is paired. -->
      <li class="sl-qs-step" :class="{ 'sl-qs-step--current': currentStep === 3 }">
        <div class="sl-qs-step-head">
          <span class="sl-qs-num">3</span>
          <h4 class="sl-h3">{{ $t("quickstart.step3_title") }}</h4>
          <span v-if="currentStep === 3" class="sl-status sl-status--muted sl-qs-state">
            <span class="sl-status-dot"></span>{{ $t("quickstart.state_ready") }}
          </span>
        </div>
        <p class="sl-body">{{ $t("quickstart.step3_desc") }}</p>
        <div class="sl-qs-actions">
          <router-link v-if="paired.length > 0" class="sl-btn sl-btn--primary" :to="{ name: 'install', params: { id: paired[0].id } }">
            {{ $t("quickstart.choose_ipa") }}
          </router-link>
          <a class="sl-btn sl-btn--secondary" :href="links.download" target="_blank" rel="noopener noreferrer">
            {{ $t("quickstart.download_ipa") }}
          </a>
        </div>
      </li>
    </ol>

    <p class="sl-caption sl-qs-free">{{ $t("quickstart.free_note") }}</p>
  </div>
</template>

<script>
import api from "@/api/api";
import { songlibConfig, SONGLIB_DEFAULT_LINKS } from "@/utils/songlib-config.mjs";

export default {
  name: "QuickStartStepper",
  props: {
    // All known devices from the parent workbench; the stepper derives its
    // current state from this instead of keeping a second source of truth.
    devices: {
      type: Array,
      default: () => [],
    },
    // Summary-card mode once a TV is paired.
    compact: {
      type: Boolean,
      default: false,
    },
    // Last renewal outcome across installed apps, for the summary card.
    lastRenewal: {
      type: Object,
      default: null,
    },
  },
  data() {
    return {
      links: { ...SONGLIB_DEFAULT_LINKS },
      scanning: false,
    };
  },
  computed: {
    paired() {
      return this.devices.filter((d) => d.status === "paired");
    },
    pairable() {
      return this.devices.filter((d) => {
        if (d.status !== "pairable") return false;
        const cls = (d.device_class || "").toLowerCase();
        return !cls || (cls !== "iphone" && cls !== "ipad");
      });
    },
    currentStep() {
      if (this.paired.length > 0) return 3;
      if (this.pairable.length > 0) return 2;
      return 1;
    },
    renewalLine() {
      if (!this.lastRenewal || !this.lastRenewal.date) {
        return this.$t("quickstart.summary_no_renewal");
      }
      return this.lastRenewal.success
        ? this.$t("quickstart.summary_renewal_ok", { date: this.lastRenewal.date })
        : this.$t("quickstart.summary_renewal_fail", { date: this.lastRenewal.date });
    },
  },
  mounted() {
    songlibConfig().then((config) => Object.assign(this.links, config));
  },
  methods: {
    // Re-read the discovery list. The devices endpoint answers fast, so hold
    // the spinner briefly to keep the feedback visible.
    async refresh() {
      this.scanning = true;
      const startedAt = Date.now();
      try {
        const res = await api.getDevices();
        this.$emit("devices-refreshed", res.data || []);
      } catch (error) {
        console.error(error);
      } finally {
        const elapsed = Date.now() - startedAt;
        if (elapsed < 600) {
          await new Promise((resolve) => setTimeout(resolve, 600 - elapsed));
        }
        this.scanning = false;
      }
    },
  },
};
</script>

<script setup>
import AppleTVIcon from "@/assets/icons/appletv.svg";
</script>

<style lang="postcss" scoped>
.sl-qs-head {
  margin-bottom: 18px;
}

.sl-qs-head .sl-body {
  margin-top: 4px;
}

.sl-qs-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 16px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.sl-qs-step {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
  padding: 18px;
  border: 1px solid var(--sl-border);
  border-radius: var(--sl-radius-md);
  background: var(--sl-panel-soft);
}

.sl-qs-step--current {
  background: var(--sl-panel);
  border-color: var(--sl-gold);
  box-shadow: var(--sl-shadow-card);
}

.sl-qs-step-head {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.sl-qs-num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 999px;
  background: var(--sl-primary);
  color: var(--sl-primary-contrast);
  font-size: 0.8rem;
  font-weight: 750;
  flex-shrink: 0;
}

.sl-qs-step--current .sl-qs-num {
  background: var(--sl-gold-bright);
  color: var(--sl-brand-ink);
}

.sl-qs-state {
  margin-left: auto;
  font-size: 12px;
}

.sl-qs-step .sl-body {
  font-size: 13.5px;
}

.sl-qs-found {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.sl-qs-devices {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  margin: 0;
  padding: 0;
  list-style: none;
}

.sl-qs-devices li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--sl-border);
  border-radius: var(--sl-radius-sm);
  background: var(--sl-panel);
}

.sl-qs-device {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--sl-text);
  font-size: 13.5px;
  font-weight: 650;
}

.sl-qs-device-icon {
  display: inline-flex;
  width: 20px;
  height: 20px;
  color: var(--sl-text-muted);
  flex-shrink: 0;
}

.sl-qs-device-icon svg {
  width: 100%;
  height: 100%;
}

.sl-qs-pair {
  min-height: 34px;
  padding: 6px 14px;
  font-size: 13px;
  flex-shrink: 0;
}

.sl-qs-hint {
  line-height: 1.6;
}

.sl-qs-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: auto;
}

.sl-qs-actions .sl-btn {
  min-height: 38px;
  padding: 7px 16px;
  font-size: 13.5px;
}

.sl-qs-free {
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px dashed var(--sl-border);
  line-height: 1.6;
}

/* ---- compact summary card ---- */

.sl-qs-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.sl-qs-summary-text .sl-body {
  margin-top: 4px;
}

.sl-qs-summary-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.sl-qs-summary-actions .sl-btn {
  min-height: 38px;
  padding: 7px 16px;
  font-size: 13.5px;
}

@media (max-width: 1023px) {
  .sl-qs-steps {
    grid-template-columns: 1fr;
  }
}
</style>
