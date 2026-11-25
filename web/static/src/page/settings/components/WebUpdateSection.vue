<template>
  <section class="section bg-base-100">
    <h2 class="atv-section-heading">{{ $t("settings.web_update.title") }}</h2>

    <div class="sl-wu-grid">
      <div class="sl-wu-row">
        <span class="sl-wu-label">{{ $t("settings.web_update.deploy_mode_label") }}</span>
        <span class="sl-chip">{{ deployModeLabel }}</span>
      </div>
      <div class="sl-wu-row">
        <span class="sl-wu-label">{{ $t("settings.web_update.core_version_label") }}</span>
        <span class="sl-wu-value">
          <code>{{ status.installed_backend_version || "—" }}</code>
          <span v-if="status.core_release && status.core_release.available" class="sl-chip sl-chip--pro">
            {{ $t("settings.web_update.core_available", { version: status.core_release.latest }) }}
          </span>
        </span>
      </div>
      <div class="sl-wu-row">
        <span class="sl-wu-label">{{ $t("settings.web_update.ui_version_label") }}</span>
        <span class="sl-wu-value">
          <code>{{ status.active_version || "—" }}</code>
          <span class="sl-caption">
            {{ status.can_restore ? $t("settings.web_update.ui_mode_hot") : $t("settings.web_update.ui_mode_packaged") }}
          </span>
        </span>
      </div>
      <div class="sl-wu-row">
        <span class="sl-wu-label">{{ $t("settings.web_update.backend_id_label") }}</span>
        <span class="sl-wu-value">
          <code class="sl-wu-code">{{ shortBackendID }}</code>
          <button type="button" class="sl-btn sl-ghost sl-wu-copy" @click="copyBackendID" :disabled="!status.backend_id">
            {{ $t("settings.web_update.copy_id") }}
          </button>
        </span>
      </div>
    </div>

    <div v-if="status.core_release && status.core_release.available" class="sl-wu-core-box">
      <p class="sl-body">
        <strong>{{ $t("settings.web_update.core_upgrade_title", { version: status.core_release.latest }) }}</strong>
      </p>
      <p class="sl-caption">{{ coreUpgradeHint }}</p>
      <a class="sl-link" :href="status.core_release.url" target="_blank" rel="noopener noreferrer">
        {{ $t("home.release_banner.view") }}
      </a>
    </div>

    <p class="sl-caption sl-wu-boundary">{{ $t("settings.web_update.boundary_hint") }}</p>

    <div v-if="status.available" class="sl-wu-available">
      <div class="sl-wu-available-head">
        <strong>{{ $t("settings.web_update.available_title", { version: status.available.version }) }}</strong>
        <span class="sl-caption">#{{ status.available.sequence }}</span>
      </div>
      <p v-if="status.available.notes" class="sl-caption">{{ status.available.notes }}</p>
      <p v-if="!status.available.compatible" class="sl-wu-warn">{{ status.available.reason }}</p>
      <div class="sl-wu-actions" v-else-if="status.available.revision !== status.active_revision">
        <button
          type="button"
          class="sl-btn sl-pri"
          :disabled="busy || status.busy"
          @click="openConfirm('apply')"
        >
          {{ $t("settings.web_update.btn_apply") }}
        </button>
      </div>
      <p v-else class="sl-caption">{{ $t("settings.web_update.already_latest") }}</p>
    </div>

    <div class="sl-wu-actions">
      <button
        type="button"
        class="sl-btn sl-pri"
        :disabled="busy || status.busy || !status.configured"
        @click="checkNow"
      >
        <span class="sl-spin" v-if="pendingAction === 'check'"></span>
        {{ $t("settings.web_update.btn_check") }}
      </button>
      <button
        type="button"
        class="sl-btn"
        :disabled="busy"
        @click="refreshStatus"
      >
        {{ $t("settings.web_update.btn_refresh") }}
      </button>
      <button
        v-if="status.can_rollback"
        type="button"
        class="sl-btn"
        :disabled="busy || status.busy"
        @click="openConfirm('rollback')"
      >
        {{ $t("settings.web_update.btn_rollback", { version: status.previous_version || status.packaged_version }) }}
      </button>
      <button
        v-if="status.can_restore"
        type="button"
        class="sl-btn sl-ghost"
        :disabled="busy || status.busy"
        @click="openConfirm('restore')"
      >
        {{ $t("settings.web_update.btn_restore") }}
      </button>
      <a class="sl-link sl-wu-recovery-link" :href="status.recovery_path || '/management/web-update/recovery'">
        {{ $t("settings.web_update.recovery_link") }}
      </a>
    </div>

    <p v-if="feedback" class="sl-wu-feedback" :class="{ 'sl-wu-warn': feedbackError }" role="status" aria-live="polite">
      {{ feedback }}
    </p>
    <p v-else-if="status.reason" class="sl-caption sl-wu-feedback" role="status" aria-live="polite">
      {{ status.reason }}
    </p>

    <dialog ref="confirmDialog" class="sl-dialog" aria-labelledby="sl-wu-dialog-title" @close="onDialogClose">
      <h1 id="sl-wu-dialog-title" class="sl-dialog-title">{{ dialogTitle }}</h1>
      <p class="sl-body sl-dialog-body">{{ dialogBody }}</p>
      <form method="dialog" class="sl-dialog-actions">
        <button type="submit" value="cancel" class="sl-btn" autofocus>{{ $t("settings.web_update.dialog_cancel") }}</button>
        <button type="submit" value="confirm" class="sl-btn sl-pri">{{ $t("settings.web_update.dialog_confirm") }}</button>
      </form>
    </dialog>
  </section>
