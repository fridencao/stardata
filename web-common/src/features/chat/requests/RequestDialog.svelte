<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
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
      errorMessage = "请填写想问的问题";
      return;
    }
    saving = true;
    errorMessage = "";
    try {
      await appendRequest(runtimeClient, question.trim(), note);
      eventBus.emit("notification", {
        type: "success",
        message: "需求已提交，技术团队会在 Studio 中处理",
      });
      open = false;
    } catch {
      errorMessage = "提交失败，请重试";
    }
    saving = false;
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content>
    <Dialog.Title>提数据需求</Dialog.Title>
    <Dialog.Description>
      没有得到想要的答案？把问题提给技术团队，配置好后就能直接问了。
    </Dialog.Description>

    <div class="flex flex-col gap-3 py-2">
      <label class="flex flex-col gap-1 text-sm">
        想问的问题
        <textarea
          class="min-h-[72px] rounded-md border border-gray-300 px-2 py-1.5 text-sm"
          bind:value={question}
        ></textarea>
      </label>
      <label class="flex flex-col gap-1 text-sm">
        补充说明（可选）
        <input
          class="rounded-md border border-gray-300 px-2 py-1.5 text-sm"
          placeholder="例如：月会要用"
          bind:value={note}
        />
      </label>
      {#if errorMessage}
        <p class="text-xs text-red-600">{errorMessage}</p>
      {/if}
    </div>

    <Dialog.Footer class="gap-x-2">
      <Button type="tertiary" onClick={() => (open = false)}>取消</Button>
      <Button type="primary" loading={saving} onClick={submit}>提交需求</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
