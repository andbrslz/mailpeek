import { expect, test } from "@playwright/test";
import { authHttpPort } from "./ports";

test.describe("Brazilian Portuguese browser", () => {
  test.use({ locale: "pt-BR" });
  test("is detected", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("searchbox", { name: "Buscar e-mails" })).toBeVisible();
    await expect(page.locator("html")).toHaveAttribute("lang", "pt-BR");
    await expect(page.getByRole("button", { name: "Lista de e-mails" })).toBeVisible();
  });
});

test.describe("European Portuguese browser", () => {
  test.use({ locale: "pt-PT" });
  test("gets pt-PT wording", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("searchbox", { name: "Pesquisar e-mails" })).toBeVisible();
  });
});

test("the chosen language is saved and survives a reload", async ({ page, context }) => {
  await page.goto("/");
  const picker = page.getByRole("button", { name: "Language" });
  await picker.click();
  const list = page.getByRole("listbox", { name: "Language" });
  await expect(list.getByRole("option")).toHaveCount(7);
  await expect(list.getByRole("option", { selected: true })).toHaveText(/^Automatic/);
  await list.getByRole("option", { name: "Français (France)" }).click();
  await expect(list).toBeHidden();
  await expect(page.getByRole("searchbox", { name: "Rechercher des e-mails" })).toBeVisible();
  await expect(page.locator("html")).toHaveAttribute("lang", "fr-FR");

  expect(await page.evaluate(() => localStorage.getItem("mailpeek:locale"))).toBe("fr-FR");
  expect((await context.cookies()).find((c) => c.name === "mailpeek_lang")?.value).toBe("fr-FR");

  await page.reload();
  await expect(page.getByRole("searchbox", { name: "Rechercher des e-mails" })).toBeVisible();

  // Back to automatic: follows the browser (en-US here) and forgets the choice.
  // Keyboard works too: open with ↓, go to the top, pick with Enter.
  await page.getByRole("button", { name: "Langue" }).focus();
  await page.keyboard.press("ArrowDown");
  await expect(page.getByRole("option", { name: "Français (France)" })).toBeFocused();
  await page.keyboard.press("Home");
  await page.keyboard.press("Enter");
  await expect(page.getByRole("searchbox", { name: "Search emails" })).toBeVisible();
  expect(await page.evaluate(() => localStorage.getItem("mailpeek:locale"))).toBeNull();
});

test.describe("sign-in screen", () => {
  const login = `http://127.0.0.1:${authHttpPort}/login`;

  test.describe("in the browser's language", () => {
    test.use({ locale: "es-ES" });
    test("es-ES", async ({ page }) => {
      await page.goto(login);
      await expect(page.getByRole("heading", { name: "Iniciar sesión" })).toBeVisible();
      await expect(page.getByLabel("Contraseña")).toBeVisible();
    });
  });

  test("follows the language chosen in the UI", async ({ page, context }) => {
    await context.addCookies([
      { name: "mailpeek_lang", value: "pt-PT", url: `http://127.0.0.1:${authHttpPort}` },
    ]);
    await page.goto(login);
    await expect(page.getByRole("heading", { name: "Iniciar sessão" })).toBeVisible();
    await expect(page.getByLabel("Palavra-passe")).toBeVisible();
  });
});
