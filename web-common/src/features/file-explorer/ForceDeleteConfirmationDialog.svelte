<script lang="ts">
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import * as AlertDialog from "@rilldata/web-common/components/alert-dialog/index";
  import { Button } from "@rilldata/web-common/components/button";

  export let open: boolean;
  export let onDelete: () => void;

  function handleClose() {
    open = false;
  }
</script>

<AlertDialog.Root
  {open}
  onOpenChange={(open) => {
    if (!open) {
      handleClose();
    }
  }}
>
  <AlertDialog.Content>
    <AlertDialog.Title>
      {m.file_delete_folder_title()}
    </AlertDialog.Title>

    <AlertDialog.Description>
      {m.file_delete_folder_description()}
    </AlertDialog.Description>

    <AlertDialog.Footer>
      <AlertDialog.Action>
        {#snippet child({ props })}
          <Button
            {...props}
            large
            onClick={() => {
              handleClose();
              onDelete();
            }}
            type="destructive"
          >
            {m.common_delete()}
          </Button>
        {/snippet}
      </AlertDialog.Action>

      <AlertDialog.Cancel>
        {#snippet child({ props })}
          <Button {...props} large onClick={handleClose} type="tertiary"
            >{m.common_cancel()}</Button
          >
        {/snippet}
      </AlertDialog.Cancel>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>
