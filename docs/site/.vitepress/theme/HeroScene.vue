<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useData } from "vitepress";
import { H, LOOP, STAGES, STILL_T, TRANSCRIPT, W, bubbleAt, drawScene, progressAt, setView, skyAt, stageAt, timeAtProgress, titleAt } from "./heroScene";
import type { Bubble, Title } from "./heroScene";

const { isDark } = useData();

const canvas = ref<HTMLCanvasElement | null>(null);
const root = ref<HTMLElement | null>(null);
const box = ref<HTMLElement | null>(null);
const width = ref(W); // logical canvas width: the story's stage plus any extra world a wide window shows
const sky = ref("");
const bubble = ref<Bubble | null>(null);
const stage = ref(0);
const progress = ref(0);
const title = ref<Title>({ alpha: 1, main: "Hikma", sub: "press start" });
const playing = ref(true);
const reduced = ref(false);

let raf = 0;
let clock = 0; // real seconds into the loop
let last = 0;
let userAt = -1e9; // when the reader last scrolled; the story follows the scroll bar while they do
let lastY = 0; // where the page was last scrolled by the scene itself, to tell it from the reader
let ctx: CanvasRenderingContext2D | null = null;

function paint(t: number) {
  if (!ctx) return;
  drawScene(ctx, t, isDark.value);
  const color = skyAt(t, isDark.value);
  if (color !== sky.value) {
    sky.value = color;
    // The navbar borrows the sky, so it reads as part of the picture.
    document.documentElement.style.setProperty("--hk-sky", color);
  }
  const b = bubbleAt(t);
  const prev = bubble.value;
  const same =
    b && prev && prev.who === b.who && prev.quote === b.quote && Math.abs(prev.x - b.x) < 0.2 && Math.abs(prev.y - b.y) < 0.2 && prev.lines.length === b.lines.length;
  bubble.value = same && b!.lines.every((l, i) => l.text === prev!.lines[i].text) ? prev : b;
  stage.value = stageAt(t);
  progress.value = Math.round(progressAt(t) * 500) / 500;
  const ta = titleAt(t);
  ta.alpha = Math.round(ta.alpha * 20) / 20;
  const cur = title.value;
  if (ta.alpha !== cur.alpha || ta.main !== cur.main) title.value = ta;
}

// The page scrolls as the story plays, and the story follows when the reader scrolls.
const range = () => Math.max(1, document.documentElement.scrollHeight - window.innerHeight);

function syncScroll() {
  lastY = Math.round(progressAt(clock) * range());
  window.scrollTo({ top: lastY, behavior: "instant" });
}

const SCROLL_IDLE = 250; // ms after the last scroll before the story plays on by itself again

// The reader scrolled: the story jumps to that point. If it was playing it keeps playing from there;
// only the Pause button stops it.
function onScroll() {
  if (Math.abs(window.scrollY - lastY) < 3) return;
  userAt = performance.now();
  lastY = window.scrollY;
  clock = timeAtProgress(window.scrollY / range());
  paint(clock);
  if (playing.value) run();
}

function frame(now: number) {
  raf = requestAnimationFrame(frame);
  if (!last) last = now;
  const dt = Math.min(0.1, (now - last) / 1000);
  last = now;
  // A scroll that has not raised its event yet still counts: it wins over the story moving the page.
  if (Math.abs(window.scrollY - lastY) >= 3) userAt = now;
  if (now - userAt < SCROLL_IDLE) {
    lastY = window.scrollY;
    clock = timeAtProgress(window.scrollY / range());
    paint(clock);
    return;
  }
  clock = (clock + dt) % LOOP;
  paint(clock);
  syncScroll();
}

function run() {
  cancelAnimationFrame(raf);
  last = 0;
  if (playing.value && !reduced.value && !document.hidden) raf = requestAnimationFrame(frame);
}

function toggle() {
  playing.value = !playing.value;
  if (playing.value) syncScroll();
  run();
}

// Jump to the start of a stage. The scene is a function of time, so the pet,
// the skills, and the commands are right for wherever we land.
function seek(i: number) {
  clock = STAGES[i].start + 0.01;
  last = 0;
  paint(clock);
  syncScroll();
}

// Up and down step through the stages while the timeline has focus.
function onRailKey(e: KeyboardEvent) {
  const dir = e.key === "ArrowDown" || e.key === "PageDown" ? 1 : e.key === "ArrowUp" || e.key === "PageUp" ? -1 : 0;
  if (!dir) return;
  e.preventDefault();
  seek(Math.min(STAGES.length - 1, Math.max(0, stage.value + dir)));
}

