<script lang="ts">
  import { page } from "$app/stores";
  import { LOCAL_HOST } from "../../../lib/runtime-client";
  import {
    getStardataToken,
    setStardataToken,
    decodeStardataToken,
  } from "@rilldata/web-common/runtime-client/auth-token";
  import { defaultHome } from "../../../features/portal/user-spaces";
  import { onMount } from "svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  let username = "";
  let password = "";
  let error = "";
  let loading = false;
  let oidcError = "";

  // Already authenticated? Bounce straight to the app (or the requested redirect).
  if (getStardataToken()) {
    const redirect = $page.url.searchParams.get("redirect");
    window.location.href = redirect || defaultHome();
  }

  // Extract token from OIDC callback ?token= and store it.
  onMount(() => {
    const tokenParam = $page.url.searchParams.get("token");
    if (tokenParam) {
      setStardataToken(tokenParam);
      const url = new URL(window.location.href);
      url.searchParams.delete("token");
      window.history.replaceState({}, "", url.pathname);
      const oidcSpaces = decodeStardataToken(tokenParam)?.spaces;
      window.location.href = defaultHome(oidcSpaces);
    }
  });

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = "";
    if (!username || !password) {
      error = m.login_error_empty();
      return;
    }
    loading = true;
    try {
      const resp = await fetch(`${LOCAL_HOST}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });
      if (!resp.ok) {
        error = m.login_error_bad_credentials();
        return;
      }
      const data = (await resp.json()) as { token?: string };
      if (!data.token) {
        error = m.login_error_no_token();
        return;
      }
      setStardataToken(data.token);
      const redirect = $page.url.searchParams.get("redirect");
      const loginSpaces = decodeStardataToken(data.token)?.spaces;
      window.location.href = redirect || defaultHome(loginSpaces);
    } catch {
      error = m.login_error_network();
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>{m.login_page_title()} · StarData</title>
</svelte:head>

<div class="flex h-screen w-screen items-center justify-center bg-gray-50">
  <div
    class="w-full max-w-sm rounded-xl border border-gray-200 bg-white p-8 shadow-sm"
  >
    <div class="mb-6 text-center">
      <h1 class="text-2xl font-semibold text-gray-900">StarData</h1>
      <p class="mt-1 text-sm text-gray-500">{m.login_subtitle()}</p>
    </div>

    {#if error}
      <div class="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">
        {error}
      </div>
    {/if}

    <form onsubmit={handleSubmit} class="flex flex-col gap-4">
      {#if oidcError}
        <div class="mb-2 rounded-md bg-red-50 px-3 py-2 text-sm text-red-700">
          {oidcError}
        </div>
      {/if}
      <label class="flex flex-col gap-1 text-sm font-medium text-gray-700">
        {m.login_username()}
        <input
          bind:value={username}
          type="text"
          autocomplete="username"
          class="rounded-md border border-gray-300 px-3 py-2 text-sm outline-none focus:border-gray-900"
          placeholder={m.login_username()}
        />
      </label>
      <label class="flex flex-col gap-1 text-sm font-medium text-gray-700">
        {m.login_password()}
        <input
          bind:value={password}
          type="password"
          autocomplete="current-password"
          class="rounded-md border border-gray-300 px-3 py-2 text-sm outline-none focus:border-gray-900"
          placeholder={m.login_password()}
        />
      </label>
      <button
        type="submit"
        disabled={loading}
        class="mt-2 rounded-md bg-gray-900 px-3 py-2 text-sm font-medium text-white hover:bg-gray-800 disabled:opacity-50"
      >
        {loading ? m.login_submitting() : m.login_submit()}
      </button>
    </form>
  </div>
</div>
