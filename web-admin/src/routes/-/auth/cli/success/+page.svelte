<script>
  import StarDataLogoWordmark from "@rilldata/web-common/components/icons/StarDataLogoWordmark.svelte";
  import CtaContentContainer from "@rilldata/web-common/components/calls-to-action/CTAContentContainer.svelte";
  import CtaHeader from "@rilldata/web-common/components/calls-to-action/CTAHeader.svelte";
  import CtaLayoutContainer from "@rilldata/web-common/components/calls-to-action/CTALayoutContainer.svelte";
  import CtaMessage from "@rilldata/web-common/components/calls-to-action/CTAMessage.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
</script>

<svelte:head>
  <title>{m.auth_cli_success_title()}</title>
</svelte:head>

<CtaLayoutContainer>
  <CtaContentContainer>
    <StarDataLogoWordmark size="lg" />
    <CtaHeader>{m.auth_cli_success_title()}</CtaHeader>
    <CtaMessage>{m.auth_cli_success_message()}</CtaMessage>
  </CtaContentContainer>
</CtaLayoutContainer>
