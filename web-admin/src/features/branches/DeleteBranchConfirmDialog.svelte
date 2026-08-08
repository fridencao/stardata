<script lang="ts">
  import {
    AlertDialog,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
    AlertDialogTrigger,
  } from "@rilldata/web-common/components/alert-dialog/index.js";
  import { Button } from "@rilldata/web-common/components/button/index.js";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  let {
    open = $bindable(false),
    branch,
    editable = false,
    onConfirm,
  }: {
    open: boolean;
    branch: string;
    editable?: boolean;
    onConfirm: () => void;
  } = $props();
</script>

<AlertDialog bind:open>
  <AlertDialogTrigger>
    {#snippet child({ props })}
      <div {...props} class="hidden"></div>
    {/snippet}
  </AlertDialogTrigger>
  <AlertDialogContent>
    <AlertDialogHeader>
      <AlertDialogTitle>{m.branch_delete_confirm_title()}</AlertDialogTitle>
      <AlertDialogDescription>
        <div class="mt-1">
          {m.branch_delete_confirm_prefix()}
          <span class="font-mono text-xs font-medium">{branch}</span>
          {m.branch_delete_confirm_suffix()}
          {#if editable}
            {m.branch_delete_remote_warning()}
          {/if}
        </div>
        <div class="mt-2 font-medium text-fg-primary">
          {m.branch_delete_cannot_undo()}
        </div>
      </AlertDialogDescription>
    </AlertDialogHeader>
    <AlertDialogFooter>
      <Button
        type="tertiary"
        onClick={() => {
          open = false;
        }}
      >
        {m.common_cancel()}
      </Button>
      <Button
        type="destructive"
        onClick={() => {
          open = false;
          onConfirm();
        }}>{m.branch_delete_confirm_yes()}</Button
      >
    </AlertDialogFooter>
  </AlertDialogContent>
</AlertDialog>
