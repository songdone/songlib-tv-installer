<template>
  <div class="sl-page">
    <SongLibHero />

    <!-- Quick start: full three-step guide before any TV is paired, then a
         single summary card (docs/VISUAL_SYSTEM_AND_CODEX_HANDOFF §7). -->
    <section class="sl-section-gap">
      <QuickStartStepper
        :devices="devices"
        :compact="devicesLoaded && pairedDevices.length > 0"
        :last-renewal="lastRenewal"
        @devices-refreshed="onDevicesRefreshed"
      />
    </section>

    <!-- One concise renewal-risk line when something needs attention; the
         detailed list lives in the apps view. -->
    <section class="sl-section sl-section--tight" v-if="attentionApps.length > 0">
      <div class="sl-card sl-pad sl-risk">
        <span class="sl-status sl-status--warning sl-risk-label">
          <span class="sl-status-dot" aria-hidden="true"></span>
          {{ $t("home.risk_title", { count: attentionApps.length }) }}
        </span>
        <router-link class="sl-btn sl-risk-btn" :to="{ name: 'apps' }">
          {{ $t("home.risk_open") }}
        </router-link>
      </div>
    </section>

    <!-- Device summary: real cards replace any empty placeholder. -->
    <section class="sl-section-gap" v-if="devicesLoaded && (pairedDevices.length > 0 || pairableDevices.length > 0)">
      <h2 class="sl-section-title">{{ $t("home.devices_title") }}</h2>
      <div class="sl-dev-grid">
        <DeviceStatusCard v-for="device in previewDevices" :key="device.id" :device="device" />
      </div>
      <router-link class="sl-btn sl-ghost sl-more" :to="{ name: 'devices' }">
        {{ $t("home.devices_open") }}
      </router-link>
    </section>

    <!-- Recent app / renewal snapshot: statuses from the real app list only. -->
    <section class="sl-section-gap" v-if="latestApps.length > 0">
      <h2 class="sl-section-title">{{ $t("home.apps_title") }}</h2>
      <div class="sl-card">
        <ul class="sl-apps">
          <li v-for="item in latestApps" :key="item.ID">
            <div class="sl-apps-icon">
              <img v-if="item.icon && !failedIcons[item.ID]" :src="iconUrl(item)" :alt="appName(item)" loading="lazy" @error="markIconFailed(item.ID)" />
              <span v-else class="sl-apps-initial" :aria-label="appName(item)">{{ (appName(item) || "?").charAt(0).toUpperCase() }}</span>
            </div>
            <div class="sl-apps-name">
              <span class="sl-apps-title" :title="appName(item)">{{ appName(item) }}</span>
              <span class="sl-caption">{{ item.version }}</span>
            </div>
            <span class="sl-chip" :class="expiryChipClass(item)">{{ formatExpiredTime(item) }}</span>
          </li>
        </ul>
      </div>
      <router-link class="sl-btn sl-ghost sl-more" :to="{ name: 'apps' }">
        {{ $t("home.apps_open") }}
      </router-link>
    </section>

    <!-- Pre-install safety notes: visible on the first screen, not buried. -->
    <section class="sl-section-gap">
      <h2 class="sl-section-title">{{ $t("home.safety.title") }}</h2>
      <div class="sl-card sl-pad sl-safety">
        <ul>
          <li>{{ $t("home.safety.item1") }}</li>
          <li>{{ $t("home.safety.item2") }}</li>
          <li>{{ $t("home.safety.item3") }}</li>
        </ul>
      </div>
    </section>

    <HomeFooter />
  </div>
</template>

<script>
import dayjs from "dayjs";
import api from "@/api/api";
import SongLibHero from "@/components/SongLibHero.vue";
import QuickStartStepper from "@/components/QuickStartStepper.vue";
import DeviceStatusCard from "@/components/DeviceStatusCard.vue";
import HomeFooter from "@/components/HomeFooter.vue";

