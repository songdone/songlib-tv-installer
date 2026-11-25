import i18next from "i18next";
import LanguageDetector from "i18next-browser-languagedetector";

import en from "@locales/en.json";
import zh_cn from "@locales/zh_cn.json";
import sv from "@locales/sv.json";
import sr from "@locales/sr.json";
import srLatn from "@locales/sr-Latn.json";
import ru from "@locales/ru.json";

// Start this Chinese product in Chinese on first load (no navigator override),
// while preserving deliberate language selections in localStorage across reloads.
i18next.use(LanguageDetector).init({
  detection: {
    lookupLocalStorage: "atvloadly-language",
    order: ["querystring", "localStorage"],
    caches: ["localStorage"],
    convertDetectedLanguage: (lang) => (lang === "zh" ? "zh-CN" : lang),
  },
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
    ru: {
      name: "Русский",
      translation: ru,
    },
  },
});

export default i18next;
