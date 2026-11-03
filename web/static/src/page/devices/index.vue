<template>
  <div class="sl-page">
    <header class="sl-devices-head">
      <h1 class="sl-h2">{{ $t("devices.title") }}</h1>
      <p class="sl-body">{{ $t("devices.subtitle") }}</p>
    </header>

    <div class="sl-devices-toolbar">
      <div class="join" role="tablist">
        <button
          v-for="filter in filters"
          :key="filter"
          type="button"
          role="tab"
          class="btn join-item sl-filter"
          :class="{ 'sl-filter--active': activeFilter === filter }"
          :aria-selected="activeFilter === filter"
          @click="activeFilter = filter"
        >
          {{ $t("devices.filter_" + filter) }}
        </button>
      </div>
      <button type="button" class="sl-btn sl-btn--secondary sl-scan" :disabled="scanning" @click="refresh">
        <span class="loading loading-spinner loading-xs" v-if="scanning"></span>
        {{ $t("devices.rescan") }}
      </button>
    </div>

    <!-- Scanning feedback: never a bare blank area. -->
    <div v-if="scanning && devices.length === 0" class="sl-card sl-empty">
      <span class="loading loading-spinner loading-md"></span>
      <h3>{{ $t("quickstart.scanning") }}</h3>
      <p>{{ $t("devices.scanning_hint") }}</p>
    </div>

    <!-- Single guided empty state: no stacked placeholder cards. -->
    <div v-else-if="filteredDevices.length === 0" class="sl-card sl-empty">
      <div class="sl-empty-icon"><AppleTVIcon /></div>
      <h3>{{ $t("devices.empty_title") }}</h3>
      <p>{{ $t("devices.empty_hint") }}</p>
      <router-link class="sl-btn sl-btn--primary" :to="{ name: 'add-device' }">
        {{ $t("devices.empty_action") }}
      </router-link>
    </div>

    <div v-else class="sl-dev-grid">
      <div v-for="device in filteredDevices" :key="device.id" class="sl-card sl-card-pad sl-dev">
        <div class="sl-dev-icon" aria-hidden="true">
          <AppleTVIcon />
        </div>
        <div class="sl-dev-body">
          <h3 class="sl-h3 sl-dev-name" :title="device.name">{{ device.name }}</h3>
          <p class="sl-caption">{{ device.ip ? truncateIP(device.ip) : "—" }}</p>
          <p class="sl-status" :class="statusClass(device)">
            <span class="sl-status-dot" aria-hidden="true"></span>
            <span v-if="device.manual">{{ $t("devices.status_added") }}</span>
            <span v-else>{{ $t("devices.status_" + device.status) }}</span>
          </p>
        </div>
        <div class="sl-dev-actions">
          <router-link
            v-if="device.status === 'paired'"
            class="sl-btn sl-btn--primary sl-dev-btn"
            :to="{ name: 'install', params: { id: device.id } }"
          >
            {{ $t("devices.action_install") }}
          </router-link>
          <router-link
            v-else-if="device.status === 'pairable'"
            class="sl-btn sl-btn--primary sl-dev-btn"
            :to="{ name: 'pair', params: { id: device.id } }"
          >
            {{ $t("devices.action_pair") }}
          </router-link>
          <button
            v-if="device.manual"
            type="button"
            class="sl-btn sl-btn--danger sl-dev-btn"
            :class="{ 'sl-dev-btn--armed': forgetConfirmId === device.id }"
            @click="forgetDevice(device)"
          >
            {{ forgetConfirmId === device.id ? $t("devices.action_forget_confirm") : $t("devices.action_forget") }}
          </button>
        </div>
      </div>
    </div>

    <!-- Manual entry: a compact secondary path, kept visible but quiet. -->
    <section class="sl-section">
      <div class="sl-card sl-card-pad sl-manual">
        <div class="sl-manual-text">
          <h3 class="sl-h3">{{ $t("devices.manual_title") }}</h3>
          <p class="sl-body">{{ $t("devices.manual_desc") }}</p>
        </div>
        <form class="sl-manual-form" @submit.prevent="openManualPair">
          <input
            type="text"
            class="sl-input"
            v-model="manualIp"
            :placeholder="$t('devices.manual_placeholder')"
            :aria-label="$t('devices.manual_title')"
          />
          <button type="submit" class="sl-btn sl-btn--secondary" :disabled="!manualIp.trim()">
            {{ $t("devices.manual_action") }}
          </button>
        </form>
      </div>
    </section>
  </div>
</template>

