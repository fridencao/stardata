import { expect } from "@playwright/test";
import { test } from "./setup/base";

// Portal home = the business-user landing page introduced in Phase 4 (see
// design/phase4-enterprise-app.md §4.1). Verifies that a logged-in user lands on
// the greeting + recommended-questions view and that the recommendations link
// into the chat page with the question prefilled via `?q=`.
test.describe("Portal Home", () => {
  test("business user lands on the portal home with recommended questions", async ({
    adminPage,
  }) => {
    // Governors are redirected to Studio by [project]/+page.ts unless ?preview is set,
    // so pass ?preview=1 to see exactly what a business user would see.
    await adminPage.goto("/e2e/openrtb?preview=1");

    // Greeting / search input is the portal home hero (StarData branding).
    await expect(
      adminPage.getByRole("textbox").first(),
    ).toBeVisible();

    // At least one AI-recommended question is generated from published metrics
    // views. We assert the container renders; the exact text is model-driven.
    const chatBase = "/e2e/openrtb/chat";
    const firstRecommendation = adminPage
      .locator(`a[href^="${chatBase}?new=true&q="]`)
      .first();
    await expect(firstRecommendation).toBeVisible({ timeout: 15_000 });

    // Clicking the recommendation navigates to /chat with the question prefilled.
    const href = await firstRecommendation.getAttribute("href");
    expect(href, "recommendation href").toMatch(
      /^\/e2e\/openrtb\/chat\?new=true&q=/,
    );
    await firstRecommendation.click();
    await adminPage.waitForURL(/\/chat\?new=true&q=/, { timeout: 15_000 });
  });

  test("Preview button opts governors out of the Studio redirect", async ({
    adminPage,
  }) => {
    // Direct load without ?preview would redirect governors to Studio.
    // With ?preview=1 the portal home is served instead — this is the "Preview
    // business view" button's contract (see StudioTabs.svelte).
    await adminPage.goto("/e2e/openrtb?preview=1");
    await expect(adminPage).toHaveURL(/\/e2e\/openrtb\?preview=1$/);

    // Without the preview param, the redirect kicks in.
    await adminPage.goto("/e2e/openrtb");
    await adminPage.waitForURL(/\/-\/edit\/studio(?:\/|$)/, {
      timeout: 15_000,
    });
  });
});
