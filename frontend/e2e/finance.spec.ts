import { expect, test } from "@playwright/test";

const adminEmail = process.env.E2E_ADMIN_EMAIL ?? "admin@example.test";
const adminPassword = process.env.E2E_ADMIN_PASSWORD ?? "Kuat-sekali-2026!";
const operatorEmail = process.env.E2E_OPERATOR_EMAIL ?? "operator@example.test";
const operatorPassword = process.env.E2E_OPERATOR_PASSWORD ?? "Kuat-sekali-2026!";

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

test("operator cannot read or mutate finance records", async ({ page }) => {
  await page.goto("/login");
  await page.getByLabel("Email").fill(operatorEmail);
  await page.getByLabel("Kata sandi").fill(operatorPassword);
  await page.getByRole("button", { name: "Masuk" }).click();
  await expect(page).toHaveURL(/\/$/);
  await page.goto("/finance");
  await expect(page.getByText("Akses ditolak.")).toBeVisible();
});