export default {
  name: "Home",
  components: { SongLibHero, QuickStartStepper, DeviceStatusCard, HomeFooter },
  data() {
    return {
      devices: [],
      devicesLoaded: false,
      list: [],
      failedIcons: {},
    };
  },
  computed: {
    pairedDevices() {
      return this.devices.filter((item) => item.status === "paired");
    },
    pairableDevices() {
      return this.devices.filter((item) => {
        if (item.status !== "pairable") return false;
        const cls = (item.device_class || "").toLowerCase();
        return !cls || (cls !== "iphone" && cls !== "ipad");
      });
    },
    previewDevices() {
      return [...this.pairedDevices, ...this.pairableDevices].slice(0, 3);
    },
    // Apps whose signing window is at (or nearly) its end. The 7-day free
    // signing timeline is the tool's own concern — commercial trial periods
    // never enter this calculation.
    attentionApps() {
      return this.list.filter((item) => {
        const hours = this.expiryHours(item);
        return hours !== null && hours <= 24;
      });
    },
    latestApps() {
      return this.list.slice().sort((a, b) => this.expiryValue(b) - this.expiryValue(a)).slice(0, 4);
    },
    lastRenewal() {
      const renewed = this.list
        .filter((item) => item.refreshed_date)
        .sort((a, b) => dayjs(b.refreshed_date).valueOf() - dayjs(a.refreshed_date).valueOf());
      const latest = renewed[0];
      if (!latest) return null;
      return {
        date: dayjs(latest.refreshed_date).format("YYYY-MM-DD HH:mm"),
        success: !!latest.refreshed_result,
      };
    },
  },
  created() {
    this.fetchData();
  },
  methods: {
    fetchData() {
      api.getDevices().then((res) => {
        this.devices = res.data;
        this.devicesLoaded = true;
      }).catch(() => {
        this.devicesLoaded = true;
      });
      api.getAppList().then((res) => {
        this.list = res.data;
      }).catch(() => {
        this.list = [];
      });
    },
    onDevicesRefreshed(devices) {
      this.devices = devices;
      this.devicesLoaded = true;
    },
    isAppleTVDevice(device) {
      const cls = (device.device_class || "").toLowerCase();
      return !cls || (cls !== "iphone" && cls !== "ipad");
    },
    expiryValue(item) {
      let time = item.refreshed_date || item.installed_date;
      if (item.expiration_date) return dayjs(item.expiration_date).valueOf();
      return time ? dayjs(time).add(7, "day").valueOf() : 0;
    },
    expiryHours(item) {
      const value = this.expiryValue(item);
      if (!value) return null;
      return dayjs(value).diff(dayjs(), "hour");
    },
    expiryChipClass(item) {
      const hours = this.expiryHours(item);
      if (hours === null) return "";
      if (hours <= 0) return "sl-risk-chip-danger";
      if (hours <= 24) return "sl-risk-chip-warning";
      return "";
    },
    appName(item) {
      return item.custom_name || item.ipa_name;
    },
    iconUrl(app) {
      return `/apps/${app.ID}/icon`;
    },
    markIconFailed(id) {
      this.failedIcons[id] = true;
    },
    formatExpiredTime(item) {
      let time = item.refreshed_date || item.installed_date;
      if (!time && !item.expiration_date) return "-";

      let expired = time ? dayjs(time).add(7, "day") : null;
      if (item.expiration_date) {
        expired = dayjs(item.expiration_date);
      }

      if (expired.diff(dayjs()) <= 0) {
        return this.$t("home.table.expired_time_format.expired_tips");
      }

      const days = expired.diff(dayjs(), "day");
      if (days >= 1) {
        return this.$t("home.table.expired_time_format.days", { num: days, count: days });
      }

      const hours = expired.diff(dayjs(), "hour");
      if (hours >= 1) {
        return this.$t("home.table.expired_time_format.hours", { num: hours, count: hours });
      }

      return this.$t("home.table.expired_time_format.less_than_hour");
    },
  },
};
</script>

<style lang="postcss" scoped>
.sl-section--tight {
  margin-top: 16px;
}

.sl-risk {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  border-color: var(--sl-warning);
}

.sl-risk-label {
  font-size: 14px;
}

.sl-risk-btn {
  min-height: 38px;
  padding: 7px 16px;
  font-size: 13.5px;
}

.sl-dev-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.sl-more {
  margin-top: 10px;
}

.sl-apps {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 0;
  list-style: none;
}

.sl-apps li {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 20px;
}

.sl-apps li + li {
  border-top: 1px solid var(--sl-border);
}

.sl-apps-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 10px;
  overflow: hidden;
  background: var(--sl-surface-2);
  flex-shrink: 0;
}

.sl-apps-icon img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.sl-apps-initial {
  font-weight: 750;
  color: var(--sl-muted);
}

.sl-apps-name {
  display: flex;
  flex-direction: column;
  gap: 1px;
  flex: 1;
  min-width: 0;
}

.sl-apps-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--sl-text);
  font-size: 14px;
  font-weight: 650;
}

.sl-risk-chip-warning {
  border-color: var(--sl-warning);
  color: var(--sl-warning);
  background: var(--sl-warning-soft);
}

.sl-risk-chip-danger {
  border-color: var(--sl-danger);
  color: var(--sl-danger);
  background: var(--sl-danger-soft);
}

.sl-safety ul {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding-left: 18px;
  list-style: disc;
  color: var(--sl-muted);
  font-size: 13.5px;
  line-height: 1.65;
}
</style>
