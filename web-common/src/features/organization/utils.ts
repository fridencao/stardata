import { page } from "$app/stores";
import {
  createLocalServiceGetMetadata,
  createLocalServiceListOrganizationsAndBillingMetadataRequest,
} from "@rilldata/web-common/runtime-client/local-service";
import { derived } from "svelte/store";

export function getPlanUpgradeUrl(orgName: string) {
  const metadataQuery = createLocalServiceGetMetadata();
  const orgsMetadataQuery =
    createLocalServiceListOrganizationsAndBillingMetadataRequest();

  return derived(
    [metadataQuery, orgsMetadataQuery, page],
    ([metadata, , pageState]) => {
      const adminUrl = metadata.data?.adminUrl;
      if (!adminUrl) return "";

      // Private deployment has no billing tiers, so always point to the org settings page.
      let cloudUrl = adminUrl.replace("admin.rilldata", "ui.rilldata");
      // hack for dev env
      if (cloudUrl === "http://localhost:8080") {
        cloudUrl = "http://localhost:3000";
      }

      const url = new URL(cloudUrl);
      url.pathname = `/${orgName}/-/settings`;
      url.searchParams.set("upgrade", "true");
      const redirectUrl = new URL(pageState.url);
      // set the org to avoid showing the org selector again
      redirectUrl.searchParams.set("org", orgName);
      url.searchParams.set("redirect", redirectUrl.toString());
      return url.toString();
    },
  );
}

export function getIsOrgOnTrial(orgName: string) {
  return derived(
    createLocalServiceListOrganizationsAndBillingMetadataRequest(),
    (orgsMetadata) => {
      const metadataForOrg = orgsMetadata?.data?.orgs.find(
        (o) => o.name === orgName,
      );
      return !!orgName && !!metadataForOrg?.issues && false;
    },
  );
}
