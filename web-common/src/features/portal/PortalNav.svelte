<script lang="ts">
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import StardataUserMenu from "@rilldata/web-common/features/authentication/StardataUserMenu.svelte";
  import {
    Wrench,
    Sun,
    Moon,
    Home,
    Languages,
    ShieldCheck,
    Bell,
    FileText,
  } from "lucide-svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import { getLocale, setLocale } from "@rilldata/web-common/lib/i18n/gen/runtime";

  /** 品牌区点击后的落地页(按角色由调用方决定) */
  export let brandHref = "/";
  /** 「技术工作台」入口;null = 不显示(无治理权限或已在工作台内) */
  export let studioHref: string | null = null;
  /** 「业务门户」返回入口;null = 不显示(未处于工作台内) */
  export let portalHref: string | null = null;
  /** 「平台管理」入口;null = 不显示(非 org admin) */
  export let adminHref: string | null = null;
  /** 「我的报告」入口;null = 不显示(功能未启用或无权限) */
  export let reportsHref: string | null = null;
  /** 「我的订阅」入口;null = 不显示(功能未启用或无权限) */
  export let alertsHref: string | null = null;

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
    <!-- i18n-ignore: brand name -->
    StarData
  </a>
  <div class="ml-auto flex items-center gap-3">
    <slot name="center" />
    {#if studioHref}
      <a
        href={studioHref}
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline transition-colors hover:bg-primary-100"
      >
        <Wrench class="size-4" /> {m.portal_nav_tech_workbench()}
      </a>
    {/if}
    {#if portalHref}
      <a
        href={portalHref}
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline transition-colors hover:bg-primary-100"
      >
        <Home class="size-4" /> {m.portal_nav_business_portal()}
      </a>
    {/if}
    {#if reportsHref}
      <a
        href={reportsHref}
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline transition-colors hover:bg-primary-100"
      >
        <FileText class="size-4" /> {m.nav_tab_reports()}
      </a>
    {/if}
    {#if alertsHref}
      <a
        href={alertsHref}
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline transition-colors hover:bg-primary-100"
      >
        <Bell class="size-4" /> {m.nav_tab_alerts()}
      </a>
    {/if}
    {#if adminHref}
      <a
        href={adminHref}
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-gray-600 no-underline transition-colors hover:bg-gray-100 hover:text-gray-900"
      >
        <ShieldCheck class="size-4" /> {m.portal_nav_platform_admin()}
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
    <!-- User profile dropdown (host app may inject its own, e.g. web-admin's AvatarButton) -->
    <slot name="user">
      <StardataUserMenu />
    </slot>
  </div>
</nav>
