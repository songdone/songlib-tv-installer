<template>
  <div class="sl-shell app-container">
    <div class="sl-bgfx sl-work" aria-hidden="true">
      <div class="sl-g sl-g1"></div>
      <div class="sl-g sl-g2"></div>
      <div class="sl-g sl-g3"></div>
    </div>

    <header class="sl-topbar">
      <div class="sl-topbar-in">
        <div class="sl-topbar-left">
          <!-- Hamburger only exists on phone widths, where the sidebar is a drawer. -->
          <button
            type="button"
            class="sl-iconbtn sl-menu-btn"
            :aria-label="$t('nav.menu')"
            :aria-expanded="sidebarOpen ? 'true' : 'false'"
            @click.stop="toggleSidebar"
          >
            <span aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M4 7h16M4 12h16M4 17h16" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg></span>
          </button>
          <a href="/" class="sl-brand" title="音屿-Apple TV音乐播放器安装助手">
            <img src="/songlib-mark.png" alt="音屿" />
            <span class="sl-brand-name">音屿-Apple TV音乐播放器安装助手</span>
            <span class="sl-brand-name sl-brand-name--short">音屿 TV 安装助手</span>
          </a>
        </div>

        <div class="sl-top-actions">
          <!-- Language -->
          <div class="sl-pop-anchor">
            <button type="button" class="sl-iconbtn" :aria-label="$t('nav.language')" aria-haspopup="true" :aria-expanded="openMenu === 'language' ? 'true' : 'false'" @click.stop="toggleMenu('language')">
              <WorldIcon />
            </button>
            <div v-if="openMenu === 'language'" class="sl-pop" @click.stop>
              <button v-for="item in languages" :key="item.key" class="sl-pop-i" :class="{ 'sl-on': $i18next.language === item.key }" @click="changeLanguage(item.key)">
                {{ item.name }}
                <CheckIcon v-if="$i18next.language === item.key" class="ck" />
              </button>
            </div>
          </div>

          <!-- Theme -->
          <div class="sl-pop-anchor">
            <button type="button" class="sl-iconbtn" :aria-label="$t('nav.theme')" aria-haspopup="true" :aria-expanded="openMenu === 'theme' ? 'true' : 'false'" @click.stop="toggleMenu('theme')">
              <span class="sl-theme-glyph" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M12 3 A9 9 0 0 0 12 21 Z" fill="currentColor"/><circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="2.4"/></svg></span>
            </button>
            <div v-if="openMenu === 'theme'" class="sl-pop" style="width: 236px" @click.stop>
              <div class="sl-pop-t">{{ $t("nav.theme") }}</div>
              <div class="row">
                <button :class="{ 'sl-on': themePreference === 'dark' }" @click="changeTheme('dark')">深色</button>
                <button :class="{ 'sl-on': themePreference === 'light' }" @click="changeTheme('light')">浅色</button>
                <button :class="{ 'sl-on': themePreference === 'system' }" @click="changeTheme('system')">跟随系统</button>
              </div>
            </div>
          </div>

          <!-- About -->
          <button type="button" class="sl-iconbtn" :aria-label="$t('nav.about')" @click="showAboutModal = true">
            <HelpIcon />
          </button>

          <a href="https://github.com/songdone/songlib-tv-installer" target="_blank" rel="noopener noreferrer" aria-label="安装助手公开源码 GitHub" title="安装助手公开源码" class="sl-iconbtn">
            <GithubIcon />
          </a>
        </div>
      </div>
    </header>

    <div class="sl-body">
      <button
        v-if="sidebarOpen"
        type="button"
        class="sl-sidebar-scrim"
        :aria-label="$t('nav.menu')"
        @click="sidebarOpen = false"
      ></button>

      <aside class="sl-sidebar" :class="{ 'sl-sidebar--open': sidebarOpen }">
        <nav class="sl-sidebar-nav" aria-label="主导航">
          <router-link class="sl-navitem" :class="{ 'sl-navitem--active': isActive(item.name) }" v-for="item in NAV" :key="item.name" :to="{ name: item.name }" @click="closeSidebar">
            <component :is="icons[item.icon]" />
            <span class="sl-navlabel">{{ $t(item.label) }}</span>
          </router-link>
        </nav>

        <div class="sl-sidebar-foot">
          <button type="button" class="sl-navitem sl-w-full" @click="showAboutModal = true">
            <HelpIcon />
            <span class="sl-navlabel">{{ $t("nav.about_license") }}</span>
          </button>
        </div>
      </aside>

      <main class="sl-main">
        <div class="sl-wrap">
          <router-view />
        </div>
      </main>
    </div>

    <!-- About modal: product identity, maintainer, open-source credits.
         Upstream sponsorship lives here, clearly attributed to ATVLoadly. -->
    <div v-if="showAboutModal" class="sl-overlay" @click.self="showAboutModal = false">
      <div class="sl-modal" role="dialog" :aria-label="$t('about.product')">
        <header>
          {{ $t("about.credits_title") }}
          <button type="button" class="sl-iconbtn x" :aria-label="$t('common.close')" @click="showAboutModal = false"><XIcon /></button>
        </header>
        <div class="content">
          <div class="sl-about-head">
            <img src="/songlib-mark.png" alt="音屿" />
            <div>
              <h3 class="sl-plan-name">{{ $t("about.product") }}</h3>
              <p class="sl-hint">{{ $t("about.maintainer") }}: {{ $t("about.maintainer_name") }} · v{{ appVersion }} ({{ buildDate }})</p>
            </div>
          </div>
          <p class="sl-hint" style="margin: 12px 0 0">{{ $t("about.credits_body") }}</p>
          <div class="sl-about-links">
            <a class="sl-link" href="/source">{{ $t("about.source_code") }}</a>
            <a class="sl-link" href="/license">AGPL-3.0</a>
            <a class="sl-link" href="https://github.com/songdone/songlib-tv-installer" target="_blank" rel="noopener noreferrer">songdone/songlib-tv-installer</a>
            <a class="sl-link" href="https://github.com/bitxeno/atvloadly" target="_blank" rel="noopener noreferrer">{{ $t("about.upstream_repo") }}</a>
            <a class="sl-link" href="https://songlib.playsong.cn/" target="_blank" rel="noopener noreferrer">{{ $t("about.website") }}</a>
          </div>
          <p class="sl-hint" style="margin-top: 12px">{{ $t("about.credits_footer") }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import api from "@/api/api";

export default {
  name: "App",
  data() {
    return {
      NAV: [
        { name: "home", label: "nav.home", icon: "home" },
        { name: "devices", label: "nav.devices", icon: "appletv" },
        { name: "apps", label: "nav.apps", icon: "apps" },
        { name: "account", label: "nav.account", icon: "account" },
        { name: "explore", label: "nav.explore", icon: "explore" },
        { name: "purchase", label: "nav.purchase", icon: "gold" },
        { name: "settings", label: "nav.settings", icon: "settings" },
      ],
      languages: [],
      sidebarOpen: false,
      openMenu: null,
      themePreference: "light",
      colorSchemeQuery: null,
      showAboutModal: false,
      appVersion: "unknown",
      buildDate: "unknown",
    };
  },
  created() {
    const saved = window.localStorage.getItem("atvloadly-theme");
    this.themePreference = ["system", "light", "dark"].includes(saved) ? saved : "light";
    api.syncLang({lang: this.$i18next.language})
    api.getVersion().then((res) => {
      this.appVersion = res.data?.version || "unknown";
      this.buildDate = this.formatBuildDate(res.data?.build_date);
    });
    let keys = Object.keys(this.$i18next.options.resources);
    for (const key of keys) {
      this.languages.push({
        key: key,
        name: this.$i18next.options.resources[key].name,
      });
    }
  },
  mounted() {
    this.mobileMediaQuery = window.matchMedia("(max-width: 767px)");
    this.mobileMediaQuery.addEventListener("change", this.onMobileBreakpointChange);
    this.colorSchemeQuery = window.matchMedia("(prefers-color-scheme: dark)");
    this.colorSchemeQuery.addEventListener("change", this.applyTheme);
    this.applyTheme();
    document.addEventListener("click", this.onDocumentClick);
    document.addEventListener("keydown", this.onDocumentKeydown);
  },
  beforeUnmount() {
    this.mobileMediaQuery?.removeEventListener("change", this.onMobileBreakpointChange);
    this.colorSchemeQuery?.removeEventListener("change", this.applyTheme);
    document.removeEventListener("click", this.onDocumentClick);
    document.removeEventListener("keydown", this.onDocumentKeydown);
  },
  methods: {
    isActive(name) {
      // Keep install/pair flows visually attached to their entry sections.
      if (name === "devices") {
        return ["devices", "add-device", "pair", "pair-manual"].includes(this.$route.name);
      }
      if (name === "apps") {
        return ["apps", "install"].includes(this.$route.name);
      }
      return this.$route.name === name;
    },
    toggleSidebar() {
      this.sidebarOpen = !this.sidebarOpen;
    },
    closeSidebar() {
      this.sidebarOpen = false;
    },
    onMobileBreakpointChange() {
      this.sidebarOpen = false;
      this.openMenu = null;
    },
    toggleMenu(menu) {
      this.openMenu = this.openMenu === menu ? null : menu;
    },
    onDocumentClick(event) {
      if (!event.target?.closest?.(".sl-pop-anchor")) {
        this.openMenu = null;
      }
      // Any click outside the drawer closes it.
      if (this.sidebarOpen &&
          !event.target?.closest?.(".sl-sidebar") &&
          !event.target?.closest?.(".sl-topbar")) {
        this.sidebarOpen = false;
      }
    },
    onDocumentKeydown(event) {
      if (event.key === "Escape") {
        this.openMenu = null;
        this.sidebarOpen = false;
      }
    },
    applyTheme() {
      const dark = this.themePreference === "dark" ||
        (this.themePreference === "system" && this.colorSchemeQuery?.matches);
      document.documentElement.setAttribute("data-theme", dark ? "dark" : "winter");
    },
    changeTheme(mode) {
      if (!["system", "light", "dark"].includes(mode)) return;
      this.themePreference = mode;
      window.localStorage.setItem("atvloadly-theme", mode);
      this.applyTheme();
      this.openMenu = null;
    },
    formatBuildDate(buildDate) {
      if (!buildDate) {
        return "unknown";
      }

      const text = String(buildDate);
      const matched = text.match(/^(\d{4})[-/](\d{1,2})[-/](\d{1,2})/);
      if (matched) {
        return `${matched[1]}-${matched[2].padStart(2, "0")}-${matched[3].padStart(2, "0")}`;
      }

      const monthOnly = text.match(/^(\d{4})[-/](\d{2})/);
      if (monthOnly) {
        return `${monthOnly[1]}-${monthOnly[2]}`;
      }

      const parsed = new Date(buildDate);
      if (Number.isNaN(parsed.getTime())) {
        return buildDate;
      }

      const year = parsed.getFullYear();
      const month = String(parsed.getMonth() + 1).padStart(2, "0");
      const day = String(parsed.getDate()).padStart(2, "0");
      return `${year}-${month}-${day}`;
    },
    changeLanguage(lang) {
      this.$i18next.changeLanguage(lang);
      api.syncLang({lang: lang});
      this.openMenu = null;
    },
  },
};
</script>

<script setup>
import SettingsIcon from "@/assets/icons/settings.svg";
import WorldIcon from "@/assets/icons/world.svg";
import GithubIcon from "@/assets/icons/github.svg";
import AccountIcon from "@/assets/icons/person.svg";
import HomeIcon from "@/assets/icons/home.svg";
import AppsIcon from "@/assets/icons/apps.svg";
import ExploreIcon from "@/assets/icons/explore.svg";
import HelpIcon from "@/assets/icons/help.svg";
import CheckIcon from "@/assets/icons/checkmark.svg";
import XIcon from "@/assets/icons/dismiss.svg";
import GoldIcon from "@/assets/icons/gold.svg";
import AppleTVIcon from "@/assets/icons/appletv.svg";

const icons = {
  home: HomeIcon,
  appletv: AppleTVIcon,
  apps: AppsIcon,
  account: AccountIcon,
  explore: ExploreIcon,
  gold: GoldIcon,
  settings: SettingsIcon,
};
</script>

<style lang="postcss" scoped>
.sl-topbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.sl-theme-glyph {
  display: inline-flex;
  width: 17px;
  height: 17px;
}

.sl-theme-glyph svg {
  display: block;
  width: 100%;
  height: 100%;
}

.sl-w-full {
  width: 100%;
  text-align: left;
}

.sl-about-head {
  display: flex;
  align-items: center;
  gap: 14px;
}

.sl-about-head img {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  flex: none;
}

.sl-about-links {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 16px;
  margin-top: 12px;
  font-size: 13.5px;
}

.sl-modal .sl-pop-i {
  text-decoration: none;
  color: var(--sl-text);
}
</style>
