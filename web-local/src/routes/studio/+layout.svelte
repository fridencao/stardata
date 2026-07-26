<script lang="ts">
  import StudioSidebar from "../../features/studio/StudioSidebar.svelte";
</script>

<div class="flex h-full min-h-0 bg-gray-100">
  <StudioSidebar />
  <main class="min-h-0 flex-1 overflow-y-auto px-9 py-7">
    <slot />
  </main>
</div>