<script>
import api from "@/api/api";
import { toast } from "vue3-toastify";
import { truncateIP } from "@/utils/utils";

export default {
  name: "Devices",
  data() {
    return {
      devices: [],
      devicesLoaded: false,
      scanning: false,
      activeFilter: "paired",
      filters: ["paired", "pairable", "all"],
      manualIp: "",
      forgetConfirmId: null,
    };
  },
  computed: {
    filteredDevices() {
      const isTV = (device) => {
        const cls = (device.device_class || "").toLowerCase();
        return !cls || (cls !== "iphone" && cls !== "ipad");
      };
      let list = this.devices.filter(isTV);
      if (this.activeFilter === "paired") list = list.filter((d) => d.status === "paired");
      if (this.activeFilter === "pairable") list = list.filter((d) => d.status === "pairable");
      return list;
    },
  },
  created() {
    this.refresh();
  },
  methods: {
    truncateIP,
    async refresh() {
      this.scanning = true;
      const startedAt = Date.now();
      try {
        const res = await api.getDevices();
        this.devices = res.data || [];
        this.devicesLoaded = true;
      } catch (error) {
        console.error(error);
      } finally {
        // Keep the spinner readable on fast answers.
        const elapsed = Date.now() - startedAt;
        if (elapsed < 600) {
          await new Promise((resolve) => setTimeout(resolve, 600 - elapsed));
        }
        this.scanning = false;
      }
    },
    statusClass(device) {
      if (device.manual) return "sl-status--muted";
      if (device.status === "paired") return "sl-status--success";
      if (device.status === "pairable") return "sl-status--warning";
      return "sl-status--muted";
    },
    async forgetDevice(device) {
      // Two clicks: arm, then confirm.
      if (this.forgetConfirmId !== device.id) {
        this.forgetConfirmId = device.id;
        return;
      }
      this.forgetConfirmId = null;
      try {
        await api.deleteManualDevice(device.id);
        toast.success(this.$t("devices.forget_done"));
        this.refresh();
      } catch (error) {
        console.error(error);
      }
    },
    openManualPair() {
      const ip = this.manualIp.trim();
      if (!ip) return;
      this.$router.push({ name: "pair-manual", query: { ip } });
    },
  },
};
</script>

<style lang="postcss" scoped>
.sl-devices-head {
  margin-bottom: 20px;
}

.sl-devices-head .sl-body {
  margin-top: 4px;
}

.sl-devices-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.sl-filter {
  min-height: 38px;
  padding: 6px 18px;
  border: 1px solid var(--sl-border);
  background: var(--sl-panel);
  color: var(--sl-text-muted);
  font-size: 13.5px;
  font-weight: 650;
  text-transform: none;
  border-radius: 0;
}

.sl-filter:first-child {
  border-radius: var(--sl-radius-sm) 0 0 var(--sl-radius-sm);
}

.sl-filter:last-child {
  border-radius: 0 var(--sl-radius-sm) var(--sl-radius-sm) 0;
}

.sl-filter--active {
  background: var(--sl-primary);
  border-color: var(--sl-primary);
  color: var(--sl-primary-contrast);
}

.sl-scan {
  min-height: 38px;
  padding: 7px 16px;
  font-size: 13.5px;
}

.sl-dev-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.sl-dev {
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 92px;
}

.sl-dev-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 46px;
  border-radius: var(--sl-radius-md);
  background: var(--sl-panel-soft);
  color: var(--sl-text-muted);
  flex-shrink: 0;
}

.sl-dev-icon svg {
  width: 26px;
  height: 26px;
}

.sl-dev-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}

.sl-dev-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sl-dev-actions {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex-shrink: 0;
}

.sl-dev-btn {
  min-height: 34px;
  padding: 5px 14px;
  font-size: 13px;
}

.sl-dev-btn--armed {
  background: var(--sl-danger-soft);
  border-color: var(--sl-danger);
}

.sl-manual {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.sl-manual-text .sl-body {
  margin-top: 4px;
}

.sl-manual-form {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.sl-input {
  min-height: 42px;
  padding: 8px 14px;
  border: 1px solid var(--sl-border);
  border-radius: var(--sl-radius-sm);
  background: var(--sl-panel);
  color: var(--sl-text);
  font-size: 14px;
  min-width: 220px;
}

.sl-input:focus-visible {
  outline: 2px solid var(--sl-focus);
  outline-offset: 1px;
}

@media (max-width: 767px) {
  .sl-manual-form {
    width: 100%;
  }

  .sl-manual-form .sl-input {
    flex: 1;
    min-width: 0;
  }
}
</style>
