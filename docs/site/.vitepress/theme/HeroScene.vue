<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { H, LOOP, STILL_T, TRANSCRIPT, W, balloonAt, cliAt, drawScene } from "./heroScene";
import type { Balloon, CliLine } from "./heroScene";

const canvas = ref<HTMLCanvasElement | null>(null);
const root = ref<HTMLElement | null>(null);
const balloon = ref<Balloon | null>(null);
const cli = ref<CliLine[]>([]);
const playing = ref(true);
const reduced = ref(false);

let raf = 0;
let visible = false;
let clock = 0; // seconds into the loop
let last = 0;
let ctx: CanvasRenderingContext2D | null = null;

function paint(t: number) {
  if (!ctx) return;
  drawScene(ctx, t);
  const b = balloonAt(t % LOOP);
  balloon.value = b && balloon.value?.text === b.text && Math.abs(balloon.value.x - b.x) < 0.2 && Math.abs(balloon.value.y - b.y) < 0.2 ? balloon.value : b;
  const lines = cliAt(t % LOOP);
  const prev = cli.value;
  if (lines.length !== prev.length || lines.some((l, i) => l.text !== prev[i].text)) cli.value = lines;
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

let observer: IntersectionObserver | undefined;
const onVisibility = () => run();

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
  <figure ref="root" class="hk-scene" role="group" aria-label="Hikma in a pixel-art loop">
    <canvas ref="canvas" :width="W" :height="H" role="img" :aria-label="TRANSCRIPT"></canvas>
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
</template>
