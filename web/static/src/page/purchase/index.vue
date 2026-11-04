<template>
  <div class="sl-page">
    <div class="sl-page-head">
      <div>
        <p class="sl-eyebrow">{{ $t("purchase.eyebrow") }}</p>
        <h1 class="sl-h1">{{ $t("purchase.title") }}</h1>
        <p class="sl-desc">{{ $t("purchase.subtitle") }}</p>
      </div>
      <div class="sl-head-acts">
        <a class="sl-btn" :href="links.download" target="_blank" rel="noopener noreferrer">
          {{ $t("purchase.download_tv") }}
        </a>
        <a v-if="links.trial" class="sl-btn" :href="links.trial" target="_blank" rel="noopener noreferrer">
          {{ $t("purchase.trial_link") }}
        </a>
        <router-link class="sl-btn sl-pri" :to="{ name: 'explore' }">
          {{ $t("purchase.explore") }}
        </router-link>
      </div>
    </div>

    <!-- Boundary banner: the installer is free; the license is for the player. -->
    <div class="sl-banner sl-gold">
      <GoldIcon />
      <div class="grow">{{ $t("purchase.free_note") }}</div>
    </div>

    <!-- Plan cards (shared lic-plan language): art left / body right.
         Prices and purchase CTAs only render from verified config; without a
         verified channel the card stays informative, never a dead button. -->
    <div class="sl-plans">
      <article class="sl-plan">
        <img class="sl-plan-art" src="/plan/cover-basic.jpg" :alt="$t('purchase.basic_name') + ' · ' + $t('purchase.cover_alt')" loading="lazy" />
        <div class="sl-plan-body">
          <h2 class="sl-plan-name">
            {{ $t("purchase.basic_name") }}
            <span class="sl-plan-tag">{{ $t("purchase.basic_tag") }}</span>
          </h2>
          <!-- Price mirrors the licensing backend's verified amount (cents);
               it renders only for a positive amount and keeps its 角/分. -->
          <div v-if="basicPrice" class="sl-plan-price">
            <span>¥</span>{{ basicPrice }}
          </div>
          <p class="sl-plan-note">{{ $t("purchase.basic_note") }}</p>
          <ul class="sl-plan-feats">
            <li>{{ $t("purchase.basic_f1") }}</li>
            <li>{{ $t("purchase.basic_f2") }}</li>
            <li>{{ $t("purchase.basic_f3") }}</li>
            <li>{{ $t("purchase.basic_f4") }}</li>
          </ul>
          <a class="sl-btn sl-pri sl-plan-cta" :href="links.basic" target="_blank" rel="noopener noreferrer">
            {{ $t("purchase.buy_basic") }}
          </a>
        </div>
      </article>

      <article class="sl-plan">
        <img class="sl-plan-art" src="/plan/cover-pro.jpg" :alt="$t('purchase.pro_name') + ' · ' + $t('purchase.cover_alt')" loading="lazy" />
        <div class="sl-plan-body">
          <h2 class="sl-plan-name">
            {{ $t("purchase.pro_name") }}
            <span class="sl-plan-tag">Pro</span>
          </h2>
          <div v-if="proPrice" class="sl-plan-price">
            <span>¥</span>{{ proPrice }}
          </div>
          <p class="sl-plan-note">{{ $t("purchase.pro_note") }}</p>
          <ul class="sl-plan-feats">
            <li>{{ $t("purchase.pro_f1") }}</li>
            <li>{{ $t("purchase.pro_f2") }}</li>
            <li>{{ $t("purchase.pro_f3") }}</li>
            <li>{{ $t("purchase.pro_f4") }}</li>
            <li>{{ $t("purchase.pro_f5") }}</li>
          </ul>
          <a class="sl-btn sl-pri sl-plan-cta" :href="links.pro" target="_blank" rel="noopener noreferrer">
            {{ $t("purchase.buy_pro") }}
          </a>
        </div>
      </article>
    </div>
    <p class="sl-hint" style="margin: 10px 0 0">{{ $t("purchase.price_note") }}</p>

    <!-- Experience card and activation guide share the same two-column grid
         as the paid plans. Both stack vertically on smaller screens. -->
    <section class="sl-plans sl-purchase-support">
      <article class="sl-plan sl-trial">
        <img class="sl-plan-art" src="/plan/cover-trial.jpg" :alt="$t('purchase.trial_name') + ' · ' + $t('purchase.cover_alt')" loading="lazy" />
        <div class="sl-plan-body">
          <h2 class="sl-plan-name">
            {{ $t("purchase.trial_name") }}
            <span v-if="trialOpen" class="sl-plan-tag">{{ $t("purchase.trial_days_tag", { days: trialDays }) }}</span>
            <span v-else class="sl-plan-tag sl-plan-tag-idle">{{ $t("purchase.trial_tag_idle") }}</span>
          </h2>
          <p class="sl-plan-note">{{ $t("purchase.trial_note") }}</p>
          <ul class="sl-plan-feats">
            <li>{{ $t("purchase.trial_f1") }}</li>
            <li>{{ $t("purchase.trial_f2") }}</li>
            <li>{{ $t("purchase.trial_f3") }}</li>
          </ul>
          <!-- The claim button exists only while the licensing backend reports
               the campaign open and published an official claim link; a closed
               campaign shows an explanation, never a dead button. -->
          <a v-if="trialOpen" class="sl-btn sl-pri sl-plan-cta" :href="trialUrl" target="_blank" rel="noopener noreferrer">
            {{ $t("purchase.trial_cta") }}
          </a>
          <p v-else class="sl-plan-hint">{{ $t("purchase.trial_closed") }}</p>
        </div>
      </article>

      <!-- The installer itself never handles player activation codes. -->
      <div class="sl-card sl-pad sl-activation-guide">
        <div class="sl-card-head">
          <h2 class="sl-card-title"><span class="sl-ico-t sl-gold"><GoldIcon /></span>{{ $t("purchase.steps_title") }}</h2>
          <span class="sl-card-sub">{{ $t("purchase.steps_sub") }}</span>
        </div>
        <ol class="sl-steps">
          <li>
            <span class="st">1</span>
            <div>
              <b>{{ $t("purchase.step1_t") }}</b>
              <div class="sl-hint">{{ $t("purchase.step1_d") }}</div>
            </div>
          </li>
          <li>
            <span class="st">2</span>
            <div>
              <b>{{ $t("purchase.step2_t") }}</b>
              <div class="sl-hint">{{ $t("purchase.step2_d") }}</div>
            </div>
          </li>
          <li>
            <span class="st">3</span>
            <div>
              <b>{{ $t("purchase.step3_t") }}</b>
              <div class="sl-hint">{{ $t("purchase.step3_d") }}</div>
            </div>
          </li>
        </ol>
      </div>
    </section>

    <!-- Feature comparison, from the official released feature set only. -->
    <section class="sl-section-gap">
      <div class="sl-card sl-pad">
        <div class="sl-card-head">
          <h2 class="sl-card-title"><span class="sl-ico-t"><AppsIcon /></span>{{ $t("purchase.cmp_title") }}</h2>
          <span class="sl-card-sub">{{ $t("purchase.cmp_sub") }}</span>
        </div>
        <div class="sl-tablewrap">
          <table class="sl-cmp">
            <thead>
              <tr>
                <th></th>
                <th>
                  {{ $t("purchase.basic_name") }}
                  <span class="p">{{ $t("purchase.cmp_free") }}</span>
                  <span class="sl-hint">{{ $t("purchase.cmp_free_hint") }}</span>
                </th>
                <th class="sl-gold-col">
                  {{ $t("purchase.pro_name") }}
                  <span class="p">Pro</span>
                  <span class="sl-hint">{{ $t("purchase.cmp_pro_hint") }}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in rows" :key="row.key">
                <td>{{ $t("purchase." + row.key) }}</td>
                <td>
                  <CheckIcon v-if="row.basic" class="y" />
                  <XIcon v-else class="n" />
                </td>
                <td>
                  <CheckIcon v-if="row.pro" class="y" />
                  <XIcon v-else class="n" />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>

    <!-- FAQ: keeps the 7-day signing timeline and the player license apart. -->
    <section class="sl-section-gap">
      <div class="sl-card sl-pad">
        <div class="sl-card-head">
          <h2 class="sl-card-title"><span class="sl-ico-t"><HelpIcon /></span>{{ $t("purchase.faq_title") }}</h2>
        </div>
        <details class="sl-faq">
          <summary>{{ $t("purchase.faq1_q") }}</summary>
          <div><p>{{ $t("purchase.faq1_a") }}</p></div>
        </details>
        <details class="sl-faq">
          <summary>{{ $t("purchase.faq2_q") }}</summary>
          <div><p>{{ $t("purchase.faq2_a") }}</p></div>
        </details>
        <details class="sl-faq">
          <summary>{{ $t("purchase.faq3_q") }}</summary>
          <div><p>{{ $t("purchase.faq3_a") }}</p></div>
        </details>
        <details class="sl-faq">
          <summary>{{ $t("purchase.faq4_q") }}</summary>
          <div><p>{{ $t("purchase.faq4_a") }}</p></div>
        </details>
        <details class="sl-faq">
          <summary>{{ $t("purchase.faq5_q") }}</summary>
          <div><p>{{ $t("purchase.faq5_a") }}</p></div>
        </details>
      </div>
    </section>

    <HomeFooter />
  </div>