</template>

<script>
import api from "@/api/api";
import { toast } from "vue3-toastify";

export default {
  name: "WebUpdateSection",
  data() {
    return {
      status: {
        configured: false,
        installed_backend_version: "",
        backend_id: "",
        packaged_version: "",
        active_version: "",
        active_revision: "",
        previous_version: "",
        previous_revision: "",
        can_rollback: false,
        can_restore: false,
        busy: false,
        available: null,
        reason: "",
        deploy_mode: "docker",
        core_release: null,
        recovery_path: "/management/web-update/recovery",
      },
      pendingAction: "",
      dialogAction: "",
      feedback: "",
      feedbackError: false,
    };
  },
  computed: {
    busy() {
      return this.pendingAction !== "";
    },
    shortBackendID() {
      const id = this.status.backend_id || "";
      return id.length > 16 ? `${id.slice(0, 16)}…` : id || "—";
    },
    deployModeLabel() {
      const mode = this.status.deploy_mode || "standalone";
      return this.$t(`settings.web_update.deploy_mode_${mode}`);
    },
    coreUpgradeHint() {
      if (this.status.deploy_mode === "fnos") {
        return this.$t("settings.web_update.core_hint_fnos");
      }
      return this.$t("settings.web_update.core_hint_docker");
    },
    dialogTitle() {
      if (this.dialogAction === "apply") return this.$t("settings.web_update.confirm_apply_title");
      if (this.dialogAction === "rollback") return this.$t("settings.web_update.confirm_rollback_title");
      return this.$t("settings.web_update.confirm_restore_title");
    },
    dialogBody() {
      if (this.dialogAction === "apply") {
        return this.$t("settings.web_update.confirm_apply_body", {
          version: this.status.available?.version || "",
        });
      }
      if (this.dialogAction === "rollback") {
        return this.$t("settings.web_update.confirm_rollback_body", {
          version: this.status.previous_version || this.status.packaged_version || "",
        });
      }
      return this.$t("settings.web_update.confirm_restore_body");
    },
  },
  created() {
    this.refreshStatus();
  },
  methods: {
    async refreshStatus() {
      try {
        const res = await api.getWebUpdateStatus();
        if (res?.data) {
          this.status = res.data;
        }
      } catch (err) {
        this.feedbackError = true;
        this.feedback = err?.result?.msg || err?.message || this.$t("settings.web_update.err_status");
      }
    },
    async checkNow() {
      this.pendingAction = "check";
      this.feedback = "";
      this.feedbackError = false;
      try {
        const res = await api.checkWebUpdate();
        if (res?.data) {
          this.status = res.data;
        }
        if (this.status.available?.compatible && this.status.available.revision !== this.status.active_revision) {
          this.feedback = this.$t("settings.web_update.found_update", { version: this.status.available.version });
        } else if (this.status.available && !this.status.available.compatible) {
          this.feedbackError = true;
          this.feedback = this.status.available.reason;
        } else {
          this.feedback = this.$t("settings.web_update.already_latest");
        }
      } catch (err) {
        this.feedbackError = true;
        this.feedback = err?.result?.msg || this.$t("settings.web_update.err_check");
      } finally {
        this.pendingAction = "";
      }
    },
    openConfirm(action) {
      this.dialogAction = action;
      const dlg = this.$refs.confirmDialog;
      if (dlg && typeof dlg.showModal === "function") {
        dlg.returnValue = "";
        dlg.showModal();
      } else {
        this.executeAction(action);
      }
    },
    onDialogClose() {
      const dlg = this.$refs.confirmDialog;
      const choice = dlg ? dlg.returnValue : "";
      if (dlg) dlg.returnValue = "";
      const action = this.dialogAction;
      this.dialogAction = "";
      if (choice === "confirm" && action) {
        this.executeAction(action);
      }
    },
    async executeAction(action) {
      this.pendingAction = action;
      this.feedback = "";
      this.feedbackError = false;
      const expectedActive = this.status.active_revision || "";
      const expectedPrevious = this.status.previous_revision || "";
      const targetRevision = action === "apply" ? this.status.available?.revision || "" : "";

      try {
        let res;
        if (action === "apply") {
          res = await api.applyWebUpdate({
            revision: targetRevision,
            expected_active_revision: expectedActive,
          });
        } else if (action === "rollback") {
          res = await api.rollbackWebUpdate({
            expected_active_revision: expectedActive,
            expected_previous_revision: expectedPrevious,
          });
        } else if (action === "restore") {
          res = await api.restoreWebUpdate({
            expected_active_revision: expectedActive,
          });
        }
        if (res?.data) {
          this.status = res.data;
          const expectedAfter = action === "apply" ? targetRevision : action === "rollback" ? expectedPrevious : "";
          if (this.status.active_revision === expectedAfter) {
            toast.success(this.$t("settings.web_update.switched_reloading"));
            window.setTimeout(() => window.location.reload(), 600);
            return;
          }
        }
        this.feedback = this.$t("settings.web_update.verify_state");
      } catch (err) {
        // Read back local status on any error/timeout rather than guessing state.
        await this.refreshStatus();
        this.feedbackError = true;
        this.feedback = err?.result?.msg || this.$t("settings.web_update.err_action_readback");
      } finally {
        this.pendingAction = "";
      }
    },
    async copyBackendID() {
      if (!this.status.backend_id) return;
      try {
        await navigator.clipboard.writeText(this.status.backend_id);
        toast.success(this.$t("settings.web_update.copied"));
      } catch {
        toast.info(this.status.backend_id);
      }
    },
  },
};
</script>

