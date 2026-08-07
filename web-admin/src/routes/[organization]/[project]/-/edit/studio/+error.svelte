<script lang="ts">
  import { page } from "$app/stores";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";

  // The edit session emits errors when the dev deployment is unavailable or
  // still initializing. Because Studio is nested inside /-/edit, any edit-session
  // failure bubbles up as an unhandled error. This boundary translates the
  // raw error into user-friendly language for technical governors.
  $: error = $page.error;
  $: basePath = `/${$page.params.organization}/${$page.params.project}`;
  $: message = deriveMessage(error);

  function deriveMessage(err: App.Error | null): string {
    if (!err) return "";
    const raw = err.message ?? "";
    if (/runtime.*not.*reachable|runtime.*not.*ready/i.test(raw)) {
      return m.studio_error_env_starting();
    }
    if (/failed|error|unavailable/i.test(raw)) {
      return m.studio_error_env_failed();
    }
    return `${m.studio_error_generic()} ${raw}`;
  }
</script>

<div class="flex h-full flex-col items-center justify-center gap-4 px-8 text-center">
  <div class="text-4xl font-bold text-gray-300">⚠</div>
  <p class="max-w-md text-sm text-gray-600">{message}</p>
  <a
    href={basePath + "/-/edit/studio"}
    class="mt-2 rounded-lg bg-primary-600 px-4 py-2 text-sm font-semibold text-white no-underline hover:bg-primary-700"
  >
    {m.studio_error_retry()}
  </a>
</div>
