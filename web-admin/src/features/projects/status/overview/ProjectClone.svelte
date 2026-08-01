<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import CopyableCodeBlock from "@rilldata/web-common/components/calls-to-action/CopyableCodeBlock.svelte";
  import * as Popover from "@rilldata/web-common/components/popover";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  let open = false;

  export let organization: string;
  export let project: string;
  export let disabled: boolean = false;

  // CLI commands
  $: cloneCommand = `rill project clone --org ${organization} ${project}`;
</script>

<Popover.Root bind:open>
  <Popover.Trigger>
    {#snippet child({ props })}
      <Button {...props} type="secondary" {disabled}
        >{m.status_download_project()}</Button
      >
    {/snippet}
  </Popover.Trigger>

  <Popover.Content class="w-[380px]" align="end" sideOffset={8}>
    <div class="flex flex-col gap-y-3">
      <span class="text-sm text-fg-secondary">
        {m.status_clone_description()} {m.status_learn_more()}
      </span>

      <div class="flex flex-col gap-y-2">
        <CopyableCodeBlock code={cloneCommand} />
      </div>
    </div>
  </Popover.Content>
</Popover.Root>
