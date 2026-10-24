// Renders the home page scene to PNG frames, one per 1/FPS second of the loop.
//
//   cd docs/site && npm run docs:build && npx vitepress preview . --port 4311 &
//   node ../../.claude/skills/video-pixel-scene/assets/render-frames.mjs <out-dir> [dark|light]
//
// Then assemble with ffmpeg (see SKILL.md). Run from docs/site so @playwright/test resolves.
import { createRequire } from "node:module";

const require = createRequire(`${process.cwd()}/`);
const { chromium } = require("@playwright/test");

const OUT = process.argv[2];
const THEME = process.argv[3] ?? "dark";
if (!OUT) throw new Error("usage: render-frames.mjs <out-dir> [dark|light]");
const FPS = 15;
const LOOP = 49.92; // LOOP in heroScene.ts, in real seconds
const WORKERS = 6;
const total = Math.floor(LOOP * FPS);

const browser = await chromium.launch();
let next = 0;

async function worker() {
  const ctx = await browser.newContext({ viewport: { width: 1366, height: 760 }, deviceScaleFactor: 2 });
  await ctx.addInitScript((t) => localStorage.setItem("vitepress-theme-appearance", t), THEME);
  const page = await ctx.newPage();
  for (let i = next++; i < total; i = next++) {
    const t = Math.max(0.001, i / FPS).toFixed(3);
    await page.goto(`http://localhost:4311/?t=${t}`, { waitUntil: "load" });
    // Hide the Pause button and freeze CSS animation so every frame is a clean still.
    await page.addStyleTag({ content: ".hk-scene-toggle{display:none!important}*{animation:none!important;transition:none!important}" });
    const scene = page.locator(".hk-scene");
    await scene.waitFor();
    await scene.screenshot({ path: `${OUT}/f${String(i).padStart(4, "0")}.png` });
  }
  await ctx.close();
}

await Promise.all(Array.from({ length: WORKERS }, worker));
await browser.close();
console.log(`${total} frames in ${OUT}`);
