<script lang="ts">
  import { page } from "$app/stores";
  import { portalRole } from "./portal-role-store";
  import { isStudioRoute } from "../../routes/route-constants";
  import { themeControl } from "@rilldata/web-common/features/themes/theme-control";
  import { Wrench, Sun, Moon, User } from "lucide-svelte";

  const links = [
    { label: "首页", href: "/" },
    { label: "对话", href: "/chat" },
    { label: "看板", href: "/boards" },
  ];

  $: pathname = $page.url.pathname;
  $: showStudioLink = isStudioRoute(pathname);
  $: currentTheme = $themeControl;

  function isActive(href: string, path: string) {
    return href === "/" ? path === "/" : path.startsWith(href);
  }
</script>

<nav
  class="flex h-[60px] items-center gap-7 border-b border-gray-200 bg-white/90 px-9 backdrop-blur"
>
  <a href="/" class="flex items-center gap-2 text-base font-bold text-gray-900">
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
        class="rounded-lg px-3.5 py-1.5 text-sm no-underline {isActive(
          link.href,
          pathname,
        )
          ? 'bg-primary-50 font-semibold text-primary-700'
          : 'text-gray-600 hover:bg-gray-100'}"
      >
        {link.label}
      </a>
    {/each}
  </div>
  <div class="ml-auto flex items-center gap-3">
    {#if showStudioLink}
      <a
        href="/studio"
        class="flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-semibold text-primary-700 bg-primary-50 no-underline"
      >
        <Wrench class="size-4" /> 技术工作台
      </a>
    {/if}
    <!-- Role switcher -->
    <div class="flex rounded-lg bg-gray-100 p-0.5 text-xs">
      <button
        class="rounded-md px-2.5 py-1 {$portalRole === 'business'
          ? 'bg-white font-semibold shadow-sm'
          : 'text-gray-500'}"
        on:click={() => portalRole.set("business")}
      >
        业务视角
      </button>
      <button
        class="rounded-md px-2.5 py-1 {$portalRole === 'tech'
          ? 'bg-white font-semibold shadow-sm'
          : 'text-gray-500'}"
        on:click={() => portalRole.set("tech")}
      >
        技术视角
      </button>
    </div>
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
      title={currentTheme === "dark" ? "切换浅色模式" : "切换深色模式"}
    >
      {#if currentTheme === "dark"}
        <Sun class="size-4" />
      {:else}
        <Moon class="size-4" />
      {/if}
    </button>
    <!-- User profile -->
    <button
      class="flex items-center gap-2 rounded-full border border-gray-200 bg-white px-3 py-1.5 text-sm text-gray-700 transition-colors hover:bg-gray-50"
      title="用户"
    >
      <User class="size-4" />
      <span class="hidden sm:inline">用户</span>
    </button>
  </div>
</nav>
