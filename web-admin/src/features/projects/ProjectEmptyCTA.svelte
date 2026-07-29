<script lang="ts">
  import CtaContentContainer from "@rilldata/web-common/components/calls-to-action/CTAContentContainer.svelte";
  import CtaHeader from "@rilldata/web-common/components/calls-to-action/CTAHeader.svelte";
  import CtaLayoutContainer from "@rilldata/web-common/components/calls-to-action/CTALayoutContainer.svelte";
  import CtaMessage from "@rilldata/web-common/components/calls-to-action/CTAMessage.svelte";
  import MoonCircleOutline from "@rilldata/web-common/components/icons/MoonCircleOutline.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
</script>

<CtaLayoutContainer>
  <CtaContentContainer>
    <MoonCircleOutline
      size="104px"
      className="text-gray-300"
      gradientStopColor="slate-200"
    />
    <CtaHeader variant="bold">{m.project_empty_title()}</CtaHeader>
    <CtaMessage>
      {m.project_empty_message()}
    </CtaMessage>
  </CtaContentContainer>
</CtaLayoutContainer>
