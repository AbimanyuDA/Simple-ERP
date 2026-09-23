import { expect, test } from "@playwright/test";

test("CAP-AUTH-001 operator signs in through secure session cookies", async ({ page }) => {
  await page.goto("/");
  await expect(page).toHaveURL(/\/login$/);
  await page.getByLabel("Email").fill("admin@example.test");
  await page.getByLabel("Kata sandi").fill("Kuat-sekali-2026!");
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByRole("heading", { name: "Kendali usaha, tanpa kebisingan." })).toBeVisible();
  const cookies = await page.context().cookies();
  expect(cookies.find((cookie) => cookie.name === "__Host-erp_access")?.httpOnly).toBe(true);
});
