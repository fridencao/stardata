<script lang="ts">
  import PortalNav from "../../features/portal/PortalNav.svelte";
  import PortalTabs from "../../features/portal/PortalTabs.svelte";
</script>

<div class="flex h-full min-h-0 flex-col bg-gray-50">
  <PortalNav />
  <PortalTabs />
  <main class="min-h-0 flex-1 overflow-hidden">
    <slot />
  </main>
</div>
