import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

const isPhone = (name: string) => name.startsWith("phone");

async function openHome(page: Page, theme: "light" | "dark", query = "") {
  // VitePress reads this before it paints, so the first frame is already in the right theme.
  await page.addInitScript((t) => localStorage.setItem("vitepress-theme-appearance", t), theme);
  await page.goto(query);
  await expect(page.locator("html")).toHaveClass(theme === "dark" ? /dark/ : /^((?!dark).)*$/);
  await expect(page.locator(".hk-scene canvas")).toBeVisible();
}

// A pixel the scene paints, read straight from the canvas.
async function canvasPixel(page: Page, x: number, y: number) {
  return page.evaluate(
    ([px, py]) => {
      const c = document.querySelector<HTMLCanvasElement>(".hk-scene canvas")!;
      const d = c.getContext("2d")!.getImageData(px, py, 1, 1).data;
      return [d[0], d[1], d[2]];
    },
    [x, y],
  );
}

for (const theme of ["dark", "light"] as const) {
  test.describe(`home page, ${theme}`, () => {
    test("fits one screen on a laptop", async ({ page }, info) => {
      test.skip(isPhone(info.project.name), "phones scroll by design");
      for (const size of [{ width: 1366, height: 650 }, { width: 1280, height: 720 }, { width: 1440, height: 800 }]) {
        await page.setViewportSize(size);
        await openHome(page, theme);
        const overflow = await page.evaluate(() => document.documentElement.scrollHeight - window.innerHeight);
        expect(overflow, `page is ${overflow}px taller than a ${size.width}x${size.height} window`).toBeLessThanOrEqual(1);
      }
    });

    test("does not scroll sideways", async ({ page }) => {
      await openHome(page, theme);
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth);
      expect(overflow).toBeLessThanOrEqual(1);
    });

    test("the scene paints its own palette", async ({ page }) => {
      await openHome(page, theme, "?t=13");
      // Top-left is sky: near black at night, pale in the day.
      const [r, g, b] = await canvasPixel(page, 2, 2);
      const brightness = (r + g + b) / 3;
      if (theme === "dark") expect(brightness).toBeLessThan(60);
      else expect(brightness).toBeGreaterThan(180);
    });

    test("has no serious accessibility violations", async ({ page }, info) => {
      test.skip(info.project.name !== "chromium", "axe runs once, in Chromium");
      // The hero fades in; measuring contrast mid-fade reports false violations.
      await page.emulateMedia({ reducedMotion: "reduce" });
      await openHome(page, theme, "?t=13");
      const results = await new AxeBuilder({ page }).include(".hk-hero-wrap").analyze();
      const serious = results.violations.filter((v) => v.impact === "serious" || v.impact === "critical");
      expect(serious.map((v) => `${v.id}: ${v.nodes.map((n) => n.target.join(" ")).join(", ")}`)).toEqual([]);
    });
  });
}

test.describe("scene controls", () => {
  test("each stage button jumps the scene and lights up", async ({ page }) => {
    await openHome(page, "dark");
    const stages = page.locator(".hk-stages button");
    await expect(stages).toHaveCount(4);

    await stages.nth(3).click(); // hikma sync
    await expect(stages.nth(3)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-scene-cli")).toContainText("hikma sync", { timeout: 10_000 });

    await stages.nth(1).click(); // skill install
    await expect(stages.nth(1)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-scene-cli")).toContainText("skill install", { timeout: 10_000 });

    await stages.nth(0).click(); // back to the start
    await expect(stages.nth(0)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-scene-title")).toBeVisible();
  });

  test("a later stage dims until it is reached", async ({ page }) => {
    await openHome(page, "dark");
    const last = page.locator(".hk-stages button").nth(3);
    await expect(last).not.toHaveClass(/on|done/);
    const opacity = await last.evaluate((el) => Number(getComputedStyle(el).opacity));
    expect(opacity).toBeLessThan(0.9);
  });

  test("pause stops the clock and play resumes it", async ({ page }) => {
    await openHome(page, "dark");
    const toggle = page.locator(".hk-scene-toggle");
    await expect(toggle).toHaveText("Pause");
    await toggle.click();
    await expect(toggle).toHaveText("Play");
    const cliText = () => page.evaluate(() => document.querySelector(".hk-scene-cli")?.textContent ?? "");
    const before = await cliText();
    await page.waitForTimeout(1500);
    const after = await cliText();
    expect(after).toBe(before);
    await toggle.click();
    await expect(toggle).toHaveText("Pause");
  });

  test("reduced motion shows a still frame with no pause button", async ({ browser }) => {
    const ctx = await browser.newContext({ reducedMotion: "reduce" });
    const page = await ctx.newPage();
    await page.goto("");
    await expect(page.locator(".hk-scene canvas")).toBeVisible();
    await expect(page.locator(".hk-scene-toggle")).toHaveCount(0);
    await ctx.close();
  });

  test("the theme switch repaints the scene", async ({ page }, info) => {
    test.skip(isPhone(info.project.name), "the nav switch is in a menu on phones");
    await openHome(page, "dark", "?t=13");
    const dark = await canvasPixel(page, 2, 2);
    await page.locator(".VPSwitchAppearance").first().click();
    await expect(page.locator("html")).not.toHaveClass(/dark/);
    await expect.poll(async () => (await canvasPixel(page, 2, 2)).join(",")).not.toBe(dark.join(","));
  });
});

test.describe("install line", () => {
  test("the command is readable and Copy confirms", async ({ page, context }, info) => {
    test.skip(info.project.name !== "chromium", "clipboard permission is Chromium-only");
    await context.grantPermissions(["clipboard-read", "clipboard-write"]);
    await openHome(page, "light");
    const code = page.locator(".hk-install-line code");
    await expect(code).toContainText("brew install HasanKhatib/tap/hikma");
    // The command once rendered dark green on a dark box in light mode.
    const [fg, bg] = await code.evaluate((el) => [getComputedStyle(el).color, getComputedStyle(el.parentElement!).backgroundColor]);
    const lum = (rgb: string) => {
      const [r, g, b] = rgb.match(/\d+(\.\d+)?/g)!.slice(0, 3).map(Number).map((v) => {
        const s = v / 255;
        return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
      });
      return 0.2126 * r + 0.7152 * g + 0.0722 * b;
    };
    const ratio = (Math.max(lum(fg), lum(bg)) + 0.05) / (Math.min(lum(fg), lum(bg)) + 0.05);
    expect(ratio, `command ${fg} on ${bg}`).toBeGreaterThan(4.5);

    await page.getByRole("button", { name: "Copy" }).click();
    await expect(page.getByRole("button", { name: /Copied/ })).toBeVisible();
  });
});

test("the social card is wired", async ({ page }) => {
  await page.goto("");
  await expect(page.locator('meta[property="og:image"]')).toHaveAttribute("content", /og\.png$/);
  const res = await page.request.get("og.png");
  expect(res.ok()).toBeTruthy();
  expect(res.headers()["content-type"]).toContain("image/png");
});
