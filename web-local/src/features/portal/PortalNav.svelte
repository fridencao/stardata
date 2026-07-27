<script lang="ts">
  import { page } from "$app/stores";
  import { portalRole } from "./portal-role-store";
  import { isStudioRoute } from "../../routes/route-constants";
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import { getStardataToken, clearStardataToken } from "@rilldata/web-common/runtime-client/auth-token";
  import { Wrench, Sun, Moon, User, LogOut } from "lucide-svelte";

  const links = [
    { label: "首页", href: "/" },
    { label: "对话", href: "/chat" },
    { label: "看板", href: "/boards" },
  ];

  $: pathname = $page.url.pathname;
  $: showStudioLink = isStudioRoute(pathname);
  $: currentTheme = $themeControl;
  $: hasToken = !!getStardataToken();

  function isActive(href: string, path: string) {
    return href === "/" ? path === "/" : path.startsWith(href);
  }

  function handleLogout() {
    clearStardataToken();
    window.location.href = "/login";
  }
</script>

<nav class="flex h-[60px] items-center border-b bg-white/90 px-9 backdrop-blur border-gray-200 dark:border-gray-700 dark:bg-gray-950/90">
  <a href="/" class="flex items-center gap-2 text-base font-bold text-gray-900 dark:text-gray-100">
    <span
      class="grid size-[26px] place-items-center rounded-lg bg-primary-600 text-sm text-white"
    >
      ✦
    </span>
    StarData
  </a>
  <div class="flex gap-1">
    {#each links as link (link.href)}
      <a
        href={link.href}
        class="rounded-lg px-3.5 py-1.5 text-sm no-underline transition-colors {isActive(
          link.href,
          pathname,
        )
          ? 'bg-primary-50 font-semibold text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
          : 'text-gray-600 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800'}"
      >
        {link.label}
      </a>
    {/each}
  </div>
  <div class="ml-auto flex items-center gap-3">
    {#if showStudioLink}
      <a
        href="/studio"
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline transition-colors hover:bg-primary-100 dark:text-primary-400 dark:bg-primary-900/20 dark:hover:bg-primary-900/30"
      >
        <Wrench class="size-4" /> 技术工作台
      </a>
    {/if}
    <!-- Role switcher -->
    <div class="flex rounded-lg bg-gray-100 p-0.5 text-xs dark:bg-gray-800">
      <button
        class="rounded-md px-2.5 py-1 transition-colors {$portalRole === 'business'
          ? 'bg-white font-semibold shadow-sm dark:bg-gray-700 dark:shadow-none'
          : 'text-gray-500 dark:text-gray-400'}"
        onclick={() => portalRole.set("business")}
      >
        业务视角
      </button>
      <button
        class="rounded-md px-2.5 py-1 transition-colors {$portalRole === 'tech'
          ? 'bg-white font-semibold shadow-sm dark:bg-gray-700 dark:shadow-none'
          : 'text-gray-500 dark:text-gray-400'}"
        onclick={() => portalRole.set("tech")}
      >
        技术视角
      </button>
    </div>
    <!-- Theme toggle -->
    <button
      class="rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
      onclick={() => {
        if (currentTheme === "dark") {
          themeControl.set.light();
        } else {
          themeControl.set.dark();
        }
      }}
      title={currentTheme === "dark" ? "切换浅色模式" : "切换深色模式"}
    >
      {#if currentTheme === "dark"}
        <Sun class="size-4" />
      {:else}
        <Moon class="size-4" />
      {/if}
    </button>
    <!-- User profile dropdown -->
    {#if hasToken}
      <button
        class="flex items-center gap-2 rounded-full border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-700 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-900 dark:text-gray-300 dark:hover:bg-gray-800"
        onclick={handleLogout}
        title="退出登录"
      >
        <LogOut class="size-4" />
        <span class="hidden sm:inline">退出</span>
      </button>
    {/if}
  </div>
</nav>
