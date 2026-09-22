import { expect, test } from "@playwright/test";

test("finance workspace exposes AP AR and immutable correction guidance", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill("admin@example.test");
  await page.getByLabel("Kata sandi").fill("Kuat-sekali-2026!");
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);
  await page.goto("/finance");
  await expect(page.getByRole("button", { name: "Payables · Supplier" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Receivables · Customer" })).toBeVisible();
  await expect(page.getByText(/Posted records tidak diedit/)).toBeVisible();
});

test("operator cannot read or mutate finance records", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill("operator@example.test");
  await page.getByLabel("Kata sandi").fill("Kuat-sekali-2026!");
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);
  await page.goto("/finance");
  await expect(page.getByText("Akses ditolak.")).toBeVisible();
});
