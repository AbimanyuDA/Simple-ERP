import { expect, test } from "@playwright/test";

const productID = "00000000-0000-0000-0000-000000000011";

test("core operations expose BPMN state and ledger-backed stock", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill("admin@example.test");
  await page.getByLabel("Kata sandi").fill("Kuat-sekali-2026!");
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);

  await page.goto("/purchasing");
  await page.getByLabel("Product ID").fill(productID);
  await page.getByLabel("Jumlah").fill("2");
  await page.getByLabel("Estimasi harga").fill("10000");
  await page.getByRole("button", { name: "Buat PR" }).click();
  await expect(page.getByText("Purchase request draft berhasil dibuat.")).toBeVisible();
  await expect(page.getByText(/PR-/).first()).toBeVisible();
  await expect(page.getByText("DRAFT").first()).toBeVisible();

  await page.goto("/inventory");
  await expect(page.getByRole("button", { name: "Movement ledger" })).toBeVisible();
});
