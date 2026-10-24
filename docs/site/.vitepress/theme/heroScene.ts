// The home page scene: a pixel-art loop that tells the Hikma story.
// Everything is a pure function of time (scene seconds into the story, played slower),
// so the same code can draw a live canvas or render frames for a video.

export const W = 240; // the stage the story is drawn on; wider windows get more world either side of it
export const MAX_VW = 560;
export const H = 135;
export const GROUND = 92;
export const STORY = 29.8; // length of the story, in scene seconds
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
const CHEST_X = 206; // the registry chest, in the right corner
const CHEST_AT = 3; // it appears when init sets the registry
const STRANGER_X = 142; // where the other character stands while the skill is imported
const HERO_AT = 90; // where the hero meets them
const HERO_AT2 = 110; // and where the hero waits for the install

// The story, in scene seconds: init, import, install, then a second world for sync.
const ARRIVE_A = 5.2; // the stranger walks in from the right
const ARRIVE_B = 8;
const COPY_A = 12.2; // a copy of their skill flies into the chest
const COPY_B = 13.6;
const LEAVE_A = 14.2; // and they walk off to the right, keeping their own
const LEAVE_B = 18.4;
const OPEN_A = 16.6; // the chest opens and hands the skill to the pet
const OPEN_B = 18;
const LEARN = 17.8; // the pet has it

// Stage 4 happens in a second world: a "WORLD 2" card, then a lava land where the pet starts with no skills.
const W2 = 20; // the world changes here, hidden under the card
const W2_WALK = 21.4; // the hero starts walking toward the hole
const FALL_A = 23; // the hero walks off the edge of the lava hole
const FALL_B = 23.7;
const RESPAWN = 24.5; // back at the checkpoint flag
const SYNC_A = 26.3; // the skill is restored
const JUMP_A = 27.2;
const JUMP_B = 28.8;

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
  g: "#4aa8ff", // the skill, blue like the bot it turns blue
  q: "#e6f3ff", // gem glint
  d: "#2468b3", // gem shade
  m: "#c4622f", // chest
  s: "#e8c27a", // chest trim
  p: "#d94f9a", // the other character's beanie
  y: "#f2b84b", // beanie band
  K: "#a8693f", // their skin
  j: "#3f5870", // their jacket
  z: "#9fb4c8", // jacket zip
  L: "#2a2a2a", // boots
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

// The pet is dark with no skills and turns blue with one.
const PET_COLORS = ["#44504e", "#4aa8ff"];

// The other character: a beanie, a darker skin tone, and a slate jacket, so they read apart from the hero.
const STRANGER = [
  "..pppppp..",
  ".pppppppp.",
  ".pyyyyyyp.",
  "..KKKKKK..",
  "..KeKeKK..",
  "..KKKKKK..",
  "..jjjjjj..",
  ".jjjjzjjj.",
  "jjjjjzjjjj",
  ".jjjjzjjj.",
  ".jjjjzjjj.",
  ".jjjjjjjj.",
  "..jjjjjj..",
];
const STRANGER_LEGS_A = ["...L..L..."];
const STRANGER_LEGS_B = ["..L....L.."];