let resizer: ResizeObserver | undefined;

// A wide, short window gets a wider world; the canvas keeps square pixels at any size.
function fit() {
  const el = box.value;
  if (!el || !el.clientHeight) return;
  const next = setView((H * el.clientWidth) / el.clientHeight);
  if (next === width.value) return;
  width.value = next;
  // Changing the canvas size clears it, so repaint once Vue has applied the new width.
  nextTick(() => paint(clock));
}
const onVisibility = () => run();

watch(isDark, () => paint(clock));

onMounted(() => {
  ctx = canvas.value!.getContext("2d");
  resizer = new ResizeObserver(fit);
  resizer.observe(box.value!);
  fit();
  reduced.value = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  window.history.scrollRestoration = "manual";
  window.scrollTo(0, 0);
  window.addEventListener("scroll", onScroll, { passive: true });
  // ?t=12.6 freezes the scene at that second, for stills and video frames.
  const frozen = Number(new URLSearchParams(window.location.search).get("t"));
  if (frozen > 0) {
    playing.value = false;
    reduced.value = true;
    clock = frozen;
    paint(clock);
    return;
  }
  if (reduced.value) {
    playing.value = false;
    clock = STILL_T;
    paint(clock);
    syncScroll();
    return;
  }
  paint(0);
  document.addEventListener("visibilitychange", onVisibility);
  run();
});

onBeforeUnmount(() => {
  cancelAnimationFrame(raf);
  resizer?.disconnect();
  window.removeEventListener("scroll", onScroll);
  document.documentElement.style.removeProperty("--hk-sky");
  document.removeEventListener("visibilitychange", onVisibility);
});
</script>

<template>
  <div ref="root" class="hk-scene-wrap">
    <figure class="hk-scene" :style="{ background: sky }" role="group" aria-label="Hikma in a pixel-art loop">
      <div ref="box" class="hk-box">
        <canvas ref="canvas" :width="width" :height="H" role="img" :aria-label="TRANSCRIPT"></canvas>
        <div v-if="title.alpha > 0" class="hk-scene-title" :style="{ opacity: title.alpha }" aria-hidden="true">
          <strong>{{ title.main }}</strong>
          <span>{{ title.sub }}</span>
        </div>
        <div
          v-if="bubble"
          class="hk-bubble"
          :class="bubble.who"
          :style="{ '--x': bubble.x + '%', top: bubble.y + '%' }"
          aria-hidden="true"
        >
          <p v-if="bubble.quote" class="hk-quote">{{ bubble.quote }}</p>
          <div v-if="bubble.lines.length" class="hk-term">
            <div v-for="(line, i) in bubble.lines" :key="i" :class="line.out ? 'out' : 'cmd'">
              <template v-if="!line.out"><b>$ </b></template>{{ line.text }}<span v-if="!line.out && i === bubble.lines.length - 1" class="hk-caret">&#9612;</span>
            </div>
          </div>
        </div>
      </div>
    </figure>
    <nav class="hk-rail" aria-label="Steps" :style="{ '--p': progress }" @keydown="onRailKey">
      <div class="hk-rail-track">
        <ol>
          <li v-for="(st, i) in STAGES" :key="st.step">
            <button
              type="button"
              :class="{ on: stage === i, done: stage > i }"
              :aria-current="stage === i ? 'step' : undefined"
              :aria-label="`Step ${st.step}: ${st.label}`"
              @click="seek(i)"
            >
              <i>{{ st.step }}</i>
              <code>{{ st.label }}</code>
            </button>
          </li>
        </ol>
        <span class="hk-rail-line" aria-hidden="true"><span></span></span>
      </div>
      <button
        v-if="!reduced"
        type="button"
        class="hk-scene-toggle"
        :class="{ paused: !playing }"
        :aria-label="playing ? 'Pause the animation' : 'Play the animation'"
        :title="playing ? 'Pause the animation' : 'Play the animation'"
        @click="toggle"
      >
        <svg v-if="playing" viewBox="0 0 12 12" width="12" height="12" aria-hidden="true"><path fill="currentColor" d="M2 1h3v10H2zM7 1h3v10H7z" /></svg>
        <svg v-else viewBox="0 0 12 12" width="12" height="12" aria-hidden="true"><path fill="currentColor" d="M2 1h2v10H2zM4 2h2v8H4zM6 3h2v6H6zM8 4h2v4H8z" /></svg>
        <span>{{ playing ? "Pause" : "Play" }}</span>
      </button>
    </nav>
  </div>
</template>
