import { sveltekit } from "@sveltejs/kit/vite";
import dns from "dns";
import { defineConfig } from "vitest/config";
import { paraglideVitePlugin } from "@inlang/paraglide-js";

// print dev server as `localhost` not `127.0.0.1`
dns.setDefaultResultOrder("verbatim");

const config = defineConfig({
  build: {
    rolldownOptions: {
      // This ensures that the web-admin package is not bundled into the web-local package.
      // This is necessary because the Scheduled Reports dialog lives in `web-common` and imports the admin-client.
      external: (id) => id.startsWith("@rilldata/web-admin/"),
    },
  },
  resolve: {
    alias: {
      src: "/src", // trick to get absolute imports to work
      "@rilldata/web-local": "/src",
      "@rilldata/web-common": "/../web-common/src",
      "@rilldata/web-admin": "/../web-admin/src",
    },
  },
  server: {
    strictPort: true,
    fs: {
      allow: ["."],
    },
  },
  define: {
    "import.meta.env.VITE_PLAYWRIGHT_TEST": process.env.PLAYWRIGHT_TEST,
    "import.meta.env.VITE_PLAYWRIGHT_CLOUD_TEST":
      process.env.PLAYWRIGHT_CLOUD_TEST,
  },
  optimizeDeps: {
    include: [
      "@tanstack/svelte-query",
      "@codemirror/view",
      "@codemirror/state",
      "@codemirror/language",
      "d3-scale",
      "d3-format",
      "d3-array",
      "luxon",
      "vega-lite",
      "memoize-weak",
    ],
  },
  css: {
    // lightningcss (Vite 8 default CSS transformer) mis-parses Tailwind
    // arbitrary-value utilities such as `.text-[13px]` during transform. Use
    // the postcss pipeline to transform CSS so the build does not choke.
    transformer: "postcss",
  },
  plugins: [
    sveltekit(),
    paraglideVitePlugin({
      project: "../web-common/src/lib/i18n/project.inlang",
      outdir: "../web-common/src/lib/i18n/gen",
      strategy: ["cookie", "preferredLanguage", "baseLocale"],
    }),
    {
      // SvelteKit's vite plugin forces `build.cssMinify` to a boolean derived
      // from `build.minify` (see @sveltejs/kit src/exports/vite/index.js:923),
      // ignoring any explicit cssMinify string. lightningcss's CSS *minifier*
      // mis-parses Tailwind arbitrary-value selectors such as `.text-[13px]`
      // ("No qualified name in attribute selector"), so CSS minification must
      // be routed through esbuild instead. This hook runs after SvelteKit's
      // config() hook, so it wins and forces esbuild for CSS minification
      // (esbuild does not parse selectors semantically and handles these fine)
      // while leaving JS minification untouched.
      name: "force-esbuild-css-minify",
      configResolved(config) {
        config.build.cssMinify = "esbuild";
      },
    },
  ],
  envDir: "../",
  envPrefix: "RILL_UI_PUBLIC_",
});

export default config;