<style lang="postcss" scoped>
.sl-wu-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 14px;
}

.sl-wu-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.sl-wu-label {
  font-size: 13.5px;
  color: var(--sl-muted);
}

.sl-wu-value {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.sl-wu-code {
  font-size: 12.5px;
  padding: 2px 6px;
  border-radius: 6px;
  background: var(--sl-bg-soft);
}

.sl-wu-copy {
  min-height: 28px;
  padding: 2px 10px;
  font-size: 12px;
}

.sl-wu-core-box,
.sl-wu-available {
  padding: 12px 14px;
  border-radius: 12px;
  border: 1px solid var(--sl-border);
  background: var(--sl-bg-soft);
  margin: 12px 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sl-wu-available-head {
  display: flex;
  align-items: center;
  gap: 8px;
}

.sl-wu-boundary {
  margin: 8px 0 14px;
}

.sl-wu-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.sl-wu-recovery-link {
  font-size: 13px;
  margin-inline-start: auto;
}

.sl-wu-feedback {
  margin-top: 12px;
  font-size: 13px;
}

.sl-wu-warn {
  color: var(--sl-danger);
}

dialog.sl-dialog {
  max-width: min(440px, 92vw);
  padding: 24px;
  border: 1px solid var(--sl-border-strong);
  border-radius: 16px;
  background: var(--sl-surface);
  color: var(--sl-text);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.35);
}

dialog.sl-dialog:modal {
  overscroll-behavior: contain;
}

dialog.sl-dialog::backdrop {
  background: rgba(13, 15, 17, 0.56);
  overflow: hidden;
  overscroll-behavior: contain;
}

.sl-dialog-title {
  margin: 0 0 8px;
  font-size: 17px;
  font-weight: 700;
}

.sl-dialog-body {
  margin: 0 0 18px;
}

.sl-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
