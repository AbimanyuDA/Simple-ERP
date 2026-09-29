import { expect, test } from "@playwright/test";

const productID = "00000000-0000-0000-0000-000000000011";
const adminEmail = process.env.E2E_ADMIN_EMAIL ?? "admin@example.test";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "Kuat-sekali-2026!";

test("core operations expose BPMN state and ledger-backed stock", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill(adminEmail);
  await page.getByLabel("Kata sandi").fill(adminPassword);
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);

  await page.goto("/purchasing");
  await page.getByLabel("Tujuan pembelian").fill("Browser acceptance fixture");
  await page.getByLabel("Product").selectOption(productID);
  await page.getByLabel("Jumlah").fill("2");
  await page.getByLabel("Estimasi harga").fill("10000");
  await page.getByRole("button", { name: "Buat PR" }).click();
  await expect(page.getByText("Purchase request draft berhasil dibuat.")).toBeVisible();
  await expect(page.getByText(/PR-/).first()).toBeVisible();
  await expect(page.getByText("DRAFT").first()).toBeVisible();

  await page.goto("/inventory/stock-movements");
  await expect(page.getByRole("heading", { name: "Stock Movements" })).toBeVisible();
  await expect(page.getByText("Belum ada movement.")).toBeVisible();
});
