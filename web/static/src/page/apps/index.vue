<template>
  <div class="sl-page">
    <div class="sl-page-head">
      <div>
        <h1 class="sl-h1">{{ $t("apps.title") }}</h1>
        <p class="sl-desc">{{ $t("apps.subtitle") }}</p>
      </div>
      <div class="sl-head-acts">
        <button
          type="button"
          class="sl-btn"
          v-if="hasTrackedApps"
          :disabled="checkingUpdates"
          @click="checkUpdates"
        >
          <span class="sl-spin" v-if="checkingUpdates"></span>
          {{ $t("home.source.check_updates") }}
        </button>
      </div>
    </div>

    <!-- Status summary: real numbers only; empty state stays textual. -->
    <div class="sl-apps-stats" v-if="list.length > 0">
      <div class="sl-card sl-pad sl-stat">
        <span class="sl-stat-num">{{ list.length }}</span>
        <span class="sl-caption">{{ $t("apps.stat_total") }}</span>
      </div>
      <div class="sl-card sl-pad sl-stat" :class="{ 'sl-stat--warning': expiringSoon.length > 0 }">
        <span class="sl-stat-num">{{ expiringSoon.length }}</span>
        <span class="sl-caption">{{ $t("apps.stat_expiring") }}</span>
      </div>
      <div class="sl-card sl-pad sl-stat">
        <span class="sl-status" :class="lastRenewal && !lastRenewal.success ? 'sl-status--danger' : 'sl-status--success'">
          <span class="sl-status-dot" aria-hidden="true"></span>
          {{ lastRenewal ? (lastRenewal.success ? $t("apps.stat_renewal_ok") : $t("apps.stat_renewal_fail")) : $t("apps.stat_renewal_none") }}
        </span>
        <span class="sl-caption" v-if="lastRenewal">{{ lastRenewal.date }}</span>
      </div>
    </div>

    <!-- Empty state with a path forward; no blank table. -->
    <div v-if="list.length === 0" class="sl-card sl-empty">
      <div class="sl-empty-icon"><AppsIcon /></div>
      <h3>{{ $t("apps.empty_title") }}</h3>
      <p>{{ $t("apps.empty_hint") }}</p>
      <router-link class="sl-btn sl-pri" :to="{ name: 'devices' }">
        {{ $t("apps.empty_action") }}
      </router-link>
    </div>

    <template v-else>
      <!-- ≤8 apps: lightweight row cards; more: the full sortable table. -->
      <div v-if="sortedList.length <= CARD_LIMIT" class="sl-app-list">
        <div v-for="item in sortedList" :key="item.ID" class="sl-card sl-app-card">
          <div class="sl-app-card-main">
            <div class="sl-app-card-icon">
              <span
                class="sl-app-card-badge"
                v-if="indicatorTip(item)"
                :title="indicatorTip(item)"
              >!</span>
              <img
                v-if="!failedIcons[item.ID]"
                :src="iconUrl(item)"
                :alt="appName(item)"
                loading="lazy"
                @error="markIconFailed(item.ID)"
              />
              <span v-else class="sl-app-initial" :aria-label="appName(item)">{{ (appName(item) || "?").charAt(0).toUpperCase() }}</span>
              <div class="sl-app-card-mask" v-show="isInstalling(item)">
                <span class="loading loading-spinner loading-md"></span>
              </div>
            </div>
            <div class="sl-app-card-info">
              <div class="sl-app-card-name" :title="appName(item)">{{ appName(item) }}</div>
              <div class="sl-caption">{{ item.version }}</div>
              <div class="sl-caption sl-app-card-device" v-html="formatDeviceName(item)"></div>
              <a class="sl-caption sl-app-card-log" :href="logUrl(item)" target="_blank">{{ formatRefreshDate(item) }}</a>
            </div>
            <div class="sl-app-card-side">
              <span class="sl-chip" :class="expiryChipClass(item)">{{ formatExpiredTime(item) }}</span>
              <span class="sl-caption" v-if="identityLabel(item)" :title="identityLabel(item)">{{ identityLabel(item) }}</span>
              <button
                type="button"
                class="sl-chip sl-chip--pro sl-app-card-source"
                v-if="item.source.latest_build_id"
                :title="$t('home.source.manage')"
                @click="openSourceDialog(item)"
              >
                {{ $t("home.source.available", { version: item.source.latest_version }) }}
              </button>
            </div>
          </div>
          <div class="sl-app-card-actions">
            <button
              type="button"
              class="sl-btn sl-pri sl-app-btn"
              v-if="item.source.latest_build_id"
              :disabled="isInstalling(item)"
              @click="updateApp(item)"
            >{{ $t("home.table.button.update") }}</button>
            <button
              type="button"
              class="sl-btn sl-app-btn"
              v-else
              @click="refreshApp(item)"
            >{{ isExternalApp(item) ? $t("home.table.button.reinstall") : $t("home.table.button.refresh") }}</button>
            <button
              type="button"
              class="sl-btn sl-ghost sl-app-btn"
              @click="openSourceDialog(item)"
            >{{ item.source.kind ? $t("home.source.manage") : $t("home.source.track") }}</button>
            <button
              type="button"
              class="sl-btn sl-danger sl-app-btn"
              @click="confirmDeleteId === item.ID ? deleteApp(item) : (confirmDeleteId = item.ID)"
            >{{ confirmDeleteId === item.ID ? $t("home.table.button.confirm_delete") : $t("home.table.button.delete") }}</button>
          </div>
        </div>
      </div>

      <div v-else class="sl-table-wrap">
        <table class="table table-auto static">
          <thead>
            <tr>
              <th>{{ $t("home.table.header.app") }}</th>
              <th @click="toggleSort('device')" class="cursor-pointer select-none">
                <div class="flex items-center gap-x-1">
                  {{ $t("home.table.header.device") }}
                  <span v-if="sortKey === 'device'" class="text-xs">{{ sortOrder === 'asc' ? '▲' : '▼' }}</span>
                </div>
              </th>
              <th @click="toggleSort('account')" class="cursor-pointer select-none">
                <div class="flex items-center gap-x-1">
                  {{ $t("home.table.header.account") }}
                  <span v-if="sortKey === 'account'" class="text-xs">{{ sortOrder === 'asc' ? '▲' : '▼' }}</span>
                </div>
              </th>
              <th @click="toggleSort('expired_date')" class="cursor-pointer select-none">
                <div class="flex items-center gap-x-1">
                  {{ $t("home.table.header.expired_date") }}
                  <span v-if="sortKey === 'expired_date'" class="text-xs">{{ sortOrder === 'asc' ? '▲' : '▼' }}</span>
                </div>
              </th>
              <th>{{ $t("home.table.header.operate") }}</th>
            </tr>
          </thead>
          <tbody>
            <tr class="hover" v-for="item in sortedList" :key="item.ID">
              <td>
                <div class="flex items-center gap-x-2 atv-app-cell">
                  <div class="indicator">
                    <div
                      class="tooltip tooltip-right indicator-item atv-indicator-tip"
                      :data-tip="indicatorTip(item)"
                      v-if="indicatorTip(item)"
                    >
                      <span class="badge badge-warning">!</span>
                    </div>
                    <div class="inline-flex">
                      <div class="atv-app-icon relative flex items-center justify-center">
                        <img
                          v-if="!failedIcons[item.ID]"
                          :src="iconUrl(item)"
                          :alt="appName(item)"
                          loading="lazy"
                          @error="markIconFailed(item.ID)"
                          class="atv-app-icon-img shadow-sm"
                        />
                        <span v-else class="atv-app-initial" :aria-label="appName(item)">{{ (appName(item) || "?").charAt(0).toUpperCase() }}</span>
                        <div
                          class="absolute w-full h-full top-0 flex items-center justify-center bg-[#00000066] atv-app-icon-mask"
                          v-show="isInstalling(item)"
                        >
                          <span class="loading loading-spinner loading-lg text-slate-100"></span>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="flex flex-col justify-start prose atv-app-text">
                    <div>{{ appName(item) }}</div>
                    <div class="stat-title text-sm">{{ item.version }}</div>
                    <div v-if="item.custom_identifier" class="stat-title text-xs truncate" :title="item.custom_identifier">
                      {{ item.custom_identifier }}
                    </div>
                    <div class="stat-title text-sm">
                      <a class="link link-hover stat-title font-normal" :href="logUrl(item)" :title="$t('home.table.tips.view_log')" target="_blank">{{ formatRefreshDate(item) }}</a>
                    </div>
                    <div class="flex flex-wrap items-center gap-1 atv-source-line">
                      <button
                        type="button"
                        class="badge badge-info badge-sm max-w-full atv-source-badge"
                        v-if="item.source.latest_build_id"
                        :title="$t('home.source.manage')"
                        @click="openSourceDialog(item)"
                      >
                        <span class="truncate">{{ $t("home.source.available", { version: item.source.latest_version }) }}</span>
                      </button>
                      <button
                        v-else-if="item.source.kind"
                        type="button"
                        class="atv-source-chip max-w-full"
                        :title="item.source.url"
                        @click="openSourceDialog(item)"
                      >
                        <span class="truncate">{{ item.source.version || "—" }}</span>
                        <span class="badge badge-ghost badge-xs shrink-0" v-if="item.source.auto_update">{{ $t("home.source.auto") }}</span>
                      </button>
                    </div>
                  </div>
                </div>
              </td>
              <td class="lg:break-all" v-html="formatDeviceName(item)"></td>
              <td class="lg:break-all">
                <div v-if="isExternalApp(item)" class="flex flex-col items-start gap-y-1">
                  <span class="badge atv-badge-external min-w-max">{{ $t("home.table.signing.external_badge") }}</span>
                  <div class="tooltip" :data-tip="externalErrorText(item)" v-if="externalErrorText(item)">
                    <div class="text-red-500">{{ identityName(item) }}</div>
                  </div>
                  <div v-else>{{ identityName(item) }}</div>
                </div>
                <template v-else>
                  <div class="tooltip" :data-tip="$t('home.table.tips.account_invalid')" v-if="item.refreshed_error == 1">
                    <div class="text-red-500"> {{ item.account }}</div>
                  </div>
                  <div v-else>{{ item.account }}</div>
                </template>
              </td>
              <td>
                <div class="badge atv-expiry min-w-max" :style="{ '--expiry-hue': expiryHue(item) }">
                  {{ formatExpiredTime(item) }}
                </div>
              </td>
              <td>
                <div class="flex gap-x-2 atv-app-actions">
                  <button
                    type="button"
                    class="btn atv-action atv-action--update"
                    v-if="item.source.latest_build_id"
                    :disabled="isInstalling(item)"
                    @click="updateApp(item)"
                  >{{ $t("home.table.button.update") }}</button>
                  <button
                    type="button"
                    class="btn atv-action atv-action--refresh"
                    v-else
                    @click="refreshApp(item)"
                  >{{ isExternalApp(item) ? $t("home.table.button.reinstall") : $t("home.table.button.refresh") }}</button>
                  <div
                    class="dropdown dropdown-hover dropdown-end"
                    :class="{ 'dropdown-top': index === sortedList.length - 1 }"
                    @mouseleave="resetConfirmDelete"
                  >
                    <div tabindex="0" role="button" class="btn atv-action">{{ $t("home.table.button.more") }}</div>
                    <ul tabindex="0" class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-48 gap-1">
                      <li>
                        <a @click="openSourceDialog(item)">
                          {{ item.source.kind ? $t("home.source.manage") : $t("home.source.track") }}
                        </a>
                      </li>
                      <li v-if="item.source.latest_build_id">
                        <a @click="refreshApp(item)">{{ isExternalApp(item) ? $t("home.table.button.reinstall") : $t("home.table.button.refresh") }}</a>
                      </li>
                      <li v-if="!isExternalApp(item)">
                        <a @click="reinstallApp(item)">{{ $t("home.table.button.reinstall") }}</a>
                      </li>
                      <li v-if="confirmDeleteId !== item.ID">
                        <a class="atv-menu-danger" @click="confirmDelete(item)">{{ $t("home.table.button.delete") }}</a>
                      </li>
                      <li v-else>
                        <a class="atv-menu-danger" @click="deleteApp(item)">{{ $t("home.table.button.confirm_delete") }}</a>
                      </li>
                    </ul>
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="sl-caption sl-apps-note">{{ $t("home.table.tips.footer") }}</div>
      <div class="sl-caption sl-apps-note" v-if="hasExternalApps">{{ $t("home.table.tips.footer_external") }}</div>
    </template>

    <SourceDialog
      ref="sourceDialog"
      @saved="fetchAppList"
      @update-started="onUpdateStarted"
    />
  </div>
