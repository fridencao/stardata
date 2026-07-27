<script lang="ts">
  import { page } from "$app/stores";
  import ErrorPage from "@rilldata/web-common/components/ErrorPage.svelte";

  $: statusCode = $page.status;
  $: header = ($page.error && $page.error.message) || "An error occurred";
  $: href = "/";
</script>

<ErrorPage {statusCode} {header} {href} />
