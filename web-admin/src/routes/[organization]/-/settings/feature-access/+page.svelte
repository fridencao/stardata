<script lang="ts">
  import { page } from "$app/stores";
  import SettingsContainer from "@rilldata/web-admin/features/organizations/settings/SettingsContainer.svelte";
  import Switch from "@rilldata/web-common/components/forms/Switch.svelte";
  import Label from "@rilldata/web-common/components/forms/Label.svelte";
  import DelayedCircleOutlineSpinner from "@rilldata/web-common/components/spinner/DelayedCircleOutlineSpinner.svelte";
  import {
    createAdminServiceListFeatureAccess,
    createAdminServiceSetFeatureAccess,
    createAdminServiceSetOrgFeatureDefaults,
    createAdminServiceListProjectsForOrganization,
    getAdminServiceListFeatureAccessQueryKey,
  } from "@rilldata/web-admin/client";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  let organization = $derived($page.params.organization);

  // The six governed features, in display order.
  const FEATURES = [
    { key: "chat", label: () => m.feature_access_chat() },
    { key: "dashboards", label: () => m.feature_access_dashboards() },
    { key: "reports", label: () => m.feature_access_reports() },
    { key: "alerts", label: () => m.feature_access_alerts() },
    { key: "studio", label: () => m.feature_access_studio() },
    { key: "admin", label: () => m.feature_access_admin() },
  ] as const;

  // Scope: "org" edits org-level per-subject overrides; "project" scopes to one project.
  let scope = $state<"org" | "project">("org");
  let selectedProject = $state<string>("");

  let listParams = $derived(
    scope === "project" && selectedProject
      ? { project: selectedProject }
      : undefined,
  );

  let listQuery = $derived(
    createAdminServiceListFeatureAccess(organization, listParams),
  );
  let orgDefaults = $derived($listQuery.data?.orgDefaults ?? []);
  let subjects = $derived($listQuery.data?.subjects ?? []);

  let projectsQuery = $derived(
    createAdminServiceListProjectsForOrganization(organization),
  );
  let projects = $derived($projectsQuery.data?.projects ?? []);

  const setOrgDefaultsMutation = createAdminServiceSetOrgFeatureDefaults();
  const setFeatureAccessMutation = createAdminServiceSetFeatureAccess();

  let isMutating = $derived(
    $setOrgDefaultsMutation.isPending || $setFeatureAccessMutation.isPending,
  );

  function defaultFor(key: string): boolean {
    return orgDefaults.find((d) => d.featureKey === key)?.granted ?? true;
  }

  function effectiveFor(
    features: Record<string, boolean> | undefined,
    key: string,
  ): boolean {
    return features?.[key] ?? true;
  }

  async function refetchList() {
    void queryClient.refetchQueries({
      queryKey: getAdminServiceListFeatureAccessQueryKey(
        organization,
        listParams,
      ),
    });
  }

  async function updateOrgDefault(key: string, granted: boolean) {
    await $setOrgDefaultsMutation.mutateAsync({
      org: organization,
      data: { features: [{ featureKey: key, granted }] },
    });
    await refetchList();
  }

  async function updateSubjectFeature(
    subjectType: string,
    subjectId: string,
    key: string,
    granted: boolean,
  ) {
    await $setFeatureAccessMutation.mutateAsync({
      org: organization,
      data: {
        project: scope === "project" ? selectedProject : undefined,
        subjectType,
        subjectId,
        features: [{ featureKey: key, granted }],
      },
    });
    await refetchList();
  }

  function subjectTypeLabel(type?: string): string {
    return type === "group"
      ? m.feature_access_subject_group()
      : m.feature_access_subject_user();
  }
</script>