</template>

<script>
import HomeFooter from "@/components/HomeFooter.vue";
import {
  songlibConfig,
  SONGLIB_DEFAULT_LINKS,
  formatPlanPrice,
  trialIsClaimable,
  trialClaimURL,
  trialDays as resolveTrialDays,
} from "@/utils/songlib-config.mjs";

export default {
  name: "Purchase",
  components: { HomeFooter },
  data() {
    return {
      links: { ...SONGLIB_DEFAULT_LINKS },
      // Verified, released feature differences (official site copy). Rows must
      // match what the player actually ships; adjust only with the product.
      rows: [
        { key: "cmp_row_libraries", basic: true, pro: true },
        { key: "cmp_row_lyrics", basic: true, pro: true },
        { key: "cmp_row_lossless", basic: true, pro: true },
        { key: "cmp_row_merge", basic: false, pro: true },
        { key: "cmp_row_wave", basic: false, pro: true },
        { key: "cmp_row_transcode", basic: false, pro: true },
        { key: "cmp_row_cover", basic: false, pro: true },
        { key: "cmp_row_discover", basic: false, pro: true },
        { key: "cmp_row_curator", basic: false, pro: true },
      ],
    };
  },
  computed: {
    // Prices keep their 角/分: the previous Math.floor() truncated them and
    // showed 128.50 as 128. A null means "no verified amount", never "free".
    basicPrice() {
      return formatPlanPrice(this.links.basic_price);
    },
    proPrice() {
      return formatPlanPrice(this.links.pro_price);
    },
    // Owner decision 2026-11-04: each plan button goes to its own 16688 goods
    // page (SONGLIB_PLAN_BASIC / SONGLIB_PLAN_PRO are those defaults);
    // /songlib/config can only replace them with another verified goods URL.
    // The trial CTA is gated on the campaign state, not on a fixed anchor.
    trialOpen() {
      return trialIsClaimable(this.links);
    },
    trialUrl() {
      return trialClaimURL(this.links);
    },
    trialDays() {
      return resolveTrialDays(this.links);
    },
  },
  mounted() {
    songlibConfig().then((config) => {
      Object.assign(this.links, config);
    });
  },
};
</script>

<script setup>
import GoldIcon from "@/assets/icons/gold.svg";
import AppsIcon from "@/assets/icons/apps.svg";
import HelpIcon from "@/assets/icons/help.svg";
import CheckIcon from "@/assets/icons/checkmark.svg";
import XIcon from "@/assets/icons/dismiss.svg";
</script>

<style lang="postcss" scoped>
.sl-section-gap {
  margin-top: 18px;
}

.sl-cmp .y,
.sl-cmp .n {
  display: inline-flex;
  width: 16px;
  height: 16px;
}

.sl-cmp thead th .p {
  display: block;
}
</style>
