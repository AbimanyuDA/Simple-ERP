import { expect, test } from "@playwright/test";

const productID = "00000000-0000-0000-0000-000000000011";

test("dashboard loads reconciled metrics and searchable audit data", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill("admin@example.test");
  await page.getByLabel("Kata sandi").fill("Kuat-sekali-2026!");
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);
  await expect(page.getByText("Perlu persetujuan")).toBeVisible();
  await expect(page.getByText("Piutang berjalan")).toBeVisible();

  // Produce deterministic audit evidence instead of depending on another test's fixture order.
  await page.goto("/purchasing");
  await page.getByLabel("Product ID").fill(productID);
  await page.getByLabel("Jumlah").fill("1");
  await page.getByLabel("Estimasi harga").fill("10000");
  await page.getByRole("button", { name: "Buat PR" }).click();
  await expect(page.getByText("Purchase request draft berhasil dibuat.")).toBeVisible();

  await page.goto("/");
  await page.getByRole("button", { name: "Audit" }).click();
  await expect(page.getByText("PUR-PR-001").first()).toBeVisible();
});
