<script lang="ts">
  import {
    getStardataToken,
    clearStardataToken,
    decodeStardataToken,
  } from "@rilldata/web-common/runtime-client/auth-token";
  import { ChevronDown, LogOut } from "lucide-svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  $: hasToken = !!getStardataToken();
  $: userClaims = decodeStardataToken(getStardataToken());
  let menuOpen = false;

  function handleLogout() {
    clearStardataToken();
    window.location.href = "/login";
  }
</script>

{#if hasToken}
  <div class="relative">
    <button
      class="flex items-center gap-2 rounded-full border border-gray-200 bg-surface-background px-2.5 py-1.5 text-sm text-gray-700 transition-colors hover:bg-gray-50"
      onclick={() => (menuOpen = !menuOpen)}
      title={userClaims?.name || userClaims?.id || m.portal_nav_user_fallback()}
    >
      <span
        class="grid size-6 place-items-center rounded-full bg-primary-600 text-xs font-semibold text-white"
      >
        {(userClaims?.name || userClaims?.id || "?").slice(0, 1).toUpperCase()}
      </span>
      <span class="hidden max-w-[120px] truncate sm:inline"
        >{userClaims?.name || userClaims?.id}</span
      >
      <ChevronDown class="hidden size-3.5 text-gray-400 sm:inline" />
    </button>

    {#if menuOpen}
      <!-- click-away backdrop -->
      <div
        class="fixed inset-0 z-40"
        role="button"
        tabindex="-1"
        aria-label={m.portal_nav_close_user_menu()}
        onclick={() => (menuOpen = false)}
        onkeydown={(e) => {
          if (e.key === "Escape" || e.key === "Enter" || e.key === " ") {
            menuOpen = false;
          }
        }}
      ></div>
      <div
        class="absolute right-0 top-[calc(100%+8px)] z-50 w-56 rounded-xl border border-gray-200 bg-surface-overlay p-1.5 shadow-lg"
      >
        <div class="px-3 py-2">
          <div
            class="truncate text-sm font-semibold text-gray-900"
            >{userClaims?.name || userClaims?.id}</div
          >
          {#if userClaims?.email}
            <div
              class="truncate text-xs text-gray-500"
              >{userClaims.email}</div
            >
          {/if}
          <div
            class="mt-1.5 inline-block rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-600"
          >
            {userClaims?.admin ? m.portal_nav_role_admin() : m.portal_nav_role_user()}
          </div>
        </div>
        <div class="my-1 h-px bg-gray-100"></div>
        <button
          class="flex w-full items-center gap-2 rounded-lg px-3 py-2 text-sm text-gray-700 transition-colors hover:bg-gray-100"
          onclick={handleLogout}
        >
          <LogOut class="size-4" />
          {m.portal_nav_logout()}
        </button>
      </div>
    {/if}
  </div>
{/if}
