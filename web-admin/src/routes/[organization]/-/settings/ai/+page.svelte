<script lang="ts">
  import { Bot } from "lucide-svelte";
  import SettingsContainer from "@rilldata/web-admin/features/organizations/settings/SettingsContainer.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
</script>

<!-- 智能配置（StarData）：只读占位页——LLM 配置目前在部署环境变量中，
     可视化编辑列为远期（设计稿 §6）。 -->
<SettingsContainer title={m.settings_ai_title()}>
  <div class="flex items-start gap-3 text-sm text-gray-600">
    <Bot class="mt-0.5 size-5 shrink-0 text-gray-400" />
    <p>{m.settings_ai_desc()}</p>
  </div>
</SettingsContainer>
