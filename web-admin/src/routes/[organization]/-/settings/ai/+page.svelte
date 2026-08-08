<script lang="ts">
  import { page } from "$app/stores";
  import { Bot } from "lucide-svelte";
  import SettingsContainer from "@rilldata/web-admin/features/organizations/settings/SettingsContainer.svelte";
  import Button from "@rilldata/web-common/components/button/Button.svelte";
  import Label from "@rilldata/web-common/components/forms/Label.svelte";
  import Input from "@rilldata/web-common/components/forms/Input.svelte";
  import DelayedCircleOutlineSpinner from "@rilldata/web-common/components/spinner/DelayedCircleOutlineSpinner.svelte";
  import {
    createAdminServiceGetOrgAIConfig,
    createAdminServiceSetOrgAIConfig,
    createAdminServiceDeleteOrgAIConfig,
    createAdminServiceTestOrgAIConfig,
    getAdminServiceGetOrgAIConfigQueryKey,
  } from "@rilldata/web-admin/client";
  import { queryClient } from "@rilldata/web-common/lib/svelte-query/globalQueryClient";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  // AI configuration is org-scoped. When no override is set here the deployment-wide
  // env-var config (see cli/cmd/admin/start.go) applies to every completion for
  // this org. Editing does not require a service restart — admin/ai_config.go
  // invalidates the cached driver handle after each write.

  let organization = $derived($page.params.organization);

  const DRIVERS = [
    { value: "openai", label: () => m.settings_ai_driver_openai() },
    { value: "deepseek", label: () => m.settings_ai_driver_deepseek() },
  ] as const;

  let getQuery = $derived(createAdminServiceGetOrgAIConfig(organization));
  let stored = $derived($getQuery.data?.config);
  let defaultDriver = $derived($getQuery.data?.defaultDriver ?? "");

  // Form state. Prefilled from the stored config on first successful fetch.
  let driver = $state<string>("");
  let baseUrl = $state<string>("");
  let model = $state<string>("");
  let apiKey = $state<string>("");

  let hydrated = false;
  $effect(() => {
    if (!$getQuery.isSuccess || hydrated) return;
    hydrated = true;
    driver = stored?.driver ?? DRIVERS[0].value;
    baseUrl = stored?.baseUrl ?? "";
    model = stored?.model ?? "";
    apiKey = "";
  });

  const setMutation = createAdminServiceSetOrgAIConfig();
  const deleteMutation = createAdminServiceDeleteOrgAIConfig();
  const testMutation = createAdminServiceTestOrgAIConfig();

  let banner = $state<{ tone: "ok" | "err"; text: string } | null>(null);

  async function refetch() {
    await queryClient.refetchQueries({
      queryKey: getAdminServiceGetOrgAIConfigQueryKey(organization),
    });
  }

  async function save() {
    try {
      await $setMutation.mutateAsync({
        org: organization,
        data: { driver, baseUrl, model, apiKey },
      });
      apiKey = "";
      banner = { tone: "ok", text: m.settings_ai_save_ok() };
      await refetch();
    } catch (e: any) {
      banner = { tone: "err", text: e?.response?.data?.message ?? String(e) };
    }
  }

  async function reset() {
    if (!confirm(m.settings_ai_reset_confirm())) return;
    try {
      await $deleteMutation.mutateAsync({ org: organization });
      driver = defaultDriver || DRIVERS[0].value;
      baseUrl = "";
      model = "";
      apiKey = "";
      hydrated = true;
      banner = { tone: "ok", text: m.settings_ai_reset_ok() };
      await refetch();
    } catch (e: any) {
      banner = { tone: "err", text: e?.response?.data?.message ?? String(e) };
    }
  }

  async function test() {
    try {
      const res = await $testMutation.mutateAsync({
        org: organization,
        data: { driver, baseUrl, model, apiKey },
      });
      if (res.ok) {
        banner = {
          tone: "ok",
          text: m.settings_ai_test_ok({ provider: res.provider ?? driver }),
        };
      } else {
        banner = {
          tone: "err",
          text: m.settings_ai_test_fail({ message: res.message ?? "" }),
        };
      }
    } catch (e: any) {
      banner = {
        tone: "err",
        text: m.settings_ai_test_fail({
          message: e?.response?.data?.message ?? String(e),
        }),
      };
    }
  }

  let hasStored = $derived(!!stored);
  let busy = $derived(
    $setMutation.isPending || $deleteMutation.isPending || $testMutation.isPending,
  );

  function formatTime(iso: string | undefined): string {
    if (!iso) return "";
    const d = new Date(iso);
    return Number.isNaN(d.getTime()) ? iso : d.toLocaleString();
  }
