<script lang="ts">
  import { page } from "$app/stores";
  import DelayedCircleOutlineSpinner from "@rilldata/web-common/components/spinner/DelayedCircleOutlineSpinner.svelte";
  import {
    createAdminServiceListAuditEvents,
    createAdminServiceListProjectsForOrganization,
  } from "@rilldata/web-admin/client";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  let organization = $derived($page.params.organization);

  // Event types the backend records (admin/audit.go). Kept in display order so the
  // filter reads as a workflow rather than an alphabetical dump.
  const EVENT_TYPES = [
    "project_publish",
    "project_rollback",
    "rollback_requested",
    "rollback_rejected",
    "semantic_resource_save",
    "semantic_resource_delete",
    "resource_visibility_set",
    "edit_lock_force_release",
    "feature_access_set",
    "org_feature_defaults_set",
    "org_ai_config_set",
    "member_add",
    "member_remove",
    "member_role_change",
    "usergroup_member_add",
    "usergroup_member_remove",
  ] as const;

  const EVENT_LABELS: Record<string, () => string> = {
    project_publish: () => m.audit_event_project_publish(),
    project_rollback: () => m.audit_event_project_rollback(),
    rollback_requested: () => m.audit_event_rollback_requested(),
    rollback_rejected: () => m.audit_event_rollback_rejected(),
    semantic_resource_save: () => m.audit_event_semantic_resource_save(),
    semantic_resource_delete: () => m.audit_event_semantic_resource_delete(),
    resource_visibility_set: () => m.audit_event_resource_visibility_set(),
    edit_lock_force_release: () => m.audit_event_edit_lock_force_release(),
    feature_access_set: () => m.audit_event_feature_access_set(),
    org_feature_defaults_set: () => m.audit_event_org_feature_defaults_set(),
    org_ai_config_set: () => m.audit_event_org_ai_config_set(),
    member_add: () => m.audit_event_member_add(),
    member_remove: () => m.audit_event_member_remove(),
    member_role_change: () => m.audit_event_member_role_change(),
    usergroup_member_add: () => m.audit_event_usergroup_member_add(),
    usergroup_member_remove: () => m.audit_event_usergroup_member_remove(),
  };

  let eventTypeFilter = $state<string>("");
  let projectFilter = $state<string>("");

  let params = $derived({
    ...(eventTypeFilter ? { eventType: eventTypeFilter } : {}),
    ...(projectFilter ? { project: projectFilter } : {}),
    limit: 200,
  });

  let eventsQuery = $derived(
    createAdminServiceListAuditEvents(organization, params),
  );
  let events = $derived($eventsQuery.data?.events ?? []);

  let projectsQuery = $derived(
    createAdminServiceListProjectsForOrganization(organization),
  );
  let projects = $derived($projectsQuery.data?.projects ?? []);

  function eventLabel(type: string | undefined): string {
    if (!type) return "";
    return EVENT_LABELS[type]?.() ?? type;
  }

  function actorLabel(ev: (typeof events)[number]): string {
    return ev.actorUserName || ev.actorUserEmail || m.audit_actor_system();
  }

  function formatTime(iso: string | undefined): string {
    if (!iso) return "";
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return iso;
    return d.toLocaleString();
  }

  // Payloads are small, event-specific JSON blobs. Render them as compact
  // key=value pairs rather than raw JSON so the table stays scannable.
  function formatPayload(payload: Record<string, unknown> | undefined): string {
    if (!payload) return "";
    return Object.entries(payload)
      .map(([k, v]) => `${k}=${typeof v === "object" ? JSON.stringify(v) : String(v)}`)
      .join("  ");
  }
</script>

<!-- SYSTEM LOG (audit trail) -->
<div class="flex flex-col gap-y-6">
  <div>
    <h2 class="text-lg font-semibold text-fg-primary">{m.audit_title()}</h2>
    <p class="mt-1 text-sm text-fg-tertiary">{m.audit_desc()}</p>
  </div>

  <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
    <label class="flex items-center gap-2 text-sm text-fg-secondary">
      {m.audit_filter_event()}
      <select
        class="rounded-sm border border-gray-200 bg-surface-background px-2 py-1.5 text-sm text-fg-primary dark:border-gray-700"
        bind:value={eventTypeFilter}
      >
        <option value="">{m.audit_filter_all()}</option>
        {#each EVENT_TYPES as t (t)}
          <option value={t}>{eventLabel(t)}</option>
        {/each}
      </select>
    </label>

    <label class="flex items-center gap-2 text-sm text-fg-secondary">
      {m.audit_filter_domain()}
      <select
        class="rounded-sm border border-gray-200 bg-surface-background px-2 py-1.5 text-sm text-fg-primary dark:border-gray-700"
        bind:value={projectFilter}
      >
        <option value="">{m.audit_filter_all()}</option>
        {#each projects as p (p.id)}
          <option value={p.name}>{p.name}</option>
        {/each}
      </select>
    </label>
  </div>

  {#if $eventsQuery.isLoading}
    <DelayedCircleOutlineSpinner isLoading={true} />
  {:else if events.length === 0}
    <p class="text-sm text-fg-tertiary">{m.audit_empty()}</p>
  {:else}
    <div class="overflow-x-auto">
      <table class="w-full border-collapse text-sm">
        <thead>
          <tr class="border-b border-gray-200 text-left dark:border-gray-700">
            <th class="py-2 pr-4 font-medium text-fg-secondary whitespace-nowrap">
              {m.audit_col_time()}
            </th>
            <th class="py-2 pr-4 font-medium text-fg-secondary whitespace-nowrap">
              {m.audit_col_event()}
            </th>
            <th class="py-2 pr-4 font-medium text-fg-secondary whitespace-nowrap">
              {m.audit_col_actor()}
            </th>
            <th class="py-2 pr-4 font-medium text-fg-secondary whitespace-nowrap">
              {m.audit_col_domain()}
            </th>
            <th class="py-2 pr-4 font-medium text-fg-secondary">
              {m.audit_col_detail()}
            </th>
          </tr>
        </thead>
        <tbody>
          {#each events as ev (ev.id)}
            <tr class="border-b border-gray-100 align-top dark:border-gray-800">
              <td class="py-2 pr-4 text-fg-tertiary whitespace-nowrap">
                {formatTime(ev.createdOn)}
              </td>
              <td class="py-2 pr-4 text-fg-primary whitespace-nowrap">
                {eventLabel(ev.eventType)}
              </td>
              <td class="py-2 pr-4 text-fg-secondary whitespace-nowrap">
                {actorLabel(ev)}
              </td>
              <td class="py-2 pr-4 text-fg-secondary whitespace-nowrap">
                {ev.projectName || "—"}
              </td>
              <td class="py-2 pr-4 font-mono text-xs text-fg-tertiary break-all">
                {formatPayload(ev.payload)}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>
