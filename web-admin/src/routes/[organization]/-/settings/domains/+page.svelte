<script lang="ts">
  import { page } from "$app/stores";
  import OrgDomainAllowListSettings from "@rilldata/web-admin/features/organizations/settings/OrgDomainAllowListSettings.svelte";

  let organization = $derived($page.params.organization);
</script>

<!-- DOMAIN ALLOWLIST — email domains that may auto-join this org.
     Deliberately separate from the StarData business concept of 数据域 (data domain),
     which maps onto a project; see design/phase4-enterprise-app.md §8. -->
<OrgDomainAllowListSettings {organization} />
