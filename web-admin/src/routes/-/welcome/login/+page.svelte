<script lang="ts">
  import StarDataLogoWordmark from "@rilldata/web-common/components/icons/StarDataLogoWordmark.svelte";
  import SignIn from "@rilldata/web-admin/features/authentication/SignIn.svelte";
  import { m } from "@rilldata/web-common/lib/i18n/gen/messages";
</script>

<div class="flex flex-col items-center gap-4 mx-auto w-fit">
  <StarDataLogoWordmark size="lg" />
  <div class="auth-title text-center">
    {m.auth_signin_title()}
  </div>

  <div class="flex flex-col gap-6 text-left auth-card">
    <div>
      <div class="auth-card__title">{m.auth_welcome_back()}</div>
      <div class="auth-card__subtitle">
        {m.auth_signin_subtitle()}
      </div>
    </div>
    <SignIn />
  </div>
</div>
