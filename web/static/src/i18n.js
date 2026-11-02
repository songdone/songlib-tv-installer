import i18next from "i18next";

import en from "@locales/en.json";
import zh_cn from "@locales/zh_cn.json";
import sv from "@locales/sv.json";
import sr from "@locales/sr.json";
import srLatn from "@locales/sr-Latn.json";

// Start this Chinese product in Chinese even on English browsers and older cached builds.
// Deliberate language changes remain available during the session.
i18next.init({
  lng: "zh-CN",
  //   debug: true,
  fallbackLng: "zh-CN",
  // Translations are rendered as text (Vue interpolation, toasts), which
  // escapes them already: escaping here would show entities like &#39;.
  interpolation: {
    escapeValue: false,
  },
  resources: {
    en: {
      name: "English",
      translation: en,
    },
    "zh-CN": {
      name: "中文",
      translation: zh_cn,
    },
    sv: {
      name: "Svenska",
      translation: sv,
    },
    sr: {
      name: "Српски (ћирилица)",
      translation: sr,
    },
    "sr-Latn": {
      name: "Srpski (latinica)",
      translation: srLatn,
    },
  },
});

export default i18next;