</script>

<!-- ORG AI CONFIGURATION -->
<SettingsContainer title={m.settings_ai_title()}>
  <div class="flex items-start gap-3 text-sm text-fg-tertiary">
    <Bot class="mt-0.5 size-5 shrink-0 text-fg-tertiary" />
    <p>
      {#if hasStored}
        {m.settings_ai_using_override({ driver: stored?.driver ?? "" })}
      {:else}
        {m.settings_ai_using_default({
          driver: defaultDriver || m.settings_ai_deployment_default(),
        })}
      {/if}
    </p>
  </div>

  {#if $getQuery.isLoading}
    <div class="mt-4">
      <DelayedCircleOutlineSpinner isLoading={true} />
    </div>
  {:else}
    <div class="mt-4 flex flex-col gap-4">
      <div class="flex flex-col gap-1">
        <Label for="ai-driver">{m.settings_ai_driver()}</Label>
        <select
          id="ai-driver"
          class="rounded-sm border border-gray-200 bg-surface-background px-2 py-1.5 text-sm text-fg-primary dark:border-gray-700"
          bind:value={driver}
        >
          {#each DRIVERS as d (d.value)}
            <option value={d.value}>{d.label()}</option>
          {/each}
        </select>
      </div>

      <div class="flex flex-col gap-1">
        <Label for="ai-base-url">{m.settings_ai_base_url()}</Label>
        <Input id="ai-base-url" bind:value={baseUrl} />
      </div>

      <div class="flex flex-col gap-1">
        <Label for="ai-model">{m.settings_ai_model()}</Label>
        <Input id="ai-model" bind:value={model} />
      </div>

      <div class="flex flex-col gap-1">
        <Label for="ai-api-key">{m.settings_ai_api_key()}</Label>
        <input
          id="ai-api-key"
          type="password"
          bind:value={apiKey}
          class="rounded-sm border border-gray-200 bg-surface-background px-2 py-1.5 text-sm text-fg-primary dark:border-gray-700"
          placeholder={hasStored
            ? m.settings_ai_api_key_stored()
            : m.settings_ai_api_key_placeholder()}
        />
      </div>

      {#if stored?.updatedOn}
        <p class="text-xs text-fg-tertiary">
          {m.settings_ai_saved_on({ time: formatTime(stored.updatedOn) })}
        </p>
      {/if}

      <div class="flex flex-wrap items-center gap-2 pt-2">
        <Button type="primary" onClick={save} loading={$setMutation.isPending}>
          {m.settings_ai_save()}
        </Button>
        <Button type="secondary" onClick={test} loading={$testMutation.isPending}>
          {m.settings_ai_test()}
        </Button>
        {#if hasStored}
          <Button type="secondary" onClick={reset} loading={$deleteMutation.isPending}>
            {m.settings_ai_reset()}
          </Button>
        {/if}
      </div>

      {#if banner}
        <div
          class="rounded-sm border px-3 py-2 text-sm {banner.tone === 'ok'
            ? 'border-green-300 bg-green-50 text-green-800 dark:border-green-700 dark:bg-green-900/30 dark:text-green-200'
            : 'border-red-300 bg-red-50 text-red-800 dark:border-red-700 dark:bg-red-900/30 dark:text-red-200'}"
        >
          {banner.text}
        </div>
      {/if}

      {#if busy}
        <div>
          <DelayedCircleOutlineSpinner isLoading={true} />
        </div>
      {/if}
    </div>
  {/if}
</SettingsContainer>
