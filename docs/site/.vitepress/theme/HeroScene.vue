<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useData } from "vitepress";
import { H, LOOP, STAGES, STILL_T, TRANSCRIPT, W, balloonAt, cliAt, drawScene, stageAt, titleAlpha } from "./heroScene";
import type { Balloon, CliLine } from "./heroScene";

const { isDark } = useData();

const canvas = ref<HTMLCanvasElement | null>(null);
const root = ref<HTMLElement | null>(null);
const balloon = ref<Balloon | null>(null);
const cli = ref<CliLine[]>([]);
const stage = ref(0);
const title = ref(1);
const playing = ref(true);
const reduced = ref(false);

let raf = 0;
let visible = false;
let clock = 0; // real seconds into the loop
let last = 0;
let ctx: CanvasRenderingContext2D | null = null;

function paint(t: number) {
  if (!ctx) return;
  drawScene(ctx, t, isDark.value);
  const b = balloonAt(t);
  const prev = balloon.value;
  balloon.value = b && prev && prev.text === b.text && Math.abs(prev.x - b.x) < 0.2 && Math.abs(prev.y - b.y) < 0.2 ? prev : b;
  const lines = cliAt(t);
  const old = cli.value;
  if (lines.length !== old.length || lines.some((l, i) => l.text !== old[i].text)) cli.value = lines;
  stage.value = stageAt(t);
  const ta = Math.round(titleAlpha(t) * 20) / 20;
  if (ta !== title.value) title.value = ta;
}

function frame(now: number) {
  raf = requestAnimationFrame(frame);
  if (!last) last = now;
  clock = (clock + Math.min(0.1, (now - last) / 1000)) % LOOP;
  last = now;
  paint(clock);
}

function run() {
  cancelAnimationFrame(raf);
  last = 0;
  if (playing.value && visible && !reduced.value && !document.hidden) raf = requestAnimationFrame(frame);
}

function toggle() {
  playing.value = !playing.value;
  run();
}

// Jump to the start of a stage. The scene is a function of time, so the pet,
// the skills, and the commands are right for wherever we land.
function seek(i: number) {
  clock = STAGES[i].start + 0.01;
  last = 0;
  paint(clock);
}

let observer: IntersectionObserver | undefined;
const onVisibility = () => run();

watch(isDark, () => paint(clock));

onMounted(() => {
  ctx = canvas.value!.getContext("2d");
  reduced.value = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
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
    return;
  }
  paint(0);
  observer = new IntersectionObserver(([entry]) => {
    visible = entry.isIntersecting;
    run();
  });
  if (root.value) observer.observe(root.value);
  document.addEventListener("visibilitychange", onVisibility);
});

onBeforeUnmount(() => {
  cancelAnimationFrame(raf);
  observer?.disconnect();
  document.removeEventListener("visibilitychange", onVisibility);
});
</script>

<template>
  <div class="hk-scene-wrap">
    <figure ref="root" class="hk-scene" role="group" aria-label="Hikma in a pixel-art loop">
      <canvas ref="canvas" :width="W" :height="H" role="img" :aria-label="TRANSCRIPT"></canvas>
      <div v-if="title > 0" class="hk-scene-title" :style="{ opacity: title }" aria-hidden="true">
        <strong>Hikma</strong>
        <span>press start</span>
      </div>
      <div
        v-if="balloon"
        class="hk-scene-balloon"
        :class="balloon.who"
        :style="{ left: balloon.x + '%', top: balloon.y + '%' }"
        aria-hidden="true"
      >
        {{ balloon.text }}
      </div>
      <div v-if="cli.length" class="hk-scene-cli" aria-hidden="true">
        <div v-for="(line, i) in cli" :key="i" :class="line.out ? 'out' : 'cmd'">
          <template v-if="!line.out"><b>$ </b></template>{{ line.text }}
        </div>
      </div>
      <button v-if="!reduced" type="button" class="hk-scene-toggle" @click="toggle">
        {{ playing ? "Pause" : "Play" }}
      </button>
    </figure>
    <nav class="hk-stages" aria-label="Jump to a step">
      <button
        v-for="(st, i) in STAGES"
        :key="st.step"
        type="button"
        :class="{ on: stage === i, done: stage > i }"
        :aria-current="stage === i ? 'step' : undefined"
        @click="seek(i)"
      >
        <i>{{ st.step }}</i>
        <code>{{ st.label }}</code>
      </button>
    </nav>
  </div>
</template>
