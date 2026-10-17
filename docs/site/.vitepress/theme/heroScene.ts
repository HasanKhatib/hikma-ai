// The home page scene: a pixel-art loop that tells the Hikma story.
// Everything is a pure function of time (scene seconds into the story, played slower),
// so the same code can draw a live canvas or render frames for a video.

export const W = 240;
export const H = 135;
export const GROUND = 92;
export const STORY = 22; // length of the story, in scene seconds
export const INTRO = 1.4; // a title screen before the story starts
export const SLOW = 1.6; // real seconds per scene second
export const LOOP = (STORY + INTRO) * SLOW; // real length of one loop

const HOLE_A = 142;
const HOLE_B = 172;
const FLAG_X = 118;
const CHEST_X = 206;

type Palette = Record<string, string>;

const PAL: Palette = {
  h: "#e0ae4e", // hood
  y: "#f2b84b", // hoodie
  k: "#f6d9b0", // face
  e: "#14201e", // eyes
  l: "#d3ccb8", // legs
  b: "#2fa89f", // bot
  w: "#f1ede2", // bot eye
  o: "#ff9a6b", // antenna tip
  g: "#4aa8ff", // first skill, blue like the bot it turns blue
  q: "#e6f3ff", // gem glint
  d: "#2468b3", // gem shade
  n: "#a78bfa", // second skill, violet like the bot with two skills
  v: "#6d4fc4", // second gem shade
  m: "#c4622f", // chest
  s: "#e8c27a", // chest trim
};

const CHAR = [
  "..hhhh..",
  ".hhhhhh.",
  ".hkkkkh.",
  ".hkekeh.",
  "..kkkk..",
  ".yyyyyy.",
  "yyyyyyyy",
  "y.yyyy.y",
  ".yyyyyy.",
];
const LEGS_A = ["..l..l.."];
const LEGS_B = [".l....l."];

const PET = ["...o....", "...b....", ".bbbbbb.", "bbwbbwbb", "bbbbbbbb", ".bb..bb."];
const PET_BLINK = ["...o....", "...b....", ".bbbbbb.", "bbbbbbbb", "bbbbbbbb", ".bb..bb."];

// The pet is dark with no skills, turns blue with the first, and violet with two.
const PET_COLORS = ["#44504e", "#4aa8ff", "#a78bfa"];

const GEM = ["..g..", ".gqg.", "ggggg", ".gdg.", "..g.."];
const GEM2 = ["..n..", ".nqn.", "nnnnn", ".nvn.", "..n.."];
const CHEST = ["..mmmmmmmmmm..", ".mssssssssssm.", ".mmmmmsmmmmmm.", ".mmmmmmmmmmmm.", ".mmmmmmmmmmmm.", ".mmmmmmmmmmmm."];
const SIGN = ["wwwww", "wlllw", "wwwww", "..d..", "..d.."];
const FLAG = ["d", "d", "d", "d", "d", "d", "d", "d", "d", "d"];

type Sprite = HTMLCanvasElement;
let sprites: Record<string, Sprite> | null = null;

function make(rows: string[], override: Palette = {}): Sprite {
  const c = document.createElement("canvas");
  c.width = rows[0].length;
  c.height = rows.length;
  const x = c.getContext("2d")!;
  rows.forEach((row, j) =>
    [...row].forEach((ch, i) => {
      const color = override[ch] ?? PAL[ch];
      if (ch === "." || !color) return;
      x.fillStyle = color;
      x.fillRect(i, j, 1, 1);
    }),
  );
  return c;
}

function getSprites() {
  if (sprites) return sprites;
  sprites = {
    charA: make([...CHAR, ...LEGS_A]),
    charB: make([...CHAR, ...LEGS_B]),
    pet0: make(PET, { b: PET_COLORS[0] }),
    pet1: make(PET, { b: PET_COLORS[1] }),
    pet2: make(PET, { b: PET_COLORS[2] }),
    petBlink0: make(PET_BLINK, { b: PET_COLORS[0] }),
    petBlink1: make(PET_BLINK, { b: PET_COLORS[1] }),
    petBlink2: make(PET_BLINK, { b: PET_COLORS[2] }),
    gem: make(GEM),
    gem2: make(GEM2),
    chest: make(CHEST),
    sign: make(SIGN),
  };
  return sprites;
}

