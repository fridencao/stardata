import { paraglideVitePlugin } from "@inlang/paraglide-js";
import { sveltekit } from "@sveltejs/kit/vite";
import dns from "dns";
import { defineConfig } from "vitest/config";
import { readPublicEmailDomains } from "./src/features/projects/user-management/readPublicEmailDomains";

// print dev server as `localhost` not `127.0.0.1`
dns.setDefaultResultOrder("verbatim");

export default defineConfig({
  resolve: {
    alias: {
      "@rilldata/web-admin": "/src",
      "@rilldata/web-common": "/../web-common/src",
    },
  },
  server: {
    port: 3000,
    strictPort: true,
  },
  preview: {
    port: 3000,
    strictPort: true,
  },
  define: {
    StarDataPublicEmailDomains: readPublicEmailDomains(),
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
    exclude: ["sveltekit-superforms"],
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
      name: "stardata-esbuild-css-minify",
      configResolved(config) {
        config.build.cssMinify = "esbuild";
      },
    },
  ],
  envDir: "../",
  envPrefix: "RILL_UI_PUBLIC_",
  css: {
    // lightningcss (Vite 8 default CSS transformer) mis-parses Tailwind
    // arbitrary-value utilities such as `.text-[13px]` during transform. Use
    // the postcss pipeline to transform CSS so the build does not choke.
    transformer: "postcss",
  },
});