const GEM = ["..g..", ".gqg.", "ggggg", ".gdg.", "..g.."];
const CHEST = ["..mmmmmmmmmm..", ".mssssssssssm.", ".mmmmmsmmmmmm.", ".mmmmmmmmmmmm.", ".mmmmmmmmmmmm.", ".mmmmmmmmmmmm."];
// Open: the lid stays up and light spills out through the gap.
const CHEST_OPEN = ["..mmmmmmmmmm..", ".mssssssssssm.", ".qgqgqgqgqgqg.", ".mmmmmsmmmmmm.", ".mmmmmmmmmmmm.", ".mmmmmmmmmmmm."];
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
    petBlink0: make(PET_BLINK, { b: PET_COLORS[0] }),
    petBlink1: make(PET_BLINK, { b: PET_COLORS[1] }),
    gem: make(GEM),
    chest: make(CHEST),
    chestOpen: make(CHEST_OPEN),
    strangerA: make([...STRANGER, ...STRANGER_LEGS_A], {}, edge),
    strangerB: make([...STRANGER, ...STRANGER_LEGS_B], {}, edge),
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
  if (t < ARRIVE_A) return 28;
  if (t < ARRIVE_B) return lerp(28, HERO_AT, seg(t, ARRIVE_A, ARRIVE_B));
  if (t < 15.6) return HERO_AT;
  if (t < 16.8) return lerp(HERO_AT, HERO_AT2, ease(seg(t, 15.6, 16.8)));
  if (t < W2) return HERO_AT2;
  if (t < W2_WALK) return 28;
  if (t < FALL_A) return lerp(28, 134, seg(t, W2_WALK, FALL_A));
  if (t < FALL_B) return lerp(134, 146, seg(t, FALL_A, FALL_B));
  if (t < RESPAWN) return 146;
  if (t < JUMP_A) return FLAG_X;
  if (t < JUMP_B) return lerp(FLAG_X, 178, ease(seg(t, JUMP_A, JUMP_B)));
  return lerp(178, 188, seg(t, JUMP_B, STORY));
}

// The other character comes in from beyond the right edge of whatever width the canvas has, and leaves the same way.
function strangerOff(): number {
  return (VW >= W ? W + OFF : W) + 30;
}

function strangerX(t: number): number {
  if (t < ARRIVE_B) return lerp(strangerOff(), STRANGER_X, seg(t, ARRIVE_A, ARRIVE_B));
  if (t < LEAVE_A) return STRANGER_X;
  return lerp(STRANGER_X, strangerOff(), seg(t, LEAVE_A, LEAVE_B));
}

function strangerVisible(t: number): boolean {
  return t >= ARRIVE_A && t < LEAVE_B && t < W2;
}

function strangerMoving(t: number): boolean {
  return (t > ARRIVE_A && t < ARRIVE_B) || (t > LEAVE_A && t < LEAVE_B);
}

// What the camera looks at: the character, then the middle of the import (where the other character and the chest are),
// and the checkpoint after the fall.
function camTarget(t: number): number {
  if (t >= FALL_B && t < RESPAWN + 0.6) return lerp(146, FLAG_X, ease(seg(t, FALL_B, RESPAWN + 0.6)));
  const look = seg(t, ARRIVE_A + 0.4, ARRIVE_A + 1.2) * (1 - seg(t, LEAVE_B, LEAVE_B + 0.6));
  return lerp(charX(t), 130, look);
}

function applyCamera(t: number) {
  SHIFT = VW >= W ? OFF : -clamp(camTarget(t) + 8 - VW * 0.42, 0, W - VW);
}

