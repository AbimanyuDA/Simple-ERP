import { expect, test } from "@playwright/test";

const adminEmail = process.env.E2E_ADMIN_EMAIL ?? "admin@example.test";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "Kuat-sekali-2026!";
const staffEmail = process.env.E2E_STAFF_EMAIL ?? "staff@example.test";
const staffPassword = process.env.E2E_STAFF_PASSWORD ?? "Kuat-sekali-2026!";

test("finance workspace exposes AP AR and immutable correction guidance", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill(adminEmail);
  await page.getByLabel("Kata sandi").fill(adminPassword);
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);
  await page.goto("/finance");
  await expect(page.getByRole("button", { name: "Payables · Supplier" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Receivables · Customer" })).toBeVisible();
  await expect(page.getByText(/Posted records tidak diedit/)).toBeVisible();
});

test("staff can access payables and receivables draft workflows", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill(staffEmail);
  await page.getByLabel("Kata sandi").fill(staffPassword);
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);
  await page.goto("/finance");
  await expect(page.getByRole("button", { name: "Payables · Supplier" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Receivables · Customer" })).toBeVisible();
  await page.getByRole("button", { name: "Receivables · Customer" }).click();
  await expect(page.getByText("Buat invoice penjualan")).toBeVisible();
});
