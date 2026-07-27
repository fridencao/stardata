<script lang="ts">
  import { page } from "$app/stores";
  import {
    canViewBusiness,
    canViewTech,
    defaultHome,
  } from "./user-spaces";
  import { isStudioRoute } from "../../routes/route-constants";
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import StardataUserMenu from "@rilldata/web-common/features/authentication/StardataUserMenu.svelte";
  import { Wrench, Sun, Moon, Home, Languages } from "lucide-svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import { getLocale, setLocale } from "@rilldata/web-common/lib/i18n/gen/runtime";

  $: pathname = $page.url.pathname;
  $: onStudio = isStudioRoute(pathname);
  $: showTechNav = canViewTech();
  $: showStudioLink = showTechNav && !onStudio;
  $: showPortalLink = canViewBusiness() && onStudio;
  $: brandHref = defaultHome();
  $: currentTheme = $themeControl;

  function toggleLocale() {
    // setLocale writes localStorage and reloads the page by default
    setLocale(getLocale() === "zh" ? "en" : "zh");
  }
</script>

<nav class="flex h-[60px] items-center border-b border-gray-200 bg-surface-background/90 px-9 backdrop-blur">
  <a href={brandHref} class="flex items-center gap-2 text-base font-bold text-gray-900">
    <span
      class="grid size-[26px] place-items-center rounded-lg bg-primary-600 text-sm text-white"
    >
      ✦
    </span>
    StarData
  </a>
  <div class="ml-auto flex items-center gap-3">
    {#if showStudioLink}
      <a
        href="/studio"
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline transition-colors hover:bg-primary-100"
      >
        <Wrench class="size-4" /> {m.portal_nav_tech_workbench()}
      </a>
    {/if}
    {#if showPortalLink}
      <a
        href="/"
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline transition-colors hover:bg-primary-100"
      >
        <Home class="size-4" /> {m.portal_nav_business_portal()}
      </a>
    {/if}
    <!-- Theme toggle -->
    <button
      class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100"
      onclick={() => {
        if (currentTheme === "dark") {
          themeControl.set.light();
        } else {
          themeControl.set.dark();
        }
      }}
      title={currentTheme === "dark" ? m.portal_nav_switch_light() : m.portal_nav_switch_dark()}
    >
      {#if currentTheme === "dark"}
        <Sun class="size-4" />
      {:else}
        <Moon class="size-4" />
      {/if}
    </button>
    <!-- Language toggle -->
    <button
      class="flex items-center gap-1 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100"
      onclick={toggleLocale}
      title={m.portal_nav_switch_language()}
    >
      <Languages class="size-4" />
      <span class="text-xs font-semibold">{getLocale() === "zh" ? "EN" : "中"}</span>
    </button>
    <!-- User profile dropdown -->
    <StardataUserMenu />
  </div>
</nav>
