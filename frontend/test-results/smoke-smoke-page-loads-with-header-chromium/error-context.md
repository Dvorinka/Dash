# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: smoke.spec.ts >> smoke: page loads with header
- Location: e2e/smoke.spec.ts:3:5

# Error details

```
Error: expect(locator).toBeVisible() failed

Locator: locator('header')
Expected: visible
Timeout: 5000ms
Error: element(s) not found

Call log:
  - Expect "toBeVisible" with timeout 5000ms
  - waiting for locator('header')

```

# Page snapshot

```yaml
- generic [active] [ref=e1]:
  - alert [ref=e2]
  - generic [ref=e7] [cursor=pointer]:
    - button "Open Next.js Dev Tools" [ref=e8]:
      - img [ref=e9]
    - generic [ref=e12]:
      - button "Open issues overlay" [ref=e13]:
        - generic [ref=e14]:
          - generic [ref=e15]: "0"
          - generic [ref=e16]: "1"
        - generic [ref=e17]: Issue
      - button "Collapse issues badge" [ref=e18]:
        - img [ref=e19]
  - generic [ref=e21]:
    - img [ref=e22]
    - paragraph [ref=e24]: Failed to load dashboard
    - paragraph [ref=e25]: Failed to fetch
```

# Test source

```ts
  1  | import { test, expect } from "@playwright/test";
  2  | 
  3  | test("smoke: page loads with header", async ({ page }) => {
  4  |   await page.goto("http://localhost:3000");
> 5  |   await expect(page.locator("header")).toBeVisible();
     |                                        ^ Error: expect(locator).toBeVisible() failed
  6  |   await expect(page.getByText("Dash")).toBeVisible();
  7  | });
  8  | 
  9  | test("smoke: theme toggle works", async ({ page }) => {
  10 |   await page.goto("http://localhost:3000");
  11 |   const toggle = page.getByLabel("Toggle theme");
  12 |   await toggle.click();
  13 |   await page.getByText("CasaOS").click();
  14 |   const theme = await page.evaluate(() => document.documentElement.getAttribute("data-theme"));
  15 |   expect(theme).toBe("casaos");
  16 | });
  17 | 
  18 | test("smoke: empty state shows add button", async ({ page }) => {
  19 |   await page.goto("http://localhost:3000");
  20 |   // If no services exist, the empty state should be visible
  21 |   const emptyState = page.getByText("No apps yet");
  22 |   if (await emptyState.isVisible()) {
  23 |     await expect(page.getByRole("button", { name: /add app/i })).toBeVisible();
  24 |   }
  25 | });
  26 | 
```