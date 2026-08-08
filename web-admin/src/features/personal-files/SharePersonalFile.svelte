<script lang="ts">
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
  import * as Popover from "@rilldata/web-common/components/popover";
  import * as Tooltip from "@rilldata/web-common/components/tooltip-v2";
  import Button from "web-common/src/components/button/Button.svelte";
  import { createAdminServiceGetPersonalFile } from "@rilldata/web-admin/client";
  import { parseDocument, YAMLMap } from "yaml";
  import { fileArtifacts } from "@rilldata/web-common/features/entity-management/file-artifacts.ts";
  import { addLeadingSlash } from "@rilldata/web-common/features/entity-management/entity-mappers.ts";
  import { eventBus } from "@rilldata/web-common/lib/event-bus/event-bus.ts";

  let {
    organization,
    project,
    name,
  }: {
    organization: string;
    project: string;
    name: string;
  } = $props();

  let open = $state(false);
  let sharing = $state(false);

  let personalFileQuery = $derived(
    createAdminServiceGetPersonalFile(organization, project, name),
  );
  let { data, isPending } = $derived($personalFileQuery);

  let parsedDocument = $derived(parseDocument(data?.yaml ?? ""));
  let shared = $derived(
    (parsedDocument.get("annotations") as YAMLMap | null)?.get(
      "admin_shared",
    ) === "true",
  );

  let loading = $derived(isPending || sharing);

  async function handleShareToggle(share: boolean) {
    if (!data) return;

    sharing = true;
    try {
      let annotations = parsedDocument.get("annotations") as YAMLMap | null;
      if (!annotations) {
        annotations = new YAMLMap();
        parsedDocument.set("annotations", annotations);
      }
      annotations.set("admin_shared", share ? "true" : "false");
      const yaml = parsedDocument.toString();

      const fileArtifact = fileArtifacts.getFileArtifact(
        addLeadingSlash(data.path ?? ""),
      );
      fileArtifact.updateEditorContent(yaml);
      await fileArtifact.saveLocalContent();

      eventBus.emit("notification", {
        type: "success",
        message: share
          ? m.personal_shared_success()
          : m.personal_hidden_success(),
      });
    } catch (e) {
      console.error("Error sharing dashboard:", e);
    }
    sharing = false;
    open = false;
  }
</script>

<Popover.Root bind:open>
  <Popover.Trigger>
    {#snippet child({ props })}
      <Tooltip.Root disabled={open}>
        <Tooltip.Trigger>
          <Button
            {...props}
            type="secondary"
            selected={open}
            loading={isPending}
          >
            {m.common_share()}
          </Button>
        </Tooltip.Trigger>
        <Tooltip.Content>{m.personal_share_dashboard()}</Tooltip.Content>
      </Tooltip.Root>
    {/snippet}
  </Popover.Trigger>
  <Popover.Content align="end">
    {#if shared}
      {m.personal_shared_with_all()}
    {:else}
      {m.personal_share_confirm()}
    {/if}

    <div class="flex pt-2">
      <div class="grow"></div>
      <Button
        type="primary"
        onClick={() => handleShareToggle(!shared)}
        {loading}
        disabled={loading}
      >
        {shared ? m.common_hide() : m.common_share()}
      </Button>
    </div>
  </Popover.Content>
</Popover.Root>
