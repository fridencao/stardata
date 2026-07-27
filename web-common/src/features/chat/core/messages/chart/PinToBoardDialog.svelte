<script lang="ts">
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import * as Dialog from "@rilldata/web-common/components/dialog";
  import { useDashboards } from "@rilldata/web-common/features/dashboards/listing/selectors";
  import { getName } from "@rilldata/web-common/features/entity-management/name-utils";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus";
  import {
    runtimeServiceGetFile,
    runtimeServicePutFile,
  } from "@rilldata/web-common/runtime-client";
  import { useRuntimeClient } from "@rilldata/web-common/runtime-client/v2";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import type { ChartType } from "../../../../components/charts/types";
  import { appendChartToCanvasYaml, newCanvasYaml } from "./pin-to-board";

  export let open = false;
  export let chartType: ChartType;
  export let chartSpec: unknown;

  const NEW_BOARD = "__new__";

  const runtimeClient = useRuntimeClient();

  $: dashboardsQuery = useDashboards(runtimeClient);
  $: boards = ($dashboardsQuery.data ?? []).filter((res) => res.canvas);

  let selectedName = NEW_BOARD;
  let newBoardName = "";
  let saving = false;
  let errorMessage = "";
  let touched = false;

  // Default to first existing board when opened
  $: if (boards.length > 0 && selectedName === NEW_BOARD && !touched) {
    selectedName = boards[0].meta?.name?.name ?? NEW_BOARD;
  }

  async function pin() {
    saving = true;
    errorMessage = "";
    const spec = chartSpec as Record<string, unknown>;
    try {
      let boardName: string;
      if (selectedName !== NEW_BOARD) {
        const board = boards.find((b) => b.meta?.name?.name === selectedName);
        const path = board?.meta?.filePaths?.[0];
        if (!path) throw new Error(m.chat_pin_board_file_missing());
        const file = await runtimeServiceGetFile(runtimeClient, { path });
        const blob = appendChartToCanvasYaml(file.blob ?? "", chartType, spec);
        await runtimeServicePutFile(runtimeClient, {
          path,
          blob,
          create: false,
          createOnly: false,
        });
        boardName = selectedName;
      } else {
        const existing = boards.map((b) => b.meta?.name?.name ?? "");
        const safeName = (newBoardName.trim() || m.chat_pin_default_board_name())
          .replace(/[^a-zA-Z0-9_一-龥]/g, "_");
        boardName = getName(safeName, existing);
        await runtimeServicePutFile(runtimeClient, {
          path: `dashboards/${boardName}.yaml`,
          blob: newCanvasYaml(boardName, chartType, spec),
          create: true,
          createOnly: true,
        });
      }
      eventBus.emit("notification", {
        type: "success",
        message: m.chat_pin_success(),
        link: { text: m.chat_pin_view_board(), href: `/boards/${boardName}` },
      });
      open = false;
    } catch (e) {
      errorMessage = e instanceof Error ? e.message : m.chat_pin_write_failed();
    }
    saving = false;
  }
</script>

<Dialog.Root bind:open>
  <Dialog.Content>
    <Dialog.Title>{m.chat_pin_dialog_title()}</Dialog.Title>

    <div class="flex flex-col gap-3 py-2">
      <label class="flex flex-col gap-1 text-sm">
        {m.chat_pin_select_board()}
        <select
          class="rounded-md border border-gray-300 px-2 py-1.5 text-sm"
          bind:value={selectedName}
          onchange={() => (touched = true)}
        >
          {#each boards as board (board.meta?.name?.name)}
            <option value={board.meta?.name?.name}>
              {board.canvas?.spec?.displayName || board.meta?.name?.name}
            </option>
          {/each}
          <option value={NEW_BOARD}>{m.chat_pin_new_board_option()}</option>
        </select>
      </label>

      {#if selectedName === NEW_BOARD}
        <label class="flex flex-col gap-1 text-sm">
          {m.chat_pin_new_board_name()}
          <input
            class="rounded-md border border-gray-300 px-2 py-1.5 text-sm"
            placeholder={m.chat_pin_new_board_placeholder()}
            bind:value={newBoardName}
          />
        </label>
      {/if}

      {#if errorMessage}
        <p class="text-xs text-red-600">{errorMessage}</p>
      {/if}
    </div>

    <Dialog.Footer class="gap-x-2">
      <Button type="tertiary" onClick={() => (open = false)}>{m.common_cancel()}</Button>
      <Button type="primary" loading={saving} onClick={pin}>{m.chat_pin_dialog_title()}</Button>
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>
