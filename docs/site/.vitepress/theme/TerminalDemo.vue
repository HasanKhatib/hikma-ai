<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";

type Step = { kind: "cmd" | "out"; text: string };
type Scene = { title: string; sub: string; steps: Step[] };

// Output is captured from the real CLI. Registry and source names are shortened
// for the page, and the push confirmation is taken from the skill push docs.
const scenes: Scene[] = [
  {
    title: "Found a skill you like?",
    sub: "Install it from any repo.",
    steps: [
      { kind: "cmd", text: "hikma skill install anthropics/skills pdf" },
      { kind: "out", text: "Agents: claude" },
      { kind: "out", text: "Source: anthropics/skills (argument)" },
      { kind: "out", text: "Commit: b1c99c7" },
      { kind: "out", text: "  -> .claude/skills/pdf/ (1 files)" },
    ],
  },
  {
    title: "Got an idea?",
    sub: "Create it, then publish it to your registry.",
    steps: [
      { kind: "cmd", text: "hikma skill create bs-detector" },
      { kind: "out", text: "Created .claude/skills/bs-detector/" },
      { kind: "out", text: "  SKILL.md" },
      { kind: "cmd", text: "hikma skill push bs-detector" },
      { kind: "out", text: "You are pushing to registry: myghaccount/myrepo" },
      { kind: "out", text: "  skill: bs-detector" },
      { kind: "out", text: "? Push to myghaccount/myrepo? (y/N)" },
    ],
  },
  {
    title: "New laptop?",
    sub: "Same skills, from the lockfile.",
    steps: [
      { kind: "cmd", text: "hikma sync" },
      { kind: "out", text: "Skills recorded in .hikma/lock.json: 1" },
      { kind: "out", text: "  restore   .claude/skills/pdf" },
      { kind: "out", text: "  restored .claude/skills/pdf" },
      { kind: "out", text: "Done: 1 restored, 0 up to date" },
    ],
  },
];

const active = ref(0);
const shown = ref<{ kind: Step["kind"]; text: string }[]>([]);
const playing = ref(true);
const reduced = ref(false);
const root = ref<HTMLElement | null>(null);

let timer: number | undefined;
let visible = false;
let run = 0;

const wait = (ms: number) =>
  new Promise<void>((resolve) => {
    timer = window.setTimeout(resolve, ms);
  });

async function play(token: number) {
  while (token === run) {
    for (let i = 0; i < scenes.length; i++) {
      if (token !== run) return;
      active.value = i;
      shown.value = [];
      for (const step of scenes[i].steps) {
        if (token !== run) return;
        if (step.kind === "cmd") {
          shown.value.push({ kind: "cmd", text: "" });
          const line = shown.value[shown.value.length - 1];
          for (const ch of step.text) {
            if (token !== run) return;
            line.text += ch;
            await wait(24);
          }
          await wait(380);
        } else {
          shown.value.push({ kind: "out", text: step.text });
          await wait(170);
        }
      }
      await wait(2300);
    }
  }
}

function start() {
  run++;
  if (reduced.value || !playing.value || !visible) return;
  void play(run);
}

function stop() {
  run++;
  window.clearTimeout(timer);
}

function toggle() {
  playing.value = !playing.value;
  if (playing.value) start();
  else stop();
}

function pick(i: number) {
  stop();
  playing.value = false;
  active.value = i;
  shown.value = scenes[i].steps.map((s) => ({ ...s }));
}

let observer: IntersectionObserver | undefined;

onMounted(() => {
  reduced.value = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  if (reduced.value) {
    playing.value = false;
    pick(0);
    return;
  }
  observer = new IntersectionObserver(([entry]) => {
    visible = entry.isIntersecting;
    if (visible && playing.value) start();
    else stop();
  });
  if (root.value) observer.observe(root.value);
});

onBeforeUnmount(() => {
  stop();
  observer?.disconnect();
});
</script>

<template>
  <section ref="root" class="hk-demo" aria-label="Hikma in use">
    <ol class="hk-demo-index">
      <li v-for="(scene, i) in scenes" :key="scene.title" :class="{ on: active === i }">
        <button type="button" :aria-current="active === i" @click="pick(i)">
          <strong>{{ scene.title }}</strong>
          <span>{{ scene.sub }}</span>
        </button>
      </li>
    </ol>
    <div class="hk-demo-term">
      <div class="hk-demo-bar">
        <span class="hk-demo-dots" aria-hidden="true"><i></i><i></i><i></i></span>
        <button v-if="!reduced" type="button" class="hk-demo-toggle" @click="toggle">
          {{ playing ? "Pause" : "Play" }}
        </button>
      </div>
      <pre class="hk-demo-screen" role="img" :aria-label="scenes[active].title + ' ' + scenes[active].sub"><code><template v-for="(line, i) in shown" :key="i"><span :class="line.kind"><template v-if="line.kind === 'cmd'"><b>$ </b></template>{{ line.text }}</span>
</template></code></pre>
    </div>
  </section>
</template>
