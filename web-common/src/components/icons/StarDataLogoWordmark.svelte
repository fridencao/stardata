<script lang="ts">
  /**
   * Brand mark used in the global header and authentication pages.
   * Renders only the StarData sparkle glyph (no wordmark text).
   */
  export let size: "sm" | "md" | "lg" = "md";

  const iconSizeClasses = {
    sm: "size-[22px] text-xs",
    md: "size-[26px] text-sm",
    lg: "size-9 text-base",
  };
</script>

<span
  class="grid {iconSizeClasses[size]} place-items-center rounded-lg bg-primary-600 text-white"
>
  ✦
</span>
