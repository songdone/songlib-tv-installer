<template>
  <div class="sl-card sl-pad sl-dev">
    <div class="sl-dev-icon" aria-hidden="true">
      <AppleTVIcon />
    </div>
    <div class="sl-dev-body">
      <h3 class="sl-h3 sl-dev-name" :title="device.name">{{ device.name }}</h3>
      <p class="sl-caption">{{ maskedIp }}</p>
      <p class="sl-status" :class="statusClass">
        <span class="sl-status-dot" aria-hidden="true"></span>
        <span v-if="device.manual">{{ $t("devices.status_added") }}</span>
        <span v-else>{{ $t("devices.status_" + device.status) }}</span>
      </p>
    </div>
    <div class="sl-dev-action">
      <router-link
        v-if="device.status === 'paired'"
        class="sl-btn sl-pri sl-dev-btn"
        :to="{ name: 'install', params: { id: device.id } }"
      >
        {{ $t("devices.action_install") }}
      </router-link>
      <router-link
        v-else-if="device.status === 'pairable'"
        class="sl-btn sl-pri sl-dev-btn"
        :to="{ name: 'pair', params: { id: device.id } }"
      >
        {{ $t("devices.action_pair") }}
      </router-link>
    </div>
  </div>
</template>

<script>
import { truncateIP } from "@/utils/utils";

export default {
  name: "DeviceStatusCard",
  props: {
    device: {
      type: Object,
      required: true,
    },
  },
  computed: {
    maskedIp() {
      return this.device.ip ? truncateIP(this.device.ip) : "—";
    },
    statusClass() {
      if (this.device.manual) return "sl-status--muted";
      if (this.device.status === "paired") return "sl-status--success";
      if (this.device.status === "pairable") return "sl-status--warning";
      return "sl-status--muted";
    },
  },
};
</script>

<script setup>
import AppleTVIcon from "@/assets/icons/appletv.svg";
</script>

<style lang="postcss" scoped>
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
  background: var(--sl-surface-2);
  color: var(--sl-muted);
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

.sl-dev-action {
  flex-shrink: 0;
}

.sl-dev-btn {
  min-height: 36px;
  padding: 6px 14px;
  font-size: 13px;
}
</style>
