import { test, expect } from "@playwright/test";

test("smoke: page loads with header", async ({ page }) => {
  await page.goto("http://localhost:3000");
  await expect(page.locator("header")).toBeVisible();
  await expect(page.getByText("Dash")).toBeVisible();
});

test("smoke: theme toggle works", async ({ page }) => {
  await page.goto("http://localhost:3000");
  const toggle = page.getByLabel("Toggle theme");
  await toggle.click();
  await page.getByText("Light").click();
  const theme = await page.evaluate(() => document.documentElement.getAttribute("data-theme"));
  expect(theme).toBe("light");
});

test("smoke: empty state shows add button", async ({ page }) => {
  await page.goto("http://localhost:3000");
  // If no services exist, the empty state should be visible
  const emptyState = page.getByText("No apps yet");
  if (await emptyState.isVisible()) {
    await expect(page.getByRole("button", { name: /add app/i })).toBeVisible();
  }
});
