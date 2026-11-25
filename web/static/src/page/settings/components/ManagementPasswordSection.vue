<template>
  <section class="section bg-base-100">
    <h2 id="mg-pw-heading" class="atv-section-heading">{{ $t("settings.management_password.title") }}</h2>
    <p class="sl-caption sl-mp-desc">{{ $t("settings.management_password.desc") }}</p>

    <form class="sl-mp-form" method="post" aria-labelledby="mg-pw-heading" novalidate @submit.prevent="submitPassword">
      <div class="sl-mp-field">
        <label for="mg-current-password" class="sl-mp-label">
          {{ $t("settings.management_password.current_label") }}
          <span aria-hidden="true">*</span>
        </label>
        <input
          id="mg-current-password"
          ref="currentInput"
          name="current-password"
          type="password"
          class="sl-mp-input"
          v-model="currentPassword"
          autocomplete="current-password"
          enterkeyhint="next"
          required
          maxlength="64"
          :aria-invalid="currentError ? 'true' : undefined"
          @input="clearError"
        />
      </div>

      <div class="sl-mp-field">
        <div class="sl-mp-label-row">
          <label for="mg-new-password" class="sl-mp-label">
            {{ $t("settings.management_password.new_label") }}
            <span aria-hidden="true">*</span>
          </label>
          <button
            type="button"
            class="sl-mp-toggle"
            :aria-pressed="showNew ? 'true' : 'false'"
            @click="showNew = !showNew"
          >
            {{ showNew ? $t("settings.management_password.hide") : $t("settings.management_password.show") }}
          </button>
        </div>
        <input
          id="mg-new-password"
          ref="newInput"
          name="new-password"
          :type="showNew ? 'text' : 'password'"
          class="sl-mp-input"
          v-model="newPassword"
          autocomplete="new-password"
          enterkeyhint="done"
          required
          minlength="8"
          maxlength="64"
          aria-describedby="mg-pw-hint"
          :aria-invalid="newError ? 'true' : undefined"
          @input="clearError"
        />
        <span id="mg-pw-hint" class="sl-caption">{{ $t("settings.management_password.hint") }}</span>
      </div>

      <div class="sl-mp-actions">
        <button type="submit" class="sl-btn sl-pri" :disabled="saving">
          <span class="sl-spin" v-if="saving"></span>
          {{ $t("settings.management_password.submit") }}
        </button>
      </div>

      <p v-if="message" class="sl-mp-status" :class="{ 'sl-mp-status--err': isError }" role="status" aria-live="polite">
        {{ message }}
      </p>
    </form>
  </section>
</template>

<script>
import { managementSession, updateManagementCSRF } from "@/utils/management";

export default {
  name: "ManagementPasswordSection",
  data() {
    return {
      currentPassword: "",
      newPassword: "",
      showNew: false,
      saving: false,
      message: "",
      isError: false,
      currentError: false,
      newError: false,
    };
  },
  methods: {
    clearError() {
      this.currentError = false;
      this.newError = false;
      if (this.isError) {
        this.message = "";
        this.isError = false;
      }
    },
    async submitPassword() {
      this.message = "";
      this.isError = false;
      this.currentError = false;
      this.newError = false;

      if (!this.currentPassword) {
        this.currentError = true;
        this.isError = true;
        this.message = this.$t("settings.management_password.err_current_required");
        this.$refs.currentInput?.focus();
        return;
      }
      const len = [...this.newPassword].length;
      if (len < 8 || len > 64 || /[\r\n\0]/.test(this.newPassword)) {
        this.newError = true;
        this.isError = true;
        this.message = this.$t("settings.management_password.err_new_invalid");
        this.$refs.newInput?.focus();
        return;
      }

      this.saving = true;
      try {
        const session = await managementSession();
        const resp = await fetch("/management/password", {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": session.csrf,
          },
          body: JSON.stringify({
            current_password: this.currentPassword,
            new_password: this.newPassword,
          }),
        });
        const data = await resp.json().catch(() => ({}));
        if (!resp.ok) {
          this.isError = true;
          this.message = data.error || this.$t("settings.management_password.err_failed");
          if (resp.status === 401) {
            this.currentError = true;
            this.$refs.currentInput?.focus();
          }
          return;
        }
        if (data.csrf) {
          updateManagementCSRF(data.csrf);
        }
        this.currentPassword = "";
        this.newPassword = "";
        this.showNew = false;
        this.message = this.$t("settings.management_password.saved");
      } catch {
        this.isError = true;
        this.message = this.$t("settings.management_password.err_failed");
      } finally {
        this.saving = false;
      }
    },
  },
};
</script>

<style lang="postcss" scoped>
.sl-mp-desc {
  margin-bottom: 14px;
}

.sl-mp-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-width: 440px;
}

.sl-mp-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.sl-mp-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.sl-mp-label {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--sl-text);
}

.sl-mp-toggle {
  border: 0;
  background: none;
  color: var(--sl-accent);
  font: inherit;
  font-size: 12.5px;
  font-weight: 600;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 6px;
}

.sl-mp-toggle:focus-visible {
  outline: 2px solid var(--sl-accent);
  outline-offset: 2px;
}

.sl-mp-input {
  font: inherit;
  font-size: max(1rem, 1em);
  min-height: 44px;
  padding: 8px 12px;
  border: 1px solid var(--sl-border-strong);
  border-radius: 10px;
  background: var(--sl-bg-soft);
  color: var(--sl-text);
}

.sl-mp-input:focus-visible {
  outline: 2px solid var(--sl-accent);
  outline-offset: 1px;
}

.sl-mp-input[aria-invalid="true"] {
  border-color: var(--sl-danger);
}

.sl-mp-actions {
  display: flex;
  gap: 10px;
}

.sl-mp-status {
  font-size: 13.5px;
  color: var(--sl-success, #2f855a);
}

.sl-mp-status--err {
  color: var(--sl-danger);
}
</style>
