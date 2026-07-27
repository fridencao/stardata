<script lang="ts">
  import PortalNav from "../../features/portal/PortalNav.svelte";
  import StudioTabs from "../../features/studio/StudioTabs.svelte";
</script>

<div class="flex h-full min-h-0 flex-col bg-app-surface">
  <PortalNav />
  <StudioTabs />
  <main class="w-full flex-1 overflow-y-auto p-8 xl:max-w-6xl xl:mx-auto">
    <slot />
  </main>
</div>
