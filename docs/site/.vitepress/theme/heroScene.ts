// The home page scene: a pixel-art loop that tells the Hikma story.
// Everything is a pure function of time (scene seconds into the story, played slower),
// so the same code can draw a live canvas or render frames for a video.

export const W = 240; // the stage the story is drawn on; wider windows get more world either side of it
export const MAX_VW = 560;
export const H = 135;
export const GROUND = 92;
export const STORY = 24.8; // length of the story, in scene seconds
export const INTRO = 1.4; // a title screen before the story starts
export const SLOW = 1.6; // real seconds per scene second
export const LOOP = (STORY + INTRO) * SLOW; // real length of one loop

// The canvas can be wider than the stage. The story stays centered; the world fills the rest.
// On a narrow screen (a phone) the canvas is narrower than the stage and a camera follows the character.
const MIN_VW = 112;
let VW = W;
let OFF = 0;
let SHIFT = 0; // where the stage's left edge sits on the canvas right now: the centering offset, or minus the camera
export function setView(width: number) {
  VW = Math.max(MIN_VW, Math.min(MAX_VW, Math.round(width)));
  OFF = Math.max(0, Math.floor((VW - W) / 2));
  SHIFT = OFF;
  return VW;
}

const HOLE_A = 142;
const HOLE_B = 172;
const FLAG_X = 118;
const CHEST_X = 206;

// Stage 4 happens in a second world: a "WORLD 2" card, then a lava land where the pet starts with no skills.
const W2 = 15; // the world changes here, hidden under the card
const FALL_A = 18; // the hero walks off the edge of the lava hole
const FALL_B = 18.7;
const RESPAWN = 19.5; // back at the checkpoint flag
const SYNC_A = 21.3; // first skill restored
const SYNC_B = 21.6; // second skill restored
const JUMP_A = 22.2;
const JUMP_B = 23.8;

type Palette = Record<string, string>;

const PAL: Palette = {
  u: "#f4f1e6", // ghutra, the white headcloth
  a: "#1c1a18", // agal, the black ring that holds it
  k: "#d8a574", // face
  e: "#14201e", // eyes
  t: "#f7f4ea", // thobe
  T: "#d9d3c0", // thobe shade
  r: "#c98f2e", // gold collar
  l: "#8a6a45", // sandals
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
  "..uuuuuu..",
  ".uuuuuuuu.",
  ".uaaaaaau.",
  ".uukkkkuu.",
  ".uukekeuu.",
  ".uukkkkuu.",
  "..uuuuuu..",
  "...rrrr...",
  "..tttttt..",
  "..ttTttt..",
  ".tttTtttt.",
  ".tttTtttt.",
  ".tttttttt.",
];
const CH = 28; // sprite height in scene pixels: 14 rows at 2x
const LEGS_A = ["...l..l..."];
const LEGS_B = ["..l....l.."];

const PET = ["...o....", "...b....", ".bbbbbb.", "bbwbbwbb", "bbbbbbbb", ".bb..bb."];
const PET_BLINK = ["...o....", "...b....", ".bbbbbb.", "bbbbbbbb", "bbbbbbbb", ".bb..bb."];

// The pet is dark with no skills, turns blue with the first, and violet with two.
const PET_COLORS = ["#44504e", "#4aa8ff", "#a78bfa"];

const GEM = ["..g..", ".gqg.", "ggggg", ".gdg.", "..g.."];
const GEM2 = ["..n..", ".nqn.", "nnnnn", ".nvn.", "..n.."];
const CHEST = ["..mmmmmmmmmm..", ".mssssssssssm.", ".mmmmmsmmmmmm.", ".mmmmmmmmmmmm.", ".mmmmmmmmmmmm.", ".mmmmmmmmmmmm."];
const SIGN = ["wwwww", "wlllw", "wwwww", "..d..", "..d.."];
const FLAG = ["d", "d", "d", "d", "d", "d", "d", "d", "d", "d"];