function charMoving(t: number): boolean {
  return (t > ARRIVE_A && t < ARRIVE_B) || (t > 15.6 && t < 16.8) || (t > W2_WALK && t < FALL_B) || t > JUMP_A;
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
  if (t >= FALL_A && t < FALL_A + 0.5) x = lerp(charX(FALL_A - 0.6) - 22, 138, seg(t, FALL_A, FALL_A + 0.5));
  else if (t >= FALL_A + 0.5 && t < FALL_A + 1.2) {
    const p = seg(t, FALL_A + 0.5, FALL_A + 1.2);
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
  return !(t >= FALL_A + 1.2 && t < FALL_A + 1.6);
}

// The pet is dark with no skills and blue with one. It loses it in the new world, and sync gives it back.
function skillCount(t: number): number {
  if (t < LEARN || (t >= W2 && t < SYNC_A)) return 0;
  return 1;
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

// A 3x5 pixel font, just the letters the registry name needs, for the tag on the chest.
const FONT: Record<string, string[]> = {
  m: ["101", "111", "111", "101", "101"],
  y: ["101", "101", "010", "010", "010"],
  g: ["111", "101", "111", "001", "111"],
  h: ["100", "100", "111", "101", "101"],
  a: ["010", "101", "111", "101", "101"],
  c: ["111", "100", "100", "100", "111"],
  o: ["111", "101", "101", "101", "111"],
  u: ["101", "101", "101", "101", "111"],
  n: ["110", "101", "101", "101", "101"],
  t: ["111", "010", "010", "010", "010"],
  r: ["110", "101", "110", "101", "101"],
  e: ["111", "100", "111", "100", "111"],
  p: ["111", "101", "111", "100", "100"],
  "/": ["001", "001", "010", "100", "100"],
};
const TAG = "myghaccount/myrepo";

function drawTag(ctx: CanvasRenderingContext2D, x: number, y: number, color: string) {
  let cx = x;
  for (const ch of TAG) {
    FONT[ch].forEach((row, j) =>
      [...row].forEach((bit, i) => {
        if (bit === "1") px(ctx, color, cx + i, y + j);
      }),
    );
    cx += 4;
  }
}

function chestIsOpen(t: number): boolean {
  return (t >= COPY_A + 0.3 && t < COPY_B + 0.5) || (t >= OPEN_A && t < OPEN_B);
}

function drawProps(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>, dark: boolean) {
  // signposts: the repo gets set up
  for (const [x, at] of [[46, 1.4], [62, 1.7]] as const) {
    const p = ease(seg(t, at, at + 0.4));
    if (p > 0 && t < W2) blit(ctx, s.sign, x, GROUND - 10 * p);
  }
  // checkpoint flag, in the lava world
  if (t >= W2_WALK && t < STORY) {
    px(ctx, "#f1ede2", FLAG_X - 14, GROUND - 22, 2, 22);
    px(ctx, "#ff9a6b", FLAG_X - 12, GROUND - 22, 9, 6);
  }
  // the registry chest appears when init sets the registry, with its name on a tag
  if (t >= CHEST_AT && t < W2) {
    const rise = ease(seg(t, CHEST_AT, CHEST_AT + 0.4));
    blit(ctx, chestIsOpen(t) ? s.chestOpen : s.chest, CHEST_X, GROUND - 12 * rise);
    if (rise === 1 && t >= CHEST_AT + 0.5) {
      const w = TAG.length * 4 + 2;
      const tx = Math.min(CHEST_X + 14 - w / 2, W - w - 2);
      px(ctx, dark ? "#0b1514" : "#fffaf0", tx, GROUND - 24, w, 9);
      px(ctx, dark ? "#2fa89f" : "#3b2f22", tx, GROUND - 24, w, 1);
      px(ctx, dark ? "#2fa89f" : "#3b2f22", tx, GROUND - 16, w, 1);
      drawTag(ctx, tx + 1, GROUND - 22, dark ? "#e8e6da" : "#14201e");
    }
    burst(ctx, CHEST_X + 14, GROUND - 10, CHEST_AT + 0.4, t, "#e8c27a");
  }
}

function drawGems(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  const pet = petPos(t);
  // the other character holds their skill up, and keeps it
  if (strangerVisible(t)) {
    const bob = Math.sin(t * 4) * 1.5;
    blit(ctx, s.gem, strangerX(t) + 5, GROUND - CH - 12 + bob);
  }
  // a copy of it flies into the chest
  if (t >= COPY_A && t < COPY_B) {
    const p = ease(seg(t, COPY_A, COPY_B));
    const fromX = STRANGER_X + 5;
    const fromY = GROUND - CH - 12;
    blit(ctx, s.gem, lerp(fromX, CHEST_X + 9, p), lerp(fromY, GROUND - 14, p) - Math.sin(Math.PI * p) * 36);
  }
  burst(ctx, CHEST_X + 14, GROUND - 10, COPY_B, t);
  // the chest hands the skill to the pet
  if (t >= OPEN_A + 0.2 && t < LEARN) {
    const p = ease(seg(t, OPEN_A + 0.2, LEARN));
    blit(ctx, s.gem, lerp(CHEST_X + 9, pet.x + 3, p), lerp(GROUND - 14, pet.y - 10, p) - Math.sin(Math.PI * p) * 30);
  }
  burst(ctx, pet.x + 8, pet.y + 4, LEARN, t);
  // the sync brings the skill back from above
  if (t >= SYNC_A - 0.6 && t < SYNC_A) {
    const p = ease(seg(t, SYNC_A - 0.6, SYNC_A));
    blit(ctx, s.gem, lerp(pet.x - 30, pet.x + 3, p), lerp(6, pet.y - 10, p));
  }
  burst(ctx, pet.x + 8, pet.y + 4, SYNC_A, t);
}

function drawPet(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  if (!petVisible(t)) return;
  const { x, y } = petPos(t);
  const blink = (((t % 3.2) + 3.2) % 3.2) < 0.12;
  const tone = skillCount(t);
  blit(ctx, s[(blink ? "petBlink" : "pet") + tone], x, y);
  // the skill the pet knows, shown as a tiny gem above its antenna
  if (tone) px(ctx, "#4aa8ff", x + 2, y - 8, 4, 4);
  if (t >= FALL_A + 1.6 && t < SYNC_A) {
    px(ctx, "#a9b0a6", x + 7, y - 6, 2, 2);
    px(ctx, "#a9b0a6", x + 7, y - 12, 2, 4);
  }
}

function drawStranger(ctx: CanvasRenderingContext2D, t: number, s: Record<string, Sprite>) {
  if (!strangerVisible(t)) return;
  blit(ctx, strangerMoving(t) && Math.floor(t * 6) % 2 ? s.strangerB : s.strangerA, strangerX(t), GROUND - CH);
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
  if (t < W2 - 0.7 || t >= W2 + 1.5) return 0;
  return t < W2 + 0.9 ? seg(t, W2 - 0.7, W2 - 0.1) : 1 - seg(t, W2 + 0.9, W2 + 1.5);
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
  drawProps(ctx, t, s, dark);
  drawStranger(ctx, t, s);
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

export type Who = "char" | "pet" | "other";
export type CliLine = { out: boolean; text: string };
// One speech bubble: what the speaker says, then the terminal lines that follow in the same bubble.
export type Bubble = { who: Who; quote: string | null; lines: CliLine[]; x: number; y: number };

const QUOTES: { a: number; b: number; who: Who; text: string }[] = [
  { a: 0.3, b: 3.0, who: "char", text: "New repo. Registry first." },
  { a: 5.4, b: 7.9, who: "char", text: "Someone has a great skill." },
  { a: 8.1, b: 9.4, who: "other", text: "Take a look." },
  { a: 9.5, b: 14.2, who: "char", text: "A copy for my registry, please." },
  { a: 14.3, b: 15.8, who: "other", text: "Enjoy! Bye!" },
  { a: 15.9, b: 16.9, who: "char", text: "Now, install it." },
  { a: 18.3, b: 19.3, who: "pet", text: "Learned it!" },
  { a: 22.1, b: 22.9, who: "char", text: "Hole!" },
  { a: 24.8, b: 26, who: "char", text: "No skills." },
  { a: 27.1, b: 28.2, who: "pet", text: "Watch this." },
];

type Cmd = { at: number; text: string; out?: boolean; group: number };

// Commands start about a second after the quote, so the speaker talks first and the terminal follows.
// The import review is from a real run of the built CLI; the source name is shortened for the page.
const CLI: Cmd[] = [
  { group: 1, at: 1.1, text: "hikma init --registry myghaccount/myrepo" },
  { group: 1, at: 2.7, text: "Wrote .hikma/config.json", out: true },
  { group: 1, at: 3.4, text: "myghaccount/myrepo: commits to main", out: true },
  { group: 2, at: 9.8, text: "hikma skill import someone/skills walking-above-holes" },
  { group: 2, at: 11.5, text: "files: 2, 153 B   licence: MIT", out: true },
  { group: 2, at: 12.0, text: "Import? y", out: true },
  { group: 2, at: 12.9, text: "Pushed to myghaccount/myrepo", out: true },
  { group: 3, at: 16.0, text: "hikma skill install walking-above-holes" },
  { group: 3, at: 17.4, text: "-> .claude/skills/walking-above-holes/", out: true },
  { group: 4, at: 25.3, text: "hikma sync" },
  { group: 4, at: 26.2, text: "restored .claude/skills/walking-above-holes", out: true },
  { group: 4, at: 26.5, text: "Done: 1 restored, 0 failed", out: true },
];

// [group, start, end, who speaks]
const GROUPS: [number, number, number, Who][] = [
  [1, 0.4, 5.0, "char"],
  [2, 9.5, 14.2, "char"],
  [3, 15.9, 18.2, "char"],
  [4, 24.8, 27.2, "char"],
];

function cliLines(t: number, group: number): CliLine[] {
  const lines: CliLine[] = [];
  for (const c of CLI) {
    if (c.group !== group || t < c.at) continue;
    if (c.out) lines.push({ out: true, text: c.text });
    else lines.push({ out: false, text: c.text.slice(0, Math.floor((t - c.at) / 0.03)) });
  }
  return lines.slice(-4);
}

export function bubbleAt(raw: number): Bubble | null {
  const t = sceneTime(raw);
  applyCamera(t);
  const quote = QUOTES.find((q) => t >= q.a && t < q.b);
  const group = GROUPS.find(([, a, b]) => t >= a && t < b);
  if (!quote && !group) return null;
  const who: Who = group ? group[3] : quote!.who;
  const lines = group ? cliLines(t, group[0]) : [];
  const p = who === "pet" ? petPos(t) : who === "other" ? { x: strangerX(t), y: GROUND - CH } : { x: charX(t), y: charY(t) };
  const x = ((p.x + 8 + SHIFT) / VW) * 100;
  const y = ((p.y - (who === "pet" ? 14 : 6)) / H) * 100;
  return { who, quote: quote ? quote.text : null, lines, x, y };
}

export const TRANSCRIPT =
  "A pixel-art loop. A developer and a small bot pet start a new repo with hikma init, and a chest appears: the registry, named myghaccount/myrepo. Another character in a beanie and jacket walks in holding a skill called walking-above-holes. The developer runs hikma skill import, a short review shows the files and the licence, and a copy of the skill flies into the chest while the other character keeps theirs and walks away. Then hikma skill install hands the skill to the bot, which turns blue. In World 2, a lava land on a new machine, the bot is dark again with no skills. They fall into a hole, respawn, and run hikma sync, which restores the skill, and the bot turns blue and jumps the hole.";
  "A pixel-art loop. A developer and a small bot pet start a new repo with hikma init, find a skill called go-blue and install it, which turns the bot blue, then create a new skill called walking-above-holes and push it to a registry. In World 2, a lava land on a new machine, the bot is dark again with no skills. They fall into a hole, respawn, and run hikma sync, which restores both skills, and the bot turns blue and then violet. Now they jump the hole.";

// ── Stages ───────────────────────────────────────────────────────────────────

export type Stage = { step: string; label: string; start: number };

// `start` is in real seconds into the loop, so a click can seek straight to it.
const at = (sceneStart: number) => (sceneStart + INTRO) * SLOW;

export const STAGES: Stage[] = [
  { step: "1", label: "hikma init", start: 0 },
  { step: "2", label: "skill import", start: at(4.9) },
  { step: "3", label: "skill install", start: at(15.5) },
  { step: "4", label: "hikma sync", start: at(19.2) },
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

// A frame that reads well without motion: the skill in flight from the other character to the chest, and the import review in the terminal.
export const STILL_T = at(12.7);