</template>

<script>
import dayjs from "dayjs";
import api from "@/api/api";
import { toast } from "vue3-toastify";
import { truncateIP } from "@/utils/utils";
import SourceDialog from "@/components/SourceDialog.vue";
import { updateFailed } from "@/utils/source.mjs";

export default {
  name: "Apps",
  components: { SourceDialog },
  data() {
    return {
      list: [],
      devices: [],
      services: [],
      installingApps: [],
      checkInstallingTimer: null,
      newInstallToastId: null,
      sortKey: "",
      sortOrder: "asc",
      failedIcons: {},
      identities: null,
      checkingUpdates: false,
      pendingUpdates: {},
      confirmDeleteId: null,
      // Row cards up to this many apps; the sortable table takes over beyond
      // it (docs/VISUAL_SYSTEM_AND_CODEX_HANDOFF §5 AppList).
      CARD_LIMIT: 8,
    };
  },
  computed: {
    hasTrackedApps() {
      return this.list.some((item) => item.source.kind);
    },
    sortedList() {
      let list = this.list.slice();
      if (this.sortKey) {
        let order = this.sortOrder === "asc" ? 1 : -1;
        list.sort((a, b) => {
          let valA = this.getSortValue(a, this.sortKey);
          let valB = this.getSortValue(b, this.sortKey);
          if (valA < valB) return -1 * order;
          if (valA > valB) return 1 * order;
          return 0;
        });
      }
      return list;
    },
    hasExternalApps() {
      return this.list.some((item) => this.isExternalApp(item));
    },
    expiringSoon() {
      return this.list.filter((item) => {
        const hours = this.expiryHours(item);
        return hours !== null && hours <= 7 * 24;
      });
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
  unmounted() {
    this.checkInstallingTimer && clearTimeout(this.checkInstallingTimer);
    if (this.newInstallToastId) {
      toast.update(this.newInstallToastId, {
        type: "success",
        isLoading: false,
        autoClose: 3000,
      });
      this.newInstallToastId = null;
    }
  },
  methods: {
    fetchData() {
      let _this = this;

      api.getDevices().then((res) => {
        _this.devices = res.data;
      });

      api.getSigningIdentities().then((res) => {
        _this.identities = res.data || [];
      }).catch(() => {
        _this.identities = null;
      });

      _this.checkInstallingApp();
      _this.fetchAppList();
    },
    fetchAppList() {
      let _this = this;

      api.getAppList().then((res) => {
        _this.list = res.data;
      });
    },
    checkInstallingApp() {
      let _this = this;

      api.getInstallingApps().then((res) => {
        if (_this.installingApps && (res.data || []).length == 0) {
          _this.fetchAppList();

          if (_this.newInstallToastId) {
            toast.remove(_this.newInstallToastId);
            _this.newInstallToastId = null;
          }
        }

        _this.installingApps = res.data || [];

        let newInstall = (_this.installingApps || []).find(a => a.ID == 0);
        if (newInstall) {
          if (!_this.newInstallToastId) {
            _this.newInstallToastId = toast.loading(
              _this.$t("home.toast.installing_app", { name: newInstall.ipa_name }),
              { autoClose: false }
            );
          }
        }

        if (_this.installingApps && _this.installingApps.length > 0) {
          _this.checkInstallingAppDelay();
        }
      });
    },
    checkInstallingAppDelay() {
      let _this = this;

      if (this.checkInstallingTimer) {
        clearTimeout(this.checkInstallingTimer);
      }

      this.checkInstallingTimer = setTimeout(function () {
        _this.checkInstallingApp();
      }, 10 * 1000);
    },
    confirmDelete(item) {
      this.confirmDeleteId = item.ID;
    },
    resetConfirmDelete() {
      this.confirmDeleteId = null;
    },
    deleteApp(item) {
      let _this = this;

      _this.confirmDeleteId = null;
      api.deleteApp(item.ID).then((res) => {
        _this.fetchData();
      });
    },
    refreshApp(item) {
      let _this = this;

      _this.installingApps.push(item);

      api.refreshApp(item.ID).then((res) => {
        _this.checkInstallingAppDelay();
        if (_this.isExternalApp(item)) {
          toast.info(this.$t("home.toast.reinstall_app_started", { name: item.ipa_name }));
          return;
        }
        toast.info(this.$t("home.toast.refresh_app_started", { name: item.ipa_name }));
      });
    },
    reinstallApp(item) {
      let _this = this;

      _this.installingApps.push(item);

      api.reinstallApp(item.ID).then((res) => {
        _this.checkInstallingAppDelay();
        toast.info(this.$t("home.toast.reinstall_app_started", { name: item.ipa_name }));
      });
    },
    isExternalApp(item) {
      return item.signing_mode === "external_certificate";
    },
    identityName(item) {
      if (!this.identities) {
        return "—";
      }
      const identity = this.identities.find((entry) => entry.id === item.signing_identity_id);
      return identity ? identity.name : this.$t("home.table.signing.identity_missing");
    },
    // Short identity label for the card layout (external certs only; Apple
    // ID accounts keep their own field).
    identityLabel(item) {
      if (!this.isExternalApp(item)) return "";
      if (!this.identities) return "";
      const identity = this.identities.find((entry) => entry.id === item.signing_identity_id);
      return identity ? `${this.$t("home.table.signing.external_badge")} · ${identity.name}` : this.$t("home.table.signing.identity_missing");
    },
    externalErrorText(item) {
      switch (item.refreshed_error) {
        case 2:
          return this.$t("home.table.tips.signing_identity_error");
        case 3:
          return this.$t("home.table.tips.signing_error");
        case 4:
          return this.$t("home.table.tips.transport_error");
        default:
          return "";
      }
    },
    updateApp(item) {
      this.pendingUpdates[item.ID] = true;
      api
        .updateAppFromSource(item.ID, {})
        .then((res) => {
          this.onUpdateStarted(item, res.data);
        })
        .catch((err) => {
          console.error(err);
        })
        .finally(() => {
          delete this.pendingUpdates[item.ID];
        });
    },
    onUpdateStarted(item, data) {
      this.installingApps.push(item);
      this.checkInstallingAppDelay();
      toast.info(
        this.$t("home.toast.update_started", {
          name: this.appName(item),
          version: data.version,
        })
      );
    },
    openSourceDialog(item) {
      this.$refs.sourceDialog.show(item);
    },
    sourceWarning(item) {
      const source = item.source;
      if (source.check_error) return source.check_error;
      return updateFailed(source) ? this.$t("home.source.update_failed", { version: source.latest_version }) : "";
    },
    refreshErrorTip(item) {
      if (this.isExternalApp(item)) return this.externalErrorText(item);
      if (item.refreshed_error == 1) {
        return this.$t("home.table.tips.account_invalid");
      }
      if (item.refreshed_error == 5) {
        return this.$t("home.table.tips.certificate_limit");
      }
      if (!item.refreshed_result && item.refreshed_date) {
        return this.$t("home.table.tips.refresh_failed");
      }
      return "";
    },
    indicatorTip(item) {
      const tips = [];
      const refreshTip = this.refreshErrorTip(item);
      if (refreshTip) tips.push(refreshTip);
      const warning = this.sourceWarning(item);
      if (warning) tips.push(warning);
      return tips.join("\n");
    },
    checkUpdates() {
      this.checkingUpdates = true;
      api
        .checkSourceUpdates()
        .then((res) => {
          this.list = res.data || [];
          const tracked = this.list.filter((item) => item.source.kind);
          const updates = tracked.filter((item) => item.source.latest_build_id).length;
          const errors = tracked.filter((item) => item.source.check_error).length;
          if (updates > 0) {
            toast.info(this.$t("home.toast.updates_available", { count: updates }));
          } else if (errors > 0) {
            toast.warning(this.$t("home.toast.check_errors", { count: errors }));
          } else {
            toast.success(this.$t("home.toast.check_done"));
          }
        })
        .catch((err) => {
          console.error(err);
        })
        .finally(() => {
          this.checkingUpdates = false;
        });
    },
    appName(item) {
      return item.custom_name || item.ipa_name;
    },
    expiryHours(item) {
      const value = this.expiryValue(item);
      if (!value) return null;
      return dayjs(value).diff(dayjs(), "hour");
    },
    expiryValue(item) {
      let time = item.refreshed_date || item.installed_date;
      if (item.expiration_date) return dayjs(item.expiration_date).valueOf();
      return time ? dayjs(time).add(7, "day").valueOf() : 0;
    },
    expiryHue(item) {
      const date = item.expiration_date || item.refreshed_date || item.installed_date;
      if (!date) return null;
      const expires = item.expiration_date ? dayjs(date) : dayjs(date).add(7, "day");
      if (!expires.isValid()) return null;

      const remainingMs = expires.diff(dayjs());
      const sevenDaysMs = 7 * 24 * 60 * 60 * 1000;
      const fraction = Math.max(0, Math.min(1, remainingMs / sevenDaysMs));
      return Math.round(fraction * 120);
    },
    expiryChipClass(item) {
      const hours = this.expiryHours(item);
      if (hours === null) return "";
      if (hours <= 0) return "sl-chip-expired";
      if (hours <= 24) return "sl-chip-warning";
      return "";
    },
    formatExpiredTime(item) {
      let time = item.refreshed_date || item.installed_date;
      if (!time && !item.expiration_date) return "-";

      let expired_date = time ? dayjs(time).add(7, "day") : null;
      if (item.expiration_date) {
        expired_date = dayjs(item.expiration_date);
      }

      if (expired_date.diff(dayjs()) <= 0) {
        return this.$t("home.table.expired_time_format.expired_tips");
      }

      let days = expired_date.diff(dayjs(), "day");
      if (days >= 1) {
        return this.$t("home.table.expired_time_format.days", { num: days, count: days });
      }

      let hours = expired_date.diff(dayjs(), "hour");
      if (hours >= 1) {
        return this.$t("home.table.expired_time_format.hours", { num: hours, count: hours });
      }

      return this.$t("home.table.expired_time_format.less_than_hour");
    },
    formatRefreshDate(item) {
      let _this = this;
      if (_this.isInstalling(item)) {
        return _this.$t("home.table.refresh_date_format.installing_tips");
      }

      if (!item.refreshed_date) return "-";

      let seconds = dayjs().diff(dayjs(item.refreshed_date), "second");
      if (seconds < 60) {
        return _this.$t("home.table.refresh_date_format.seconds", { num: seconds, count: seconds });
      }
      let miniutes = parseInt(seconds / 60, 10);
      if (miniutes < 60) {
        return _this.$t("home.table.refresh_date_format.miniutes", { num: miniutes, count: miniutes });
      }
      let hours = parseInt(seconds / 3600, 10);
      if (hours < 24) {
        return _this.$t("home.table.refresh_date_format.hours", { num: hours, count: hours });
      }
      let days = parseInt(seconds / 24 / 3600, 10);
      return _this.$t("home.table.refresh_date_format.days", { num: days, count: days });
    },
    getSortValue(item, key) {
      if (key === "device") {
        for (let i = 0; i < this.devices.length; i++) {
          const dev = this.devices[i];
          if (dev.udid == item.udid && dev.status == "paired") {
            return dev.name;
          }
        }
        return "";
      } else if (key === "account") {
        return item.account || "";
      } else if (key === "expired_date") {
        return this.expiryValue(item);
      }
      return "";
    },
    toggleSort(key) {
      if (this.sortKey === key) {
        this.sortOrder = this.sortOrder === "asc" ? "desc" : "asc";
      } else {
        this.sortKey = key;
        this.sortOrder = "asc";
      }
    },
    formatDeviceName(item) {
      let _this = this;
      for (let i = 0; i < _this.devices.length; i++) {
        const dev = _this.devices[i];
        if (dev.udid == item.udid && dev.status == "paired") {
          return `${dev.name}<br/>(${truncateIP(dev.ip)})`;
        }
      }
      return this.$t("home.sidebar.device_status.unpaired");
    },
    isInstalling(item) {
      if (this.pendingUpdates[item.ID]) return true;
      if (!this.installingApps || this.installingApps.length == 0) return false;

      for (let i = 0; i < this.installingApps.length; i++) {
        if (this.installingApps[i].ID == item.ID) {
          return true;
        }
      }
      return false;
    },
    markIconFailed(id) {
      this.failedIcons[id] = true;
    },
    iconUrl(app) {
      if (app.icon) {
        return `/apps/${app.ID}/icon`;
      } else {
        return "/img/dummy.jpg";
      }
    },
    logUrl(item) {
      return `/apps/${item.ID}/log`;
    },
  },
};
</script>

<script setup>
import AppsIcon from "@/assets/icons/apps.svg";
</script>

<style lang="postcss" scoped>
.sl-apps-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.sl-apps-head .sl-body {
  margin-top: 4px;
}

.sl-apps-check {
  min-height: 38px;
  padding: 7px 16px;
  font-size: 13.5px;
}

.sl-apps-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.sl-stat {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sl-stat-num {
  font-size: 26px;
  font-weight: 760;
  color: var(--sl-text);
  line-height: 1.1;
}

.sl-stat--warning {
  border-color: var(--sl-warning);
}

.sl-stat--warning .sl-stat-num {
  color: var(--sl-warning);
}

.sl-chip-warning {
  border-color: var(--sl-warning);
  color: var(--sl-warning);
  background: var(--sl-warning-soft);
}

.sl-chip-expired {
  border-color: var(--sl-danger);
  color: var(--sl-danger);
  background: var(--sl-danger-soft);
}

/* ---- card mode ---- */

.sl-app-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sl-app-card {
  padding: 16px 20px;
}

.sl-app-card-main {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.sl-app-card-icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 12px;
  overflow: hidden;
  background: var(--sl-surface-2);
  flex-shrink: 0;
}

.sl-app-card-icon img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}

.sl-app-initial {
  font-size: 1.2rem;
  font-weight: 750;
  color: var(--sl-muted);
}

.sl-app-card-badge {
  position: absolute;
  top: 4px;
  right: 4px;
  z-index: 2;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 999px;
  background: var(--sl-warning);
  color: #fff;
  font-size: 10px;
  font-weight: 800;
}

.sl-app-card-mask {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(13, 16, 20, .45);
  z-index: 1;
}

.sl-app-card-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.sl-app-card-name {
  color: var(--sl-text);
  font-size: 15px;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sl-app-card-device {
  color: var(--sl-muted);
}

.sl-app-card-log {
  color: var(--sl-muted);
  text-decoration: none;
}

.sl-app-card-log:hover {
  color: var(--sl-text);
  text-decoration: underline;
}

.sl-app-card-side {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  flex-shrink: 0;
  max-width: 40%;
}

.sl-app-card-source {
  cursor: pointer;
  border-color: var(--sl-success);
  color: var(--sl-success);
  background: var(--sl-success-soft);
}

.sl-app-card-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px dashed var(--sl-border);
}

.sl-app-btn {
  min-height: 34px;
  padding: 5px 14px;
  font-size: 13px;
}

/* ---- table mode (inherited legacy classes styled by app.css) ---- */

.sl-table-wrap {
  overflow-x: auto;
  min-width: 0;
  max-width: 100%;
}

.sl-apps-note {
  margin-top: 10px;
  line-height: 1.6;
}

/* Expiry hue chip (table mode): reuse the legacy gradient coloring. */
.atv-expiry {
  background: hsl(var(--expiry-hue, 120), 62%, 92%);
  color: hsl(var(--expiry-hue, 120), 45%, 26%);
}

html[data-theme="dark"] .atv-expiry {
  background: hsl(var(--expiry-hue, 120), 30%, 22%);
  color: hsl(var(--expiry-hue, 120), 55%, 75%);
}

:deep(.popper) {
  background: var(--sl-surface);
  padding: 12px;
  border-radius: 12px;
  border: 1px solid var(--sl-border);
  box-shadow: var(--sl-shadow-hover);
  color: var(--sl-text);
  word-break: break-all;
  text-align: justify;
  min-width: 150px;
}

.menu li > .atv-menu-danger {
  color: var(--sl-danger);
}

.menu li > .atv-menu-danger:hover,
.menu li > .atv-menu-danger:focus-visible {
  color: var(--sl-danger);
  background-color: var(--sl-danger-soft);
}

.atv-indicator-tip {
  position: absolute;
}

.atv-indicator-tip::before {
  white-space: pre-line;
  text-align: left;
  overflow-wrap: anywhere;
}
</style>
