import { expect, test } from "@playwright/test";

test("opening on a phone keeps the list width chosen on the desktop", async ({ page }) => {
  await page.goto("/");
  await page.evaluate(() => localStorage.setItem("mailpeek:list-width", "400"));

  await page.setViewportSize({ width: 390, height: 800 });
  await page.reload();
  await page.waitForTimeout(400); // longer than the width's save delay

  await page.setViewportSize({ width: 1400, height: 800 });
  await page.reload();
  await expect.poll(async () => (await page.locator("#email-list").boundingBox())?.width).toBe(400);
  expect(await page.evaluate(() => localStorage.getItem("mailpeek:list-width"))).toBe("400");
});

test("the theme button switches between light and dark and is remembered", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "light" });
  await page.goto("/");
  const html = page.locator("html");
  const toggle = page.getByRole("button", { name: "Dark theme" });
  // Follows the system until chosen.
  await expect(html).not.toHaveClass(/\bdark\b/);
  await expect(toggle).toHaveAttribute("aria-pressed", "false");

  await toggle.click();
  await expect(html).toHaveClass(/\bdark\b/);
  await expect(toggle).toHaveAttribute("aria-pressed", "true");
  await page.reload();
  await expect(html).toHaveClass(/\bdark\b/);

  await toggle.click();
  await expect(html).not.toHaveClass(/\bdark\b/);
});

test("the phone header keeps the logo and has no list toggle", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 800 });
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Email list" })).toBeHidden();
  const logo = await page.locator('header img[src="/favicon.svg"]').boundingBox();
  expect(logo?.width).toBe(32);
});
