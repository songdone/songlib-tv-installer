import { defineConfig } from "vite";
import { resolve } from "path";
import vue from "@vitejs/plugin-vue";
import svgLoader from "vite-svg-loader";

export default defineConfig({
  // Relative asset URLs resolve through the backend-injected <base href="/_web/<revision>/">
  // for both packaged go:embed builds and signed frontend hot-update snapshots.
  base: "./",
  plugins: [vue(), svgLoader()],
  resolve: {
    alias: [
      {
        find: "@",
        replacement: resolve(__dirname, "src"),
      },
      {
        find: "@locales",
        replacement: resolve(__dirname, "../../locales"),
      },
    ],
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          const localeMatch = id.match(/\/locales\/([A-Za-z0-9_-]+)\.json$/);
          if (localeMatch) {
            return `locale-${localeMatch[1]}`;
          }
          if (!id.includes("node_modules")) {
            return undefined;
          }
          if (/node_modules\/(vue|@vue|vue-router|vue3-popper|vue3-toastify)\//.test(id)) {
            return "vendor-vue";
          }
          if (/node_modules\/i18next/.test(id)) {
            return "vendor-i18n";
          }
          if (/node_modules\/(jszip|plist|crypto-js|pako|sax|xmlbuilder|@xmldom|readable-stream|lie|setimmediate|immediate|buffer)\//.test(id)) {
            return "vendor-ipa";
          }
          return "vendor-misc";
        },
      },
    },
  },
});