const SC = 2; // sprite scale: each sprite pixel is 2x2 scene pixels

function blit(ctx: CanvasRenderingContext2D, img: HTMLCanvasElement, x: number, y: number) {
  ctx.drawImage(img, Math.round(x), Math.round(y), img.width * SC, img.height * SC);
}

const clamp = (v: number, a = 0, b = 1) => Math.min(b, Math.max(a, v));
const lerp = (a: number, b: number, p: number) => a + (b - a) * p;
const seg = (t: number, a: number, b: number) => clamp((t - a) / (b - a));
const ease = (p: number) => p * p * (3 - 2 * p);

// ── Character and pet ────────────────────────────────────────────────────────

function charX(t: number): number {
  if (t < 3) return 28;
  if (t < 8) return lerp(28, 92, seg(t, 3, 8));
  if (t < 14.2) return 92;
  if (t < 15.4) return lerp(92, 152, seg(t, 14.2, 15.4));
  if (t < 16) return 152;
  if (t < 19.6) return FLAG_X;
  if (t < 21.2) return lerp(FLAG_X, 178, ease(seg(t, 19.6, 21.2)));
  return lerp(178, 188, seg(t, 21.2, 22));
}

function charMoving(t: number): boolean {
  return (t > 3 && t < 8) || (t > 14.2 && t < 15.4) || t > 21.2 || (t > 19.6 && t < 21.2);
}

function charY(t: number): number {
  let y = GROUND - 20;
  if (t >= 15.4 && t < 16) y += ((t - 15.4) / 0.6) ** 2 * 44;
  if (t >= 19.6 && t < 21.2) y -= Math.sin(Math.PI * seg(t, 19.6, 21.2)) * 34;
  return y;
}

function charVisible(t: number): boolean {
  if (t >= 16 && t < 16.9) return false;
  if (t >= 16.9 && t < 17.5) return Math.floor(t * 10) % 2 === 0;
  return true;
}

function petPos(t: number): { x: number; y: number } {
  let x = charX(Math.max(0, t - 0.6)) - 22;
  if (t >= 14.9 && t < 17.2) x = 122;
  else if (t >= 17.2 && t < 17.9) x = lerp(122, charX(17.3) - 22, ease(seg(t, 17.2, 17.9)));
  let y = GROUND - 12 + Math.round(Math.sin(t * 5));
  if (t >= 19.8 && t < 21.4) {
    x = charX(t - 0.25) - 18;
    y += -Math.sin(Math.PI * seg(t, 19.8, 21.4)) * 28 + 6;
  }
  return { x, y };
}

function skillCount(t: number): number {
  if (t < 6.6 || (t >= 16 && t < 18)) return 0;
  if (t >= 18 && t < 18.3) return 1;
  if (t >= 18) return 2;
  return t >= 10.6 ? 2 : 1;
}

// ── Drawing ──────────────────────────────────────────────────────────────────

function px(ctx: CanvasRenderingContext2D, color: string, x: number, y: number, w = 1, h = 1) {
  ctx.fillStyle = color;
  ctx.fillRect(Math.round(x), Math.round(y), w, h);
}

function burst(ctx: CanvasRenderingContext2D, cx: number, cy: number, t0: number, t: number, color = "#4aa8ff") {
  const dt = t - t0;
  if (dt < 0 || dt > 0.7) return;
  for (let i = 0; i < 8; i++) {
    const a = (i / 8) * Math.PI * 2;
    const r = 5 + dt * 30;
    px(ctx, i % 2 ? "#e6f3ff" : color, cx + Math.cos(a) * r, cy + Math.sin(a) * r, 1, 1);
  }
}

const THEMES = {
  dark: { sky: "#0b1514", mid: "#0f1d1b", low: "#12302d", hills: "#163d39", grass: "#2fa89f", soil: "#136761", dirt: "#0b3b37", speck: "#0f4a45", hole: "#050b0a" },
  light: { sky: "#cfe7e1", mid: "#dcefe9", low: "#e9f5f1", hills: "#b4d9cf", grass: "#2a8f84", soil: "#5aa79c", dirt: "#8cc2b8", speck: "#7bb5aa", hole: "#1b2f2c" },
};

