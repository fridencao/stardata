<script lang="ts">
  import StarDataLogoWordmark from "@rilldata/web-common/components/icons/StarDataLogoWordmark.svelte";
  import SignIn from "@rilldata/web-admin/features/authentication/SignIn.svelte";
</script>

<div class="flex flex-col items-center gap-4 mx-auto w-fit">
  <StarDataLogoWordmark size="lg" />
  <div class="auth-title text-center">
    Sign in to StarData
  </div>

  <div class="flex flex-col gap-6 text-left auth-card">
    <div>
      <div class="auth-card__title">Welcome back</div>
      <div class="auth-card__subtitle">
        Sign in with your StarData account to continue.
      </div>
    </div>
    <SignIn />
  </div>
</div>
