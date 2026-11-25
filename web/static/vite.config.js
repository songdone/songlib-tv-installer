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
});