type Sprite = HTMLCanvasElement & { pad?: number };
const sprites: Partial<Record<"dark" | "light", Record<string, Sprite>>> = {};

// An outline keeps the white robes readable on the pale daytime sky.
function make(rows: string[], override: Palette = {}, outline?: string): Sprite {
  const pad = outline ? 1 : 0;
  const c: Sprite = document.createElement("canvas");
  c.width = rows[0].length + pad * 2;
  c.height = rows.length + pad * 2;
  c.pad = pad;
  const x = c.getContext("2d")!;
  const cells: [number, number, string][] = [];
  rows.forEach((row, j) =>
    [...row].forEach((ch, i) => {
      const color = override[ch] ?? PAL[ch];
      if (ch !== "." && color) cells.push([i + pad, j + pad, color]);
    }),
  );
  if (outline) {
    x.fillStyle = outline;
    for (const [i, j] of cells) for (const [dx, dy] of [[1, 0], [-1, 0], [0, 1], [0, -1]]) x.fillRect(i + dx, j + dy, 1, 1);
  }
  for (const [i, j, color] of cells) {
    x.fillStyle = color;
    x.fillRect(i, j, 1, 1);
  }
  return c;
}

function getSprites(dark: boolean) {
  const key = dark ? "dark" : "light";
  const have = sprites[key];
  if (have) return have;
  const edge = dark ? undefined : "#3b2f22";
  const made = {
    charA: make([...CHAR, ...LEGS_A], {}, edge),
    charB: make([...CHAR, ...LEGS_B], {}, edge),
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
  sprites[key] = made;
  return made;
}

const SC = 2; // sprite scale: each sprite pixel is 2x2 scene pixels

function blit(ctx: CanvasRenderingContext2D, img: Sprite, x: number, y: number) {
  const pad = (img.pad ?? 0) * SC;
  ctx.drawImage(img, Math.round(x) - pad, Math.round(y) - pad, img.width * SC, img.height * SC);
}

const clamp = (v: number, a = 0, b = 1) => Math.min(b, Math.max(a, v));
const lerp = (a: number, b: number, p: number) => a + (b - a) * p;
const seg = (t: number, a: number, b: number) => clamp((t - a) / (b - a));
const ease = (p: number) => p * p * (3 - 2 * p);

// ── Character and pet ────────────────────────────────────────────────────────

function charX(t: number): number {
  if (t < 3) return 28;
  if (t < 8) return lerp(28, 92, seg(t, 3, 8));
  if (t < W2) return 92;
  if (t < 16.4) return 28;
  if (t < FALL_A) return lerp(28, 134, seg(t, 16.4, FALL_A));
  if (t < FALL_B) return lerp(134, 146, seg(t, FALL_A, FALL_B));
  if (t < RESPAWN) return 146;
  if (t < JUMP_A) return FLAG_X;
  if (t < JUMP_B) return lerp(FLAG_X, 178, ease(seg(t, JUMP_A, JUMP_B)));
  return lerp(178, 188, seg(t, JUMP_B, STORY));
}

// What the camera looks at: the character, then the chest while a skill is pushed, and the checkpoint after the fall.
function camTarget(t: number): number {
  if (t >= FALL_B && t < RESPAWN + 0.6) return lerp(146, FLAG_X, ease(seg(t, FALL_B, RESPAWN + 0.6)));
  const chest = seg(t, 11.8, 12.8) * (1 - seg(t, 14.6, 14.9));
  return lerp(charX(t), CHEST_X - 20, chest);
}

function applyCamera(t: number) {
  SHIFT = VW >= W ? OFF : -clamp(camTarget(t) + 8 - VW * 0.42, 0, W - VW);
}

function charMoving(t: number): boolean {
  return (t > 3 && t < 8) || (t > 16.4 && t < FALL_B) || t > JUMP_A;
}

function charY(t: number): number {
  let y = GROUND - CH;
  if (t >= FALL_A && t < FALL_B) y += ((t - FALL_A) / (FALL_B - FALL_A)) ** 2 * 60;
  if (t >= JUMP_A && t < JUMP_B) y -= Math.sin(Math.PI * seg(t, JUMP_A, JUMP_B)) * 34;
  return y;
}

function charVisible(t: number): boolean {
  if (t >= FALL_B && t < RESPAWN) return false;
  if (t >= RESPAWN && t < RESPAWN + 0.6) return Math.floor(t * 10) % 2 === 0;
  return true;
}

function petPos(t: number): { x: number; y: number } {
  let x = charX(Math.max(0, t - 0.6)) - 22;
  let y = GROUND - 12 + Math.round(Math.sin(t * 5));
  if (t >= FALL_A && t < 18.5) x = lerp(charX(FALL_A - 0.6) - 22, 138, seg(t, FALL_A, 18.5));
  else if (t >= 18.5 && t < 19.2) {
    const p = seg(t, 18.5, 19.2);
    x = lerp(138, 148, p);
    y += p * p * 62;
  }
  if (t >= JUMP_A + 0.2 && t < JUMP_B + 0.2) {
    x = charX(t - 0.25) - 18;
    y += -Math.sin(Math.PI * seg(t, JUMP_A + 0.2, JUMP_B + 0.2)) * 28 + 6;
  }
  return { x, y };
}

function petVisible(t: number): boolean {
  return !(t >= 19.2 && t < 19.6);
}

// The pet is dark with no skills, blue with the first, violet with two. It loses them in the new world.
function skillCount(t: number): number {
  if (t < 6.6 || (t >= W2 && t < SYNC_A)) return 0;
  if (t >= SYNC_B) return 2;
  if (t >= SYNC_A) return 1;
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
  light: { sky: "#e4d2a6", mid: "#e9d9b0", low: "#eddfba", hills: "#cdbb90", grass: "#6f7d2e", soil: "#a8803f", dirt: "#b9915a", speck: "#a8834f", hole: "#2a2118" },
};

// World 2: a lava land.
const LAVA = {
  dark: { sky: "#150808", mid: "#2a0d0a", low: "#4a170d", hills: "#3a120c", grass: "#8a3a1c", soil: "#4a1a10", dirt: "#2a0e09", speck: "#3a150d", hole: "#ff5a1f" },
  light: { sky: "#e8b98c", mid: "#e5ab79", low: "#dd9868", hills: "#b9694a", grass: "#8d4a2a", soil: "#a85a37", dirt: "#85442a", speck: "#97502f", hole: "#ff5a1f" },
};

function drawLava(ctx: CanvasRenderingContext2D, t: number, dark: boolean) {
  const HA = HOLE_A + SHIFT;
  const HB = HOLE_B + SHIFT;
  const c = dark ? LAVA.dark : LAVA.light;
  px(ctx, c.sky, 0, 0, VW, H);
  px(ctx, c.mid, 0, 28, VW, 30);
  px(ctx, c.low, 0, 58, VW, GROUND - 58);
  // jagged ridge, with a volcano that glows
  for (let x = 0; x < VW; x += 2) {
    const h = 12 + Math.round(Math.abs(Math.sin(x / 13)) * 14);
    px(ctx, c.hills, x, GROUND - h, 2, h);
  }
  for (let i = 0; i < 44; i++) {
    const w = 44 - i;
    px(ctx, c.hills, VW - 44 - w / 2, GROUND - 6 - i, w, 1);
  }
  px(ctx, "#ff7a2a", VW - 50, GROUND - 52, 12, 2);
  px(ctx, "#ffd24a", VW - 47, GROUND - 53, 6, 1);
  // drifting embers
  for (let i = 0; i < 16; i++) {
    const ex = (i * 47 + t * 6) % VW;
    const ey = (GROUND - ((t * 14 + i * 23) % (GROUND + 10))) | 0;
    px(ctx, i % 2 ? "#ff9a3a" : "#ffd24a", ex, ey);
  }
  // ground, with a lava-filled hole
  for (let x = 0; x < VW; x++) {
    if (x >= HA && x < HB) continue;
    px(ctx, c.grass, x, GROUND, 1, 2);
    px(ctx, c.soil, x, GROUND + 2, 1, 4);
    px(ctx, c.dirt, x, GROUND + 6, 1, H - GROUND - 6);
    if ((x * 7) % 11 === 0) px(ctx, c.speck, x, GROUND + 8 + (x % 5) * 3, 2, 1);
  }
  px(ctx, c.hole, HA, GROUND, HB - HA, H - GROUND);
  for (let x = HA; x < HB; x += 3) {
    const row = Math.floor(t * 3 + x / 3) % 4;
    px(ctx, "#ffd24a", x, GROUND + 4 + row * 12, 3, 2);
    px(ctx, "#c2330f", x + 1, GROUND + 10 + row * 9, 2, 2);
  }
  px(ctx, "#ffd24a", HA - 2, GROUND - 1, 2, 1);
  px(ctx, "#ffd24a", HB, GROUND - 1, 2, 1);
}

function drawWorld(ctx: CanvasRenderingContext2D, t: number, dark: boolean) {
  const HA = HOLE_A + SHIFT;
  const HB = HOLE_B + SHIFT;
  if (t >= W2) return drawLava(ctx, t, dark);
  const c = dark ? THEMES.dark : THEMES.light;
  px(ctx, c.sky, 0, 0, VW, H);
  px(ctx, c.mid, 0, 28, VW, 30);
  px(ctx, c.low, 0, 58, VW, GROUND - 58);
  // distant hills
  for (let x = 0; x < VW; x += 2) {
    const h = 10 + Math.round(Math.sin(x / 17) * 5 + Math.sin(x / 7) * 2);
    px(ctx, c.hills, x, GROUND - h, 2, h);
  }
  if (dark) {
    for (let i = 0; i < 18; i++) {
      const sx = (i * 53) % VW;
      const sy = 40 + ((i * 29) % 36); // below the nav bar that sits over the top of the page
      if (Math.floor(t * 1.5 + i) % 3) px(ctx, "#f2c46d", sx, sy);
    }
  } else {
    // a blocky sun and two slow clouds
    px(ctx, "#d79921", VW - 92, 30, 10, 10);
    px(ctx, "#ebbd5a", VW - 90, 32, 6, 6);
    for (const [cx, cy] of [[30, 60], [120, 74]] as const) {
      const x = ((cx + t * 3) % (VW + 40)) - 20;
      px(ctx, "#f7edd2", x, cy, 22, 5);
      px(ctx, "#f7edd2", x + 5, cy - 4, 12, 4);
    }
  }
  // ground, with a hole
  px(ctx, c.hole, HA, GROUND, HB - HA, H - GROUND);
  for (let x = 0; x < VW; x++) {
    if (x >= HA && x < HB) continue;
    px(ctx, c.grass, x, GROUND, 1, 2);
    px(ctx, c.soil, x, GROUND + 2, 1, 4);
    px(ctx, c.dirt, x, GROUND + 6, 1, H - GROUND - 6);
    if ((x * 7) % 11 === 0) px(ctx, c.speck, x, GROUND + 8 + (x % 5) * 3, 2, 1);
  }
  px(ctx, "#ff9a6b", HA - 2, GROUND - 1, 2, 1);
  px(ctx, "#ff9a6b", HB, GROUND - 1, 2, 1);
}

function drawProps(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  // signposts: the repo gets set up
  for (const [x, at] of [[46, 1.4], [62, 1.7]] as const) {
    const p = ease(seg(t, at, at + 0.4));
    if (p > 0 && t < W2) blit(ctx, s.sign, x, GROUND - 10 * p);
  }
  // checkpoint flag, in the lava world
  if (t >= 16.4 && t < STORY) {
    px(ctx, "#f1ede2", FLAG_X - 14, GROUND - 22, 2, 22);
    px(ctx, "#ff9a6b", FLAG_X - 12, GROUND - 22, 9, 6);
  }
  // registry chest
  if (t >= 12 && t < W2) {
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
    let y = GROUND - 46 + bob;
    if (t >= 5.6) {
      const p = ease(seg(t, 5.6, 6.6));
      x = lerp(104, pet.x + 3, p);
      y = lerp(GROUND - 46, pet.y - 10, p);
    }
    blit(ctx, s.gem, x, y);
  }
  burst(ctx, pet.x + 8, pet.y + 4, 6.6, t);
  // gem 2: the new skill, pushed to the registry
  if (t >= 10 && t < 14) {
    const bob = Math.sin(t * 4) * 1.5;
    let x = charX(t) + 3;
    let y = GROUND - 52 + bob;
    if (t >= 12.4) {
      const p = seg(t, 12.4, 14);
      x = lerp(charX(t) + 3, CHEST_X + 9, p);
      y = lerp(GROUND - 52, GROUND - 16, p) - Math.sin(Math.PI * p) * 40;
    }
    blit(ctx, s.gem2, x, y);
  }
  burst(ctx, CHEST_X + 14, GROUND - 10, 14, t, "#a78bfa");
  // the sync brings both skills back from above
  for (const [a, b, sprite, dx] of [[20.7, SYNC_A, s.gem, -30], [21, SYNC_B, s.gem2, 30]] as const) {
    if (t >= a && t < b) {
      const p = ease(seg(t, a, b));
      blit(ctx, sprite, lerp(pet.x + dx, pet.x + 3, p), lerp(6, pet.y - 10, p));
    }
  }
  burst(ctx, pet.x + 8, pet.y + 4, SYNC_A, t);
  burst(ctx, pet.x + 8, pet.y + 4, SYNC_B, t, "#a78bfa");
}

function drawPet(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  if (!petVisible(t)) return;
  const { x, y } = petPos(t);
  const blink = (((t % 3.2) + 3.2) % 3.2) < 0.12;
  const tone = Math.min(2, skillCount(t));
  blit(ctx, s[(blink ? "petBlink" : "pet") + tone], x, y);
  // skills the pet knows, shown as tiny gems above its antenna
  const n = skillCount(t);
  for (let i = 0; i < n; i++) {
    px(ctx, i === 0 ? "#4aa8ff" : "#a78bfa", x + 2 + i * 6, y - 8, 4, 4);
  }
  if (t >= 19.6 && t < SYNC_A) {
    px(ctx, "#a9b0a6", x + 7, y - 6, 2, 2);
    px(ctx, "#a9b0a6", x + 7, y - 12, 2, 4);
  }
}

function drawChar(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  if (!charVisible(t)) {
    // pop into pixels
    const dt = t - FALL_B;
    if (dt >= 0 && dt < 0.8) {
      for (let i = 0; i < 14; i++) {
        const a = (i / 14) * Math.PI * 2 + i;
        const sp = 14 + (i % 4) * 7;
        px(
          ctx,
          i % 3 ? "#ff9a3a" : "#ffd24a",
          146 + 8 + Math.cos(a) * sp * dt,
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

// The top-edge sky color, so the page can fill any space above a canvas that is wider than tall.
export function skyAt(raw: number, dark: boolean): string {
  const t = sceneTime(raw);
  if (cardAlpha(t) > 0.5) return "#14090a";
  const c = t >= W2 ? LAVA : THEMES;
  return (dark ? c.dark : c.light).sky;
}

// The "WORLD 2" card: fades in at the end of stage 3, holds while the world changes, then fades out.
function cardAlpha(t: number): number {
  if (t < 14.3 || t >= 16.5) return 0;
  return t < 15.9 ? seg(t, 14.3, 14.9) : 1 - seg(t, 15.9, 16.5);
}

// Real seconds into the loop -> scene seconds into the story (negative on the title screen).
function sceneTime(raw: number): number {
  return (((raw % LOOP) + LOOP) % LOOP) / SLOW - INTRO;
}

export function drawScene(ctx: CanvasRenderingContext2D, raw: number, dark = true) {
  const t = sceneTime(raw);
  applyCamera(t);
  const story = t + INTRO;
  const s = getSprites(dark);
  ctx.imageSmoothingEnabled = false;
  drawWorld(ctx, t, dark);
  ctx.save();
  ctx.translate(SHIFT, 0);
  drawProps(ctx, t, s);
  drawGems(ctx, t, s);
  drawPet(ctx, t, s);
  drawChar(ctx, t, s);
  ctx.restore();
  // dim behind the title screen, fade in from black, and fade out at the end of the loop
  let fade = story < INTRO ? 0.5 * (1 - seg(story, INTRO - 0.5, INTRO)) : 0;
  fade = Math.max(fade, 1 - seg(story, 0, 0.3), seg(t, STORY - 0.4, STORY));
  if (fade > 0) {
    ctx.globalAlpha = fade;
    px(ctx, dark ? "#0b1514" : "#eadfc4", 0, 0, VW, H);
    ctx.globalAlpha = 1;
  }
  const card = cardAlpha(t);
  if (card > 0) {
    ctx.globalAlpha = card;
    px(ctx, "#14090a", 0, 0, VW, H);
    ctx.globalAlpha = 1;
  }
}

// ── Text overlays ────────────────────────────────────────────────────────────

export type Who = "char" | "pet";
export type CliLine = { out: boolean; text: string };
// One speech bubble: what the speaker says, then the terminal lines that follow in the same bubble.
export type Bubble = { who: Who; quote: string | null; lines: CliLine[]; x: number; y: number };

const QUOTES: { a: number; b: number; who: Who; text: string }[] = [
  { a: 0.3, b: 3.0, who: "char", text: "New repo." },
  { a: 3.4, b: 7.2, who: "char", text: "Great skill for my agent!" },
  { a: 8.2, b: 10.6, who: "pet", text: "I learned a lot. Let's make walking-above-holes." },
  { a: 17.1, b: 17.9, who: "char", text: "Hole!" },
  { a: 19.8, b: 21, who: "char", text: "No skills." },
  { a: 22.1, b: 23.2, who: "pet", text: "Watch this." },
];

type Cmd = { at: number; text: string; out?: boolean; group: number };

// Commands start about a second after the quote, so the speaker talks first and the terminal follows.
const CLI: Cmd[] = [
  { group: 1, at: 1.1, text: "hikma init" },
  { group: 1, at: 1.7, text: "Created AGENTS.md", out: true },
  { group: 1, at: 2.0, text: "Wrote .hikma/config.json", out: true },
  { group: 2, at: 4.3, text: "hikma skill install go-blue" },
  { group: 2, at: 5.4, text: "-> .claude/skills/go-blue/ (1 files)", out: true },
  { group: 3, at: 9.0, text: "hikma skill create walking-above-holes" },
  { group: 3, at: 10.2, text: "Created .claude/skills/walking-above-holes/", out: true },
  { group: 4, at: 10.9, text: "hikma skill push walking-above-holes" },
  { group: 4, at: 12.3, text: "Pushed to myghaccount/myrepo", out: true },
  { group: 5, at: 20.5, text: "hikma sync" },
  { group: 5, at: 21.2, text: "restored .claude/skills/go-blue", out: true },
  { group: 5, at: 21.5, text: "restored .claude/skills/walking-above-holes", out: true },
  { group: 5, at: 21.8, text: "Done: 2 restored, 0 failed", out: true },
];

// [group, start, end, who speaks]
const GROUPS: [number, number, number, Who][] = [
  [1, 0.4, 3.2, "char"],
  [2, 3.4, 8.0, "char"],
  [3, 8.2, 10.8, "pet"],
  [4, 10.8, 14.2, "char"],
  [5, 19.8, 22.0, "char"],
];

function cliLines(t: number, group: number): CliLine[] {
  const lines: CliLine[] = [];
  for (const c of CLI) {
    if (c.group !== group || t < c.at) continue;
    if (c.out) lines.push({ out: true, text: c.text });
    else lines.push({ out: false, text: c.text.slice(0, Math.floor((t - c.at) / 0.03)) });
  }
  return lines.slice(-3);
}

export function bubbleAt(raw: number): Bubble | null {
  const t = sceneTime(raw);
  applyCamera(t);
  const quote = QUOTES.find((q) => t >= q.a && t < q.b);
  const group = GROUPS.find(([, a, b]) => t >= a && t < b);
  if (!quote && !group) return null;
  const who: Who = group ? group[3] : quote!.who;
  const lines = group ? cliLines(t, group[0]) : [];
  const p = who === "pet" ? petPos(t) : { x: charX(t), y: charY(t) };
  const x = ((p.x + 8 + SHIFT) / VW) * 100;
  const y = ((p.y - (who === "pet" ? 14 : 6)) / H) * 100;
  return { who, quote: quote ? quote.text : null, lines, x, y };
}

export const TRANSCRIPT =
  "A pixel-art loop. A developer and a small bot pet start a new repo with hikma init, find a skill called go-blue and install it, which turns the bot blue, then create a new skill called walking-above-holes and push it to a registry. In World 2, a lava land on a new machine, the bot is dark again with no skills. They fall into a hole, respawn, and run hikma sync, which restores both skills, and the bot turns blue and then violet. Now they jump the hole.";

// ── Stages ───────────────────────────────────────────────────────────────────

export type Stage = { step: string; label: string; start: number };

// `start` is in real seconds into the loop, so a click can seek straight to it.
const at = (sceneStart: number) => (sceneStart + INTRO) * SLOW;

export const STAGES: Stage[] = [
  { step: "1", label: "hikma init", start: 0 },
  { step: "2", label: "skill install", start: at(2.8) },
  { step: "3", label: "skill create + push", start: at(7.9) },
  { step: "4", label: "hikma sync", start: at(14.3) },
];

export function stageAt(raw: number): number {
  const r = ((raw % LOOP) + LOOP) % LOOP;
  let active = 0;
  STAGES.forEach((st, i) => {
    if (r >= st.start) active = i;
  });
  return active;
}

// The scroll runway maps onto the story: each stage gets an equal share, and the end stops
// just before the loop fades out.
export const END = at(STORY - 0.5);
const stageEnd = (i: number) => (i + 1 < STAGES.length ? STAGES[i + 1].start : END);

// Real seconds into the loop -> 0..1 along the timeline.
export function progressAt(raw: number): number {
  const r = Math.min(END, ((raw % LOOP) + LOOP) % LOOP);
  const i = stageAt(r);
  return (i + seg(r, STAGES[i].start, stageEnd(i))) / STAGES.length;
}

// 0..1 along the timeline -> real seconds into the loop.
export function timeAtProgress(p: number): number {
  const q = clamp(p) * STAGES.length;
  const i = Math.min(STAGES.length - 1, Math.floor(q));
  return lerp(STAGES[i].start, stageEnd(i), q - i);
}

export type Title = { alpha: number; main: string; sub: string };

// The start screen at the top of the loop, and the "WORLD 2" card before stage 4.
export function titleAt(raw: number): Title {
  const t = sceneTime(raw);
  const card = cardAlpha(t);
  if (card > 0) return { alpha: card, main: "World 2", sub: "new machine" };
  const story = t + INTRO;
  return { alpha: story < INTRO ? 1 - seg(story, INTRO - 0.5, INTRO) : 0, main: "Hikma", sub: "press start" };
}

// A frame that reads well without motion: the pet tells its plan and the CLI shows create and push.
export const STILL_T = at(12.6);
