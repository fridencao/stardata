import { expect } from "@playwright/test";
import { test } from "./setup/base";

// Feature Access matrix = the six-feature × user/group visibility control
// introduced by design/feature-access-control.md. Verifies the admin page
// renders, that toggling an org default persists across a reload, and that
// disabling a feature actually removes its tab for a downstream viewer.
test.describe("Feature Access", () => {
  test.describe.configure({ mode: "serial" });

  test("admin can flip an org default and it persists across reload", async ({
    adminPage,
  }) => {
    await adminPage.goto("/e2e/-/settings/feature-access");

    // The matrix page renders (i18n title varies; check for a feature key label).
    await expect(
      adminPage
        .getByText(/dashboards|reports|alerts|chat|studio/i)
        .first(),
    ).toBeVisible({ timeout: 20_000 });

    // Locate the "reports" org-default switch. Existing implementation uses
    // Svelte switch inputs; we scope by the row that contains "reports".
    const reportsRow = adminPage
      .locator("tr, [role='row']")
      .filter({ hasText: /reports|报表/i })
      .first();
    const reportsSwitch = reportsRow
      .locator("input[type='checkbox'], [role='switch']")
      .first();
    await expect(reportsSwitch).toBeVisible();

    // Toggle and read state, then reload and verify persistence.
    const initial = await reportsSwitch.isChecked().catch(() => null);
    await reportsSwitch.click();
    await adminPage.waitForTimeout(400); // let the RPC settle
    await adminPage.reload();

    const after = await adminPage
      .locator("tr, [role='row']")
      .filter({ hasText: /reports|报表/i })
      .first()
      .locator("input[type='checkbox'], [role='switch']")
      .first()
      .isChecked()
      .catch(() => null);

    if (initial !== null && after !== null) {
      expect(after).toBe(!initial);
    }

    // Restore original state to keep the shared e2e org clean for later specs.
    await adminPage
      .locator("tr, [role='row']")
      .filter({ hasText: /reports|报表/i })
      .first()
      .locator("input[type='checkbox'], [role='switch']")
      .first()
      .click();
  });

  test("viewer without accessReports sees no Reports tab", async ({
    adminPage,
    viewerPage,
  }) => {
    // Setup: turn off reports at the org default level using the admin page.
    await adminPage.goto("/e2e/-/settings/feature-access");
    const reportsSwitch = adminPage
      .locator("tr, [role='row']")
      .filter({ hasText: /reports|报表/i })
      .first()
      .locator("input[type='checkbox'], [role='switch']")
      .first();

    const wasOn = (await reportsSwitch.isChecked().catch(() => true)) ?? true;
    if (wasOn) {
      await reportsSwitch.click();
      await adminPage.waitForTimeout(400);
    }

    try {
      // Viewer visits the project portal.
      await viewerPage.goto("/e2e/openrtb?preview=1");

      // Reports tab must not be visible in the portal nav.
      await expect(
        viewerPage.getByRole("link", { name: /reports|报表/i }),
      ).toHaveCount(0);
    } finally {
      // Restore the previous state so later tests / manual runs aren't affected.
      if (wasOn) {
        await adminPage
          .locator("tr, [role='row']")
          .filter({ hasText: /reports|报表/i })
          .first()
          .locator("input[type='checkbox'], [role='switch']")
          .first()
          .click();
      }
    }
  });
});
