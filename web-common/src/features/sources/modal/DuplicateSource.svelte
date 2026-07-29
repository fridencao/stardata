<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import { onDestroy } from "svelte";
  import {
    DuplicateActions,
    duplicateSourceAction,
    duplicateSourceName,
  } from "../sources-store";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  export let onComplete: () => void;
  export let onCancel: () => void;

  function cancel() {
    $duplicateSourceName = null;
    $duplicateSourceAction = DuplicateActions.Cancel;
    onCancel();
  }

  function keepBoth() {
    $duplicateSourceName = null;
    $duplicateSourceAction = DuplicateActions.KeepBoth;
    onComplete();
  }

  function overwriteSource() {
    $duplicateSourceName = null;
    $duplicateSourceAction = DuplicateActions.Overwrite;
    onComplete();
  }

  onDestroy(() => {
    $duplicateSourceName = null;
  });
</script>

<Dialog.Description>
  {m.source_duplicate_prefix()} <b>{$duplicateSourceName}</b>
  {m.source_duplicate_suffix()}
</Dialog.Description>

<Dialog.Footer>
  <Button type="text" onClick={cancel}>{m.common_cancel()}</Button>

  <Button type="text" onClick={overwriteSource}
    >{m.source_duplicate_replace()}</Button
  >

  <Button type="primary" onClick={keepBoth}
    >{m.source_duplicate_keep_both()}</Button
  >
</Dialog.Footer>
