import { expect, test } from "@playwright/test";

const adminEmail = process.env.E2E_ADMIN_EMAIL ?? "admin@example.test";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "Kuat-sekali-2026!";

test("navigation follows the selected module and remembers a collapsed sidebar", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill(adminEmail);
  await page.getByLabel("Kata sandi").fill(adminPassword);
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.locator('link[rel="icon"]')).toHaveAttribute("href", "/simple-erp-mark.png");

  await page.getByRole("link", { name: "Purchasing" }).click();
  await expect(page).toHaveURL(/\/purchasing\/purchase-requests$/);
  await expect(page.getByRole("link", { name: "Purchasing" })).toHaveAttribute("aria-current", "page");
  await expect(page.getByRole("link", { name: "Dashboard", exact: true })).not.toHaveAttribute("aria-current");

  const openSidebar = page.getByRole("button", { name: "Buka sidebar" });
  if (await openSidebar.isVisible()) await openSidebar.click();
  await page.getByRole("button", { name: "Ciutkan sidebar" }).click();
  await expect(page.getByRole("button", { name: "Buka sidebar" })).toBeVisible();
  await page.reload();
  await expect(page.getByRole("button", { name: "Buka sidebar" })).toBeVisible();
});