function drawWorld(ctx: CanvasRenderingContext2D, t: number, dark: boolean) {
  const c = dark ? THEMES.dark : THEMES.light;
  px(ctx, c.sky, 0, 0, W, H);
  px(ctx, c.mid, 0, 28, W, 30);
  px(ctx, c.low, 0, 58, W, GROUND - 58);
  // distant hills
  for (let x = 0; x < W; x += 2) {
    const h = 10 + Math.round(Math.sin(x / 17) * 5 + Math.sin(x / 7) * 2);
    px(ctx, c.hills, x, GROUND - h, 2, h);
  }
  if (dark) {
    for (let i = 0; i < 18; i++) {
      const sx = (i * 53) % W;
      const sy = 4 + ((i * 29) % 40);
      if (Math.floor(t * 1.5 + i) % 3) px(ctx, "#f2c46d", sx, sy);
    }
  } else {
    // a blocky sun and two slow clouds
    px(ctx, "#f2c46d", 196, 12, 14, 14);
    px(ctx, "#ffe39a", 199, 15, 8, 8);
    for (const [cx, cy] of [[30, 16], [120, 30]] as const) {
      const x = ((cx + t * 3) % (W + 40)) - 20;
      px(ctx, "#ffffff", x, cy, 22, 5);
      px(ctx, "#ffffff", x + 5, cy - 4, 12, 4);
    }
  }
  // ground, with a hole
  px(ctx, c.hole, HOLE_A, GROUND, HOLE_B - HOLE_A, H - GROUND);
  for (let x = 0; x < W; x++) {
    if (x >= HOLE_A && x < HOLE_B) continue;
    px(ctx, c.grass, x, GROUND, 1, 2);
    px(ctx, c.soil, x, GROUND + 2, 1, 4);
    px(ctx, c.dirt, x, GROUND + 6, 1, H - GROUND - 6);
    if ((x * 7) % 11 === 0) px(ctx, c.speck, x, GROUND + 8 + (x % 5) * 3, 2, 1);
  }
  px(ctx, "#ff9a6b", HOLE_A - 2, GROUND - 1, 2, 1);
  px(ctx, "#ff9a6b", HOLE_B, GROUND - 1, 2, 1);
}

function drawProps(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  // signposts: the repo gets set up
  for (const [x, at] of [[46, 1.4], [62, 1.7]] as const) {
    const p = ease(seg(t, at, at + 0.4));
    if (p > 0) blit(ctx, s.sign, x, GROUND - 10 * p);
  }
  // checkpoint flag
  if (t >= 14.4 && t < 22) {
    px(ctx, "#f1ede2", FLAG_X - 14, GROUND - 22, 2, 22);
    px(ctx, "#ff9a6b", FLAG_X - 12, GROUND - 22, 9, 6);
  }
  // registry chest
  if (t >= 12 && t < 22) {
    const rise = ease(seg(t, 12, 12.4));
    blit(ctx, s.chest, CHEST_X, GROUND - 12 * rise);
    if (t >= 14 && t < 14.8) px(ctx, "#fff7d6", CHEST_X + 4, GROUND - 16, 20, 3);
  }
}

function drawGems(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  const pet = petPos(t);
  // gem 1: found, then absorbed by the pet
  if (t >= 3 && t < 6.6) {
    const bob = Math.sin(t * 4) * 1.5;
    let x = 104;
    let y = GROUND - 38 + bob;
    if (t >= 5.6) {
      const p = ease(seg(t, 5.6, 6.6));
      x = lerp(104, pet.x + 3, p);
      y = lerp(GROUND - 38, pet.y - 10, p);
    }
    blit(ctx, s.gem, x, y);
  }
  burst(ctx, pet.x + 8, pet.y + 4, 6.6, t);
  // gem 2: the new skill, pushed to the registry
  if (t >= 10 && t < 14) {
    const bob = Math.sin(t * 4) * 1.5;
    let x = charX(t) + 3;
    let y = GROUND - 46 + bob;
    if (t >= 12.4) {
      const p = seg(t, 12.4, 14);
      x = lerp(charX(t) + 3, CHEST_X + 9, p);
      y = lerp(GROUND - 46, GROUND - 16, p) - Math.sin(Math.PI * p) * 40;
    }
    blit(ctx, s.gem2, x, y);
  }
  burst(ctx, CHEST_X + 14, GROUND - 10, 14, t, "#a78bfa");
  burst(ctx, pet.x + 8, pet.y + 4, 18, t);
  burst(ctx, pet.x + 8, pet.y + 4, 18.3, t, "#a78bfa");
}

