<script lang="ts">
  import ContentContainer from "@rilldata/web-common/components/layout/ContentContainer.svelte";
  import DashboardList from "../../../features/dashboards/DashboardList.svelte";
</script>

<svelte:head>
  <title>StarData · 看板</title>
</svelte:head>

<div class="h-full overflow-y-auto">
  <ContentContainer title="看板" maxWidth={1100}>
    <DashboardList showSearch seeAllHref="/boards" />
  </ContentContainer>
</div>
