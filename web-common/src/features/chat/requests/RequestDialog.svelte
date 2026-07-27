<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import { appendRequest } from "./requests-file";

  export let open = false;
  export let defaultQuestion = "";

  const runtimeClient = useRuntimeClient();

  let question = "";
  let note = "";
  let saving = false;
  let errorMessage = "";

  // Reset form on each dialog open
  $: if (open) {
    question = defaultQuestion;
    note = "";
    errorMessage = "";
  }

  async function submit() {
    if (!question.trim()) {
      errorMessage = m.chat_request_error_empty();
      return;
    }
    saving = true;
    errorMessage = "";
    try {
      await appendRequest(runtimeClient, question.trim(), note);
      eventBus.emit("notification", {
        type: "success",
        message: m.chat_request_submitted(),
      });
      open = false;
    } catch {
      errorMessage = m.chat_request_failed();
    }
    saving = false;
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content>
    <Dialog.Title>{m.chat_request_dialog_title()}</Dialog.Title>
    <Dialog.Description>
      {m.chat_request_dialog_desc()}
    </Dialog.Description>

    <div class="flex flex-col gap-3 py-2">
      <label class="flex flex-col gap-1 text-sm">
        {m.chat_request_question_label()}
        <textarea
          class="min-h-[72px] rounded-md border border-gray-300 px-2 py-1.5 text-sm"
          bind:value={question}
        ></textarea>
      </label>
      <label class="flex flex-col gap-1 text-sm">
        {m.chat_request_note_label()}
        <input
          class="rounded-md border border-gray-300 px-2 py-1.5 text-sm"
          placeholder={m.chat_request_note_placeholder()}
          bind:value={note}
        />
      </label>
      {#if errorMessage}
        <p class="text-xs text-red-600">{errorMessage}</p>
      {/if}
    </div>

    <Dialog.Footer class="gap-x-2">
      <Button type="tertiary" onClick={() => (open = false)}>{m.common_cancel()}</Button>
      <Button type="primary" loading={saving} onClick={submit}>{m.chat_request_submit()}</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
