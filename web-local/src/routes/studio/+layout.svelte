<script lang="ts">
  import StudioTabs from "../../features/studio/StudioTabs.svelte";
</script>

<div class="flex min-h-0 flex-col bg-app-surface">
  <StudioTabs />
  <main class="flex-1 overflow-y-auto p-8 xl:max-w-6xl xl:mx-auto">
    <slot />
  </main>
</div>
