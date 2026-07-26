<script lang="ts">
  export let title: string;
  export let description: string = "";
</script>

<div>
  <h2 class="text-lg font-bold text-fg-primary">{title}</h2>
  {#if description}
    <p class="mt-0.5 text-[13px] text-fg-muted">{description}</p>
  {/if}
  <slot name="actions" />
</div>
