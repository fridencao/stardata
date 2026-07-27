<script lang="ts">
  // Minimal shared header used across the /studio workbench pages.
  // (The original Rill component was dropped in the fork; this stub
  // restores the title + description + slot contract the pages rely on.)
  export let title: string = "";
  export let description: string = "";
</script>

<div class="mb-8 border-b border-gray-200 pb-6">
  <h1 class="text-2xl font-semibold text-gray-900">
    {title}
  </h1>
  {#if description}
    <p class="mt-1.5 text-sm text-gray-500">
      {description}
    </p>
  {/if}
  <slot />
</div>
