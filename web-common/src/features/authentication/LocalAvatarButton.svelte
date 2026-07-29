<script lang="ts">
  import { page } from "$app/stores";
  import Avatar from "@rilldata/web-common/components/avatar/Avatar.svelte";
  import * as DropdownMenu from "@rilldata/web-common/components/dropdown-menu";
  import NoUser from "@rilldata/web-common/components/icons/NoUser.svelte";
  import { EntityStatus } from "@rilldata/web-common/features/entity-management/types";
  import { initPylonChat } from "@rilldata/web-common/features/help/initPylonChat";
  import {
    createLocalServiceGetCurrentUser,
    createLocalServiceGetMetadata,
  } from "@rilldata/web-common/runtime-client/local-service";
  import {
    getStardataToken,
    clearStardataToken,
  } from "../../runtime-client/auth-token";
  import Spinner from "@rilldata/web-common/features/entity-management/Spinner.svelte";
  import ThemeToggle from "@rilldata/web-common/features/themes/ThemeToggle.svelte";
  import { DOCS_BASE_URL } from "@rilldata/web-common/lib/stardata-links";

  $: user = createLocalServiceGetCurrentUser({
    query: {
      // refetch in case user does a login/logout from outside of the StarData UI
      refetchOnWindowFocus: true,
    },
  });
  $: metadata = createLocalServiceGetMetadata();

  let loginUrl: string;
  $: if ($metadata.data?.loginUrl) {
    const u = new URL($metadata.data.loginUrl);
    u.searchParams.set(
      "redirect",
      `${window.location.origin}${window.location.pathname}`,
    );
    loginUrl = u.toString();
  }

  let logoutUrl: string;
  $: if ($metadata.data?.loginUrl) {
    const u = new URL($metadata.data.loginUrl + "/logout");
    u.searchParams.set("redirect", $page.url.href);
    logoutUrl = u.toString();
  }

  $: loggedIn = $user.isSuccess && $user.data?.user;

  $: if ($user.data?.user) {
    initPylonChat($user.data.user);
  }
  function handlePylon() {
    window.Pylon("show");
  }

  // Self-hosted auth: when a StarData JWT is present, logout is a client-side
  // concern — clear the token and return to /login. For the legacy cloud
  // flow (no token) we let the default /auth/logout link proceed.
  function handleLogout(e: MouseEvent) {
    if (getStardataToken() !== null) {
      e.preventDefault();
      clearStardataToken();
      window.location.href = "/login";
    }
  }

  let photoUrlErrored = false;
</script>

{#if ($user.isLoading || $metadata.isLoading) && !$user.error && !$metadata.error}
  <div class="flex flex-row items-center h-7 mx-1.5">
    <Spinner size="16px" status={EntityStatus.Running} />
  </div>
{:else}
  <DropdownMenu.Root>
    <DropdownMenu.Trigger
      class="flex-none w-7"
      aria-label="Avatar logged {loggedIn ? 'in' : 'out'}"
    >
      {#if loggedIn && !photoUrlErrored && $user.data && $metadata.data}
        <Avatar
          src={$user.data?.user?.photoUrl}
          alt={$user.data?.user?.displayName || $user.data?.user?.email}
          avatarSize="h-7 w-7"
        />
      {:else}
        <NoUser />
      {/if}
    </DropdownMenu.Trigger>
    <DropdownMenu.Content class="p-1">
      <ThemeToggle />
      <DropdownMenu.Separator />

      <DropdownMenu.Item
        href={DOCS_BASE_URL}
        target="_blank"
        rel="noreferrer noopener"
      >
        Documentation
      </DropdownMenu.Item>

      {#if loggedIn}
        <DropdownMenu.Item onclick={handlePylon}>
          Contact StarData support
        </DropdownMenu.Item>
        <DropdownMenu.Separator />
        <DropdownMenu.Item href={logoutUrl} onclick={handleLogout} rel="external">
          Logout
        </DropdownMenu.Item>
      {:else}
        <DropdownMenu.Separator />
        <DropdownMenu.Item href={loginUrl} rel="external">
          Log in / Sign up
        </DropdownMenu.Item>
      {/if}
    </DropdownMenu.Content>
  </DropdownMenu.Root>
{/if}
