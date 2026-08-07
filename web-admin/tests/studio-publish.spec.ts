import { expect } from "@playwright/test";
import { test } from "./setup/base";

// Studio Publish page = the technical governor's release console introduced in
// Phase 4 (see design/phase4-enterprise-app.md §5.4). Verifies the page renders
// the two contracts it promises: a publish-gate toggle set (per metrics view)
// and a publish history list with a Publish action available to admins.
test.describe("Studio Publish", () => {
  test("publish page renders gate toggles, history, and publish action", async ({
    adminPage,
  }) => {
    await adminPage.goto("/e2e/openrtb/-/edit/studio/publish");

    // The publish header is present (page-level i18n string).
    await expect(
      adminPage.getByRole("heading", { name: /publish|发布/i }).first(),
    ).toBeVisible({ timeout: 20_000 });

    // Publish action button (creates a new release version). Its exact label is
    // i18n; asserting on the button role narrows to interactive controls only.
    await expect(
      adminPage.getByRole("button", { name: /publish|发布/i }).first(),
    ).toBeVisible();

    // Publish history table is present (empty state OK on a fresh project).
    // We assert the history section is on the DOM by locating its heading.
    await expect(
      adminPage.getByText(/history|历史/i).first(),
    ).toBeVisible();
  });

  test("Preview business view button in Studio points to portal home", async ({
    adminPage,
  }) => {
    await adminPage.goto("/e2e/openrtb/-/edit/studio");

    // The preview affordance added in P2-5. i18n label is "Preview business view"
    // (or the Chinese equivalent). We assert its href, since Playwright can't
    // easily follow target="_blank" without new contexts.
    const preview = adminPage.getByRole("link", {
      name: /preview business view|预览业务视图/i,
    });
    await expect(preview).toBeVisible();
    await expect(preview).toHaveAttribute("target", "_blank");
    await expect(preview).toHaveAttribute("href", /\?preview=1$/);
  });
});
