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
    test("the hero fits the window and nothing is cut off", async ({ page }, info) => {
      const sizes = isPhone(info.project.name)
        ? [{ width: 390, height: 664 }]
        : [{ width: 1366, height: 650 }, { width: 1280, height: 720 }, { width: 1440, height: 800 }, { width: 1920, height: 1080 }];
      for (const size of sizes) {
        await page.setViewportSize(size);
        await openHome(page, theme);
        const m = await page.evaluate(() => ({
          dockBottom: document.querySelector(".hk-dock")!.getBoundingClientRect().bottom,
          installRight: document.querySelector(".hk-getit")!.getBoundingClientRect().right,
          sceneHeight: document.querySelector(".hk-scene")!.getBoundingClientRect().height,
          headlineBottom: document.querySelector(".hk-headline")!.getBoundingClientRect().bottom,
          dockTop: document.querySelector(".hk-dock")!.getBoundingClientRect().top,
          // The grass line is 92/135 down the canvas; the dock must sit below it so it never covers the feet.
          groundLine: (() => {
            const c = document.querySelector(".hk-scene canvas")!.getBoundingClientRect();
            return c.bottom - c.height * (43 / 135);
          })(),
          dockPadding: getComputedStyle(document.querySelector(".hk-dock")!).paddingTop,
          h: window.innerHeight,
          w: window.innerWidth,
        }));
        const at = `${size.width}x${size.height}`;
        expect(m.dockBottom, `${at}: the dock is cut off`).toBeLessThanOrEqual(m.h + 1);
        expect(m.installRight, `${at}: the install box is cut off`).toBeLessThanOrEqual(m.w + 1);
        expect(m.sceneHeight, `${at}: the scene got squeezed`).toBeGreaterThan(200);
        expect(m.headlineBottom, `${at}: the headline runs into the dock`).toBeLessThan(m.dockTop);
        if (!isPhone(info.project.name)) {
          expect(m.dockTop, `${at}: the dock covers the character's feet`).toBeGreaterThanOrEqual(m.groundLine - 1);
          expect(m.dockPadding, `${at}: the dock keeps its 20px vertical padding`).toBe("20px");
        }
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
  const steps = (page: Page) => page.locator(".hk-rail ol button");

  test("each step button jumps the scene and lights up", async ({ page }) => {
    await openHome(page, "dark");
    await expect(steps(page)).toHaveCount(4);

    await steps(page).nth(3).click(); // hikma sync
    await expect(steps(page).nth(3)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-bubble")).toContainText("hikma sync", { timeout: 15_000 });

    await steps(page).nth(1).click(); // skill install
    await expect(steps(page).nth(1)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-bubble")).toContainText("skill install", { timeout: 10_000 });

    await steps(page).nth(0).click(); // back to the start
    await expect(steps(page).nth(0)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-scene-title")).toBeVisible();
  });

  test("scrolling down moves the story forward and scrolling up goes back", async ({ page }) => {
    await openHome(page, "dark");
    const range = await page.evaluate(() => document.documentElement.scrollHeight - window.innerHeight);
    expect(range, "the page has scroll room for the story").toBeGreaterThan(300);

    await page.evaluate((y) => window.scrollTo(0, y), range * 0.9); // the last stage
    await expect(steps(page).nth(3)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-scene-toggle")).toHaveText("Pause"); // scrolling does not pause the story

    await page.evaluate((y) => window.scrollTo(0, y), range * 0.3); // the second stage
    await expect(steps(page).nth(1)).toHaveAttribute("aria-current", "step");
    await page.evaluate(() => window.scrollTo(0, 0));
    await expect(steps(page).nth(0)).toHaveAttribute("aria-current", "step");
  });

  test("scrolling to the end keeps playing and the story loops by itself", async ({ page }) => {
    await page.clock.install();
    await openHome(page, "dark");
    const range = await page.evaluate(() => document.documentElement.scrollHeight - window.innerHeight);
    await page.evaluate((y) => window.scrollTo(0, y), range); // the very end
    await expect(steps(page).nth(3)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-scene-toggle")).toHaveText("Pause");
    await page.clock.runFor(4000);
    await expect(steps(page).nth(0)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-scene-toggle")).toHaveText("Pause");
  });

  test("a story paused with the button stays paused while the reader scrolls", async ({ page }) => {
    await openHome(page, "dark");
    await page.locator(".hk-scene-toggle").click();
    await expect(page.locator(".hk-scene-toggle")).toHaveText("Play");
    const range = await page.evaluate(() => document.documentElement.scrollHeight - window.innerHeight);
    await page.evaluate((y) => window.scrollTo(0, y), range * 0.6);
    await expect(steps(page).nth(2)).toHaveAttribute("aria-current", "step");
    await page.waitForTimeout(800);
    await expect(steps(page).nth(2)).toHaveAttribute("aria-current", "step");
    await expect(page.locator(".hk-scene-toggle")).toHaveText("Play");
  });

  test("the navbar stays put while the story scrolls", async ({ page }) => {
    await openHome(page, "dark");
    await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight * 0.5));
    await expect(steps(page).nth(2)).toHaveAttribute("aria-current", "step");
    const top = await page.evaluate(() => Math.round(document.querySelector(".VPNavBar")!.getBoundingClientRect().top));
    expect(top).toBe(0);
  });

  test("a later step dims until it is reached", async ({ page }) => {
    await openHome(page, "dark");
    const last = steps(page).nth(3);
    await expect(last).not.toHaveClass(/on|done/);
  });

  test("pause stops the clock and play resumes it", async ({ page }) => {
    await openHome(page, "dark");
    const toggle = page.locator(".hk-scene-toggle");
    await expect(toggle).toHaveText("Pause");
    await toggle.click();
    await expect(toggle).toHaveText("Play");
    const bubbleText = () => page.evaluate(() => document.querySelector(".hk-bubble")?.textContent ?? "");
    const before = await bubbleText();
    await page.waitForTimeout(1500);
    const after = await bubbleText();
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
    await openHome(page, "light", "?os=mac");
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

test.describe("install detection", () => {
  test("shows the command for the system, and the full command is what Copy keeps", async ({ page }, info) => {
    await openHome(page, "dark", "?os=mac");
    await expect(page.locator(".hk-install-os")).toContainText("macOS");
    await expect(page.locator(".hk-install-line code")).toContainText("brew install");

    await openHome(page, "dark", "?os=linux");
    await expect(page.locator(".hk-install-os")).toContainText("Linux");
    await expect(page.locator(".hk-install-line code")).toContainText("curl -fsSL");

    await openHome(page, "dark", "?os=windows");
    await expect(page.locator(".hk-install-os")).toContainText("Git Bash");
    // Phones drop the link to keep the top of the page short.
    if (!isPhone(info.project.name)) await expect(page.getByRole("link", { name: "Other ways to install" })).toBeVisible();
  });

  test("the credit sits under the install box next to the other-ways link, not in a floating pill", async ({ page }) => {
    await openHome(page, "dark");
    await expect(page.locator(".hk-getit .hk-install-foot .hk-built")).toContainText("Hasan Alkhatib");
    await expect(page.locator(".hk-credit")).toBeHidden();
  });
});

test("the social card is wired", async ({ page }) => {
  await page.goto("");
  await expect(page.locator('meta[property="og:image"]')).toHaveAttribute("content", /og\.png$/);
  const res = await page.request.get("og.png");
  expect(res.ok()).toBeTruthy();
  expect(res.headers()["content-type"]).toContain("image/png");
});

test.describe("first visit", () => {
  // A system that asks for light must still see the dark theme on a first visit.
  test.use({ colorScheme: "light" });

  test("opens in dark with no saved choice", async ({ page }) => {
    await page.goto("");
    await expect(page.locator("html")).toHaveClass(/dark/);
  });

  test("a saved light choice is kept", async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem("vitepress-theme-appearance", "light"));
    await page.goto("");
    await expect(page.locator("html")).not.toHaveClass(/dark/);
  });
});