<!-- FEATURE ACCESS -->
<div class="flex flex-col gap-y-6">
  <div>
    <h2 class="text-lg font-semibold text-fg-primary">
      {m.feature_access_title()}
    </h2>
    <p class="mt-1 text-sm text-fg-tertiary">{m.feature_access_desc()}</p>
  </div>

  <!-- Organization defaults -->
  <SettingsContainer title={m.feature_access_org_defaults()}>
    <p class="mb-3 text-sm text-fg-tertiary">
      {m.feature_access_org_defaults_desc()}
    </p>
    <div class="grid grid-cols-1 gap-x-8 gap-y-3 sm:grid-cols-2 lg:grid-cols-3">
      {#each FEATURES as f (f.key)}
        <div class="flex items-center justify-between gap-x-3">
          <Label for={`org-default-${f.key}`} class="font-normal text-fg-secondary">
            {f.label()}
          </Label>
          <Switch
            id={`org-default-${f.key}`}
            checked={defaultFor(f.key)}
            onclick={() => updateOrgDefault(f.key, !defaultFor(f.key))}
          />
        </div>
      {/each}
    </div>
  </SettingsContainer>

  <!-- Per-subject access -->
  <SettingsContainer title={m.feature_access_per_subject()}>
    <p class="mb-4 text-sm text-fg-tertiary">
      {m.feature_access_per_subject_desc()}
    </p>

    <!-- Scope selector -->
    <div class="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center">
      <Label class="font-medium text-fg-secondary">
        {m.feature_access_scope_label()}
      </Label>
      <div class="inline-flex rounded-sm border border-gray-200 dark:border-gray-700">
        <button
          type="button"
          class="px-3 py-1.5 text-sm {scope === 'org'
            ? 'bg-primary-400 text-white'
            : 'text-fg-secondary hover:bg-surface-subtle'}"
          onclick={() => (scope = 'org')}
        >
          {m.feature_access_scope_org()}
        </button>
        <button
          type="button"
          class="px-3 py-1.5 text-sm {scope === 'project'
            ? 'bg-primary-400 text-white'
            : 'text-fg-secondary hover:bg-surface-subtle'}"
          onclick={() => (scope = 'project')}
        >
          {m.feature_access_scope_project()}
        </button>
      </div>
      {#if scope === "project"}
        <select
          class="ml-0 rounded-sm border border-gray-200 bg-surface-background px-2 py-1.5 text-sm text-fg-primary dark:border-gray-700 sm:ml-2"
          bind:value={selectedProject}
        >
          <option value="" disabled selected={selectedProject === ""}>
            {m.feature_access_scope_project()}
          </option>
          {#each projects as p (p.id)}
            <option value={p.name}>{p.name}</option>
          {/each}
        </select>
      {/if}
    </div>

    <!-- Matrix -->
    {#if scope === "project" && selectedProject === ""}
      <p class="text-sm text-fg-tertiary">{m.feature_access_select_project()}</p>
    {:else if $listQuery.isLoading}
      <DelayedCircleOutlineSpinner isLoading={true} />
    {:else if subjects.length === 0}
      <p class="text-sm text-fg-tertiary">{m.settings_none()}</p>
    {:else}
      <div class="overflow-x-auto">
        <table class="w-full border-collapse text-sm">
          <thead>
            <tr class="border-b border-gray-200 text-left dark:border-gray-700">
              <th class="py-2 pr-4 font-medium text-fg-secondary">User / Group</th>
              <th class="py-2 pr-4 font-medium text-fg-secondary">Type</th>
              {#each FEATURES as f (f.key)}
                <th class="py-2 pr-2 text-center font-medium text-fg-secondary">
                  {f.label()}
                </th>
              {/each}
            </tr>
          </thead>
          <tbody>
            {#each subjects as s (s.subjectId)}
              <tr class="border-b border-gray-100 dark:border-gray-800">
                <td class="py-2 pr-4 text-fg-primary">{s.subjectName}</td>
                <td class="py-2 pr-4 text-fg-tertiary">
                  {subjectTypeLabel(s.subjectType)}
                </td>
                {#each FEATURES as f (f.key)}
                  <td class="py-2 text-center">
                    <Switch
                      checked={effectiveFor(s.features, f.key)}
                      onclick={() =>
                        updateSubjectFeature(
                          s.subjectType ?? "user",
                          s.subjectId ?? "",
                          f.key,
                          !effectiveFor(s.features, f.key),
                        )}
                    />
                  </td>
                {/each}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}

    {#if isMutating}
      <div class="mt-3">
        <DelayedCircleOutlineSpinner isLoading={true} />
      </div>
    {/if}
  </SettingsContainer>
</div>