function drawPet(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  const { x, y } = petPos(t);
  const blink = (((t % 3.2) + 3.2) % 3.2) < 0.12;
  const tone = Math.min(2, skillCount(t));
  blit(ctx, s[(blink ? "petBlink" : "pet") + tone], x, y);
  // skills the pet knows, shown as tiny gems above its antenna
  const n = skillCount(t);
  for (let i = 0; i < n; i++) {
    px(ctx, i === 0 ? "#4aa8ff" : "#a78bfa", x + 2 + i * 6, y - 8, 4, 4);
  }
  if (t >= 16 && t < 17.9) {
    px(ctx, "#a9b0a6", x + 7, y - 6, 2, 2);
    px(ctx, "#a9b0a6", x + 7, y - 12, 2, 4);
  }
}

function drawChar(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  if (!charVisible(t)) {
    // pop into pixels
    const dt = t - 16;
    if (dt >= 0 && dt < 0.9) {
      for (let i = 0; i < 14; i++) {
        const a = (i / 14) * Math.PI * 2 + i;
        const sp = 14 + (i % 4) * 7;
        px(
          ctx,
          i % 3 ? "#f2b84b" : "#f6d9b0",
          152 + 8 + Math.cos(a) * sp * dt,
          GROUND + 14 + Math.sin(a) * sp * dt + 50 * dt * dt,
          2,
          2,
        );
      }
    }
    return;
  }
  const sprite = charMoving(t) && Math.floor(t * 6) % 2 ? s.charB : s.charA;
  blit(ctx, sprite, charX(t), charY(t));
}

// Real seconds into the loop -> scene seconds into the story (negative on the title screen).
function sceneTime(raw: number): number {
  return (((raw % LOOP) + LOOP) % LOOP) / SLOW - INTRO;
}

export function drawScene(ctx: CanvasRenderingContext2D, raw: number, dark = true) {
  const t = sceneTime(raw);
  const story = t + INTRO;
  const s = getSprites();
  ctx.imageSmoothingEnabled = false;
  drawWorld(ctx, t, dark);
  drawProps(ctx, t, s);
  drawGems(ctx, t, s);
  drawPet(ctx, t, s);
  drawChar(ctx, t, s);
  // dim behind the title screen, fade in from black, and fade out at the end of the loop
  let fade = story < INTRO ? 0.5 * (1 - seg(story, INTRO - 0.5, INTRO)) : 0;
  fade = Math.max(fade, 1 - seg(story, 0, 0.3), seg(t, STORY - 0.4, STORY));
  if (fade > 0) {
    ctx.globalAlpha = fade;
    px(ctx, dark ? "#0b1514" : "#e9f5f1", 0, 0, W, H);
    ctx.globalAlpha = 1;
  }
}

// ── Text overlays ────────────────────────────────────────────────────────────

export type Balloon = { who: "char" | "pet"; text: string; x: number; y: number };

const BALLOONS: { a: number; b: number; who: "char" | "pet"; text: string }[] = [
  { a: 0.3, b: 3.0, who: "char", text: "New repo." },
  { a: 3.4, b: 7.2, who: "char", text: "Great skill for my agent!" },
  { a: 8.2, b: 13.2, who: "pet", text: "I learned a lot. Let's make walking-above-holes." },
  { a: 14.9, b: 15.7, who: "pet", text: "Hole!" },
  { a: 17.1, b: 19.0, who: "char", text: "New laptop. Same skills." },
  { a: 19.2, b: 20.4, who: "pet", text: "Watch this." },
];

