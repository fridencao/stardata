<script lang="ts">
  export let variant: "success" | "error" | "warning" | "neutral" | "info" = "neutral";
  export let size: "sm" | "md" = "md";
</script>

<span
  class="inline-flex items-center rounded-md font-semibold no-underline {variantClass} {sizeClass}"
>
  <slot />
</span>

<style>
  .variant-success { @apply bg-green-50 text-green-700; }
  .variant-error   { @apply bg-red-50 text-red-600; }
  .variant-warning { @apply bg-yellow-50 text-yellow-700; }
  .variant-neutral { @apply bg-gray-100 text-gray-500; }
  .variant-info    { @apply bg-primary-50 text-primary-700; }

  .size-sm { @apply px-1.5 py-0.5 text-[10.5px]; }
  .size-md { @apply px-2 py-0.5 text-[11px]; }

  :global(.dark) .variant-success { @apply bg-green-900/30 text-green-300; }
  :global(.dark) .variant-error   { @apply bg-red-900/30 text-red-300; }
  :global(.dark) .variant-neutral { @apply bg-gray-700/50 text-gray-400; }
  :global(.dark) .variant-info    { @apply bg-primary-900/30 text-primary-300; }
</style>
