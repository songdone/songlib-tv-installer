<template>
  <div class="sl-shell app-container">
    <header class="sl-topbar">
      <div class="flex items-center gap-x-1 min-w-0">
        <!-- Hamburger only exists on phone widths, where the sidebar is a drawer. -->
        <button
          type="button"
          class="sl-iconbtn md:hidden"
          :aria-label="$t('nav.menu')"
          :aria-expanded="sidebarOpen ? 'true' : 'false'"
          @click.stop="toggleSidebar"
        >
          <span class="atv-theme-glyph" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M4 7h16M4 12h16M4 17h16" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg></span>
        </button>
        <a href="/" class="sl-topbar-brand" title="音屿-Apple TV音乐播放器安装助手">
          <img src="/songlib-mark.png" alt="音屿" />
          <!-- Full official name on desktop; compact name on phones. The
               complete name stays visible via the title and About dialog. -->
          <span class="sl-topbar-brand-title atv-brand-full">音屿-Apple TV音乐播放器安装助手</span>
          <span class="sl-topbar-brand-title atv-brand-short">音屿 TV 安装助手</span>
        </a>
      </div>

      <div class="sl-topbar-actions">
        <!-- Language -->
        <div class="dropdown" :class="{ 'dropdown-hover': !isMobile, 'dropdown-open': isMobile && openMobileMenu === 'language' }" @click="closeMobileMenu">
          <label tabindex="0" class="sl-iconbtn cursor-pointer"
            :aria-label="$t('nav.language')"
            @click.stop="toggleMobileMenu('language')"
            @keydown.enter.prevent="toggleMobileMenu('language')"
            @keydown.space.prevent="toggleMobileMenu('language')"
            :aria-expanded="isMobile ? openMobileMenu === 'language' : undefined">
            <WorldIcon />
          </label>
          <ul tabindex="0" class="dropdown-content dropdown-end z-[1] menu menu-sm p-2 shadow rounded-box w-36 gap-1 sl-menu">
            <li v-for="item in languages" :key="item.key">
              <button :class="[{ active: $i18next.language == item.key }]" @click="changeLanguage(item.key)">
                {{ item.name }}
              </button>
            </li>
          </ul>
        </div>

        <!-- Theme -->
        <div class="dropdown" :class="{ 'dropdown-hover': !isMobile, 'dropdown-open': isMobile && openMobileMenu === 'theme' }" @click="closeMobileMenu">
          <label tabindex="0" class="sl-iconbtn cursor-pointer"
            :aria-label="$t('nav.theme')"
            @click.stop="toggleMobileMenu('theme')"
            @keydown.enter.prevent="toggleMobileMenu('theme')"
            @keydown.space.prevent="toggleMobileMenu('theme')"
            :aria-expanded="isMobile ? openMobileMenu === 'theme' : undefined">
            <span class="atv-theme-glyph" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M12 3 A9 9 0 0 0 12 21 Z" fill="currentColor"/><circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="2.4"/></svg></span>
          </label>
          <ul tabindex="0" class="dropdown-content dropdown-end z-[1] menu menu-sm p-2 shadow rounded-box w-44 gap-1 sl-menu">
            <li v-for="mode in ['system', 'light', 'dark']" :key="mode">
              <button type="button" :class="{ active: themePreference === mode }" :aria-pressed="themePreference === mode" @click="changeTheme(mode)">
                {{ $t('nav.theme_' + mode) }}
              </button>
            </li>
          </ul>
        </div>

        <!-- Help / About: product identity, maintainer and open-source credits.
             Upstream sponsorship lives here, clearly attributed to ATVLoadly,
             never as a main-UI payment entry. -->
        <button type="button" class="sl-iconbtn" :aria-label="$t('nav.about')" @click="showAboutModal = true">
          <HelpIcon />
        </button>

        <a href="https://github.com/songdone/songlib-tv-installer" target="_blank" rel="noopener noreferrer" aria-label="安装助手公开源码 GitHub" title="安装助手公开源码" class="sl-iconbtn">
          <GithubIcon />
        </a>
      </div>
    </header>

    <div class="sl-body">
      <button
        v-if="sidebarOpen"
        type="button"
        class="sl-sidebar-scrim md:hidden"
        :aria-label="$t('nav.menu')"
        @click="sidebarOpen = false"
      ></button>

      <aside class="sl-sidebar" :class="{ 'sl-sidebar--open': sidebarOpen }">
        <nav class="sl-sidebar-nav" aria-label="主导航">
          <router-link class="sl-navitem" :to="{ name: 'home' }" :class="{ 'sl-navitem--active': isActive('home') }" @click="closeSidebar">
            <HomeIcon />
            <span class="sl-navlabel">{{ $t("nav.home") }}</span>
          </router-link>
          <router-link class="sl-navitem" :to="{ name: 'devices' }" :class="{ 'sl-navitem--active': isActive('devices') }" @click="closeSidebar">
            <AppleTVIcon />
            <span class="sl-navlabel">{{ $t("nav.devices") }}</span>
          </router-link>
          <router-link class="sl-navitem" :to="{ name: 'apps' }" :class="{ 'sl-navitem--active': isActive('apps') }" @click="closeSidebar">
            <AppsIcon />
            <span class="sl-navlabel">{{ $t("nav.apps") }}</span>
          </router-link>
          <router-link class="sl-navitem" :to="{ name: 'account' }" :class="{ 'sl-navitem--active': isActive('account') }" @click="closeSidebar">
            <AccountIcon />
            <span class="sl-navlabel">{{ $t("nav.account") }}</span>
          </router-link>
          <router-link class="sl-navitem" :to="{ name: 'explore' }" :class="{ 'sl-navitem--active': isActive('explore') }" @click="closeSidebar">
            <ExploreIcon />
            <span class="sl-navlabel">{{ $t("nav.explore") }}</span>
          </router-link>
          <router-link class="sl-navitem" :to="{ name: 'settings' }" :class="{ 'sl-navitem--active': isActive('settings') }" @click="closeSidebar">
            <SettingsIcon />
            <span class="sl-navlabel">{{ $t("nav.settings") }}</span>
          </router-link>
        </nav>

        <div class="sl-sidebar-foot">
          <button type="button" class="sl-navitem w-full text-left" @click="showAboutModal = true">
            <HelpIcon />
            <span class="sl-navlabel">{{ $t("nav.about_license") }}</span>
          </button>
        </div>
      </aside>

      <main class="sl-main">
        <router-view />
      </main>
    </div>

    <!-- About modal: full product name, maintainer, version and open-source
         credits (docs/VISUAL_SYSTEM_AND_CODEX_HANDOFF_20261104.md §0.7). -->
    <div v-if="showAboutModal" class="modal modal-open modal-middle" @click.self="showAboutModal = false">
      <div class="modal-box relative overflow-hidden sl-about-box">
        <label @click="showAboutModal = false" class="btn btn-sm btn-circle absolute right-2 top-2">✕</label>
        <div class="relative flex flex-col items-center text-center gap-4 py-4">
          <img src="/songlib-mark.png" class="w-20 h-20" alt="音屿" />
          <div class="space-y-1">
            <h3 class="font-bold text-2xl tracking-tight sl-about-title">{{ $t("about.product") }}</h3>
            <p class="text-sm sl-about-muted">{{ $t("about.maintainer") }}: {{ $t("about.maintainer_name") }}</p>
          </div>
          <div class="grid w-full max-w-md grid-cols-2 gap-3">
            <div class="sl-about-cell">
              <div class="sl-caption uppercase tracking-wide">{{ $t("about.version") }}</div>
              <div class="mt-1 text-lg font-semibold">{{ appVersion }}</div>
            </div>
            <div class="sl-about-cell">
              <div class="sl-caption uppercase tracking-wide">{{ $t("about.build_date") }}</div>
              <div class="mt-1 text-lg font-semibold">{{ buildDate }}</div>
            </div>
          </div>
          <div class="w-full max-w-md text-left sl-about-credits">
            <div class="sl-h3">{{ $t("about.credits_title") }}</div>
            <p class="mt-1.5 text-sm leading-relaxed sl-about-muted">
              {{ $t("about.credits_body") }}
            </p>
            <div class="mt-3 flex flex-wrap gap-x-4 gap-y-1.5 text-sm">
              <a class="sl-link" href="/source">{{ $t("about.source_code") }}</a>
              <a class="sl-link" href="/license">AGPL-3.0</a>
              <a class="sl-link" href="https://github.com/songdone/songlib-tv-installer" target="_blank" rel="noopener noreferrer">songdone/songlib-tv-installer</a>
              <a class="sl-link" href="https://github.com/bitxeno/atvloadly" target="_blank" rel="noopener noreferrer">{{ $t("about.upstream_repo") }}</a>
              <a class="sl-link" href="https://songlib.playsong.cn/" target="_blank" rel="noopener noreferrer">{{ $t("about.website") }}</a>
            </div>
            <div class="mt-3 text-xs leading-relaxed sl-about-muted">
              {{ $t("about.upstream_support") }}:
              <a class="sl-link" href="https://github.com/sponsors/bitxeno" target="_blank" rel="noopener noreferrer">GitHub Sponsors</a> ·
              <a class="sl-link" href="https://ko-fi.com/bitxeno" target="_blank" rel="noopener noreferrer">Ko-fi</a> ·
              <a class="sl-link" href="https://ifdian.net/a/bitxeno" target="_blank" rel="noopener noreferrer">爱发电</a>
            </div>
          </div>
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
      languages: [],
      isMobile: window.matchMedia("(max-width: 767px)").matches,
      mobileMediaQuery: null,
      openMobileMenu: null,
      sidebarOpen: false,
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
    document.addEventListener("click", this.onDocumentClick);
    document.addEventListener("keydown", this.onDocumentKeydown);
    this.colorSchemeQuery = window.matchMedia("(prefers-color-scheme: dark)");
    this.colorSchemeQuery.addEventListener("change", this.applyTheme);
    this.applyTheme();
  },
  beforeUnmount() {
    this.colorSchemeQuery?.removeEventListener("change", this.applyTheme);
    this.mobileMediaQuery?.removeEventListener("change", this.onMobileBreakpointChange);
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
    onMobileBreakpointChange(event) {
      this.isMobile = event.matches;
      this.openMobileMenu = null;
      this.sidebarOpen = false;
    },
    toggleMobileMenu(menu) {
      if (!this.isMobile) return;
      if (this.openMobileMenu === menu) {
        this.closeMobileMenu();
      } else {
        this.openMobileMenu = menu;
      }
    },
    closeMobileMenu() {
      if (!this.isMobile) return;
      this.openMobileMenu = null;
      const focused = document.activeElement;
      if (focused?.closest?.(".sl-topbar .dropdown")) {
        focused.blur();
      }
    },
    onDocumentClick(event) {
      if (this.isMobile &&
          !event.target?.closest?.(".sl-topbar .dropdown")) {
        this.closeMobileMenu();
      }
      // Any click outside the drawer closes it.
      if (this.sidebarOpen &&
          !event.target?.closest?.(".sl-sidebar") &&
          !event.target?.closest?.(".sl-topbar")) {
        this.sidebarOpen = false;
      }
    },
    onDocumentKeydown(event) {
      if (this.isMobile && event.key === "Escape") {
        this.closeMobileMenu();
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
      api.syncLang({lang: lang})
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
</script>

<style lang="postcss" scoped>
/* AppShell visuals that stay component-specific; generic shell styles live
   in styles/layout.css. */

.sl-menu {
  background: var(--sl-panel);
  border: 1px solid var(--sl-border);
  border-radius: var(--sl-radius-md);
  box-shadow: var(--sl-shadow-popover);
  color: var(--sl-text);
}

.sl-menu .active {
  color: var(--sl-text);
  background: var(--sl-panel-hover);
}

/* Brand name swap: full name from 768px up (sidebar carries navigation),
   compact name on phones where horizontal space is tight. */
.atv-brand-full {
  display: none;
}

.atv-brand-short {
  display: inline;
}

@media (min-width: 768px) {
  .atv-brand-full {
    display: inline;
  }

  .atv-brand-short {
    display: none;
  }
}

.sl-about-box {
  background: var(--sl-panel);
  color: var(--sl-text);
  border: 1px solid var(--sl-border);
  box-shadow: var(--sl-shadow-popover);
}

.sl-about-title {
  color: var(--sl-text);
}

.sl-about-muted {
  color: var(--sl-text-muted);
}

.sl-about-cell {
  border: 1px solid var(--sl-border);
  background: var(--sl-panel-soft);
  border-radius: var(--sl-radius-md);
  padding: 12px 16px;
}

.sl-about-credits {
  border: 1px solid var(--sl-border);
  background: var(--sl-panel-soft);
  border-radius: var(--sl-radius-md);
  padding: 16px 20px;
}
</style>