export function balloonAt(raw: number): Balloon | null {
  const t = sceneTime(raw);
  const b = BALLOONS.find((x) => t >= x.a && t < x.b);
  if (!b) return null;
  const p = b.who === "pet" ? petPos(t) : { x: charX(t), y: charY(t) };
  const x = clamp((p.x + 8) / W, 0.24, 0.76) * 100;
  const y = ((p.y - (b.who === "pet" ? 14 : 6)) / H) * 100;
  return { who: b.who, text: b.text, x, y };
}

type Cmd = { at: number; text: string; out?: boolean; group: number };

const CLI: Cmd[] = [
  { group: 1, at: 0.4, text: "hikma init" },
  { group: 1, at: 1.4, text: "Created AGENTS.md", out: true },
  { group: 1, at: 1.7, text: "Wrote .hikma/config.json", out: true },
  { group: 2, at: 3.4, text: "hikma skill install go-blue" },
  { group: 2, at: 5.3, text: "-> .claude/skills/go-blue/ (1 files)", out: true },
  { group: 3, at: 8.2, text: "hikma skill create walking-above-holes" },
  { group: 3, at: 9.8, text: "Created .claude/skills/walking-above-holes/", out: true },
  { group: 4, at: 10.4, text: "hikma skill push walking-above-holes" },
  { group: 4, at: 12.2, text: "Pushed to myghaccount/myrepo", out: true },
  { group: 5, at: 17.0, text: "hikma sync" },
  { group: 5, at: 18.0, text: "restored .claude/skills/go-blue", out: true },
  { group: 5, at: 18.3, text: "restored .claude/skills/walking-above-holes", out: true },
  { group: 5, at: 18.6, text: "Done: 2 restored, 0 failed", out: true },
];

const GROUPS: [number, number, number][] = [
  [1, 0.4, 3.2],
  [2, 3.4, 8.0],
  [3, 8.2, 10.3],
  [4, 10.4, 14.2],
  [5, 17.0, 21.4],
];

export type CliLine = { out: boolean; text: string };

export function cliAt(raw: number): CliLine[] {
  const t = sceneTime(raw);
  const g = GROUPS.find(([, a, b]) => t >= a && t < b);
  if (!g) return [];
  const lines: CliLine[] = [];
  for (const c of CLI) {
    if (c.group !== g[0] || t < c.at) continue;
    if (c.out) {
      lines.push({ out: true, text: c.text });
    } else {
      const n = Math.min(c.text.length, Math.floor((t - c.at) / 0.045));
      lines.push({ out: false, text: c.text.slice(0, n) });
    }
  }
  return lines.slice(-3);
}

export const TRANSCRIPT =
  "A pixel-art loop. A developer and a small bot pet start a new repo with hikma init, find a skill called go-blue and install it, which turns the bot blue, create a new skill called walking-above-holes and push it to a registry, fall into a hole, and respawn on a new laptop where hikma sync restores their skills.";

// ── Stages ───────────────────────────────────────────────────────────────────

export type Stage = { step: string; label: string; start: number };

// `start` is in real seconds into the loop, so a click can seek straight to it.
const at = (sceneStart: number) => (sceneStart + INTRO) * SLOW;

export const STAGES: Stage[] = [
  { step: "1", label: "hikma init", start: 0 },
  { step: "2", label: "skill install", start: at(2.8) },
  { step: "3", label: "skill create + push", start: at(7.9) },
  { step: "4", label: "hikma sync", start: at(14.4) },
];

export function stageAt(raw: number): number {
  const r = ((raw % LOOP) + LOOP) % LOOP;
  let active = 0;
  STAGES.forEach((st, i) => {
    if (r >= st.start) active = i;
  });
  return active;
}

// 1 on the title screen, fading to 0 as the story starts.
export function titleAlpha(raw: number): number {
  const story = sceneTime(raw) + INTRO;
  return story < INTRO ? 1 - seg(story, INTRO - 0.5, INTRO) : 0;
}

// A frame that reads well without motion: the pet tells its plan and the CLI shows create and push.
export const STILL_T = at(12.6);
