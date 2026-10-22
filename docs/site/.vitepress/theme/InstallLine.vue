<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { withBase } from "vitepress";

type Os = "mac" | "linux" | "windows";

const SCRIPT = "curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash";
const INSTALL: Record<Os, { name: string; how: string; command: string }> = {
  mac: { name: "macOS", how: "Homebrew", command: "brew install HasanKhatib/tap/hikma" },
  linux: { name: "Linux", how: "install script", command: SCRIPT },
  windows: { name: "Windows", how: "install script, run in Git Bash", command: SCRIPT },
};

// The first paint (and the server render) assume macOS; the real system is read once the page is in a browser.
// ?os=linux or ?os=windows shows another system, for docs and screenshots.
const os = ref<Os>("mac");
const copied = ref(false);
const pick = computed(() => INSTALL[os.value]);

function detect(): Os {
  const asked = new URLSearchParams(window.location.search).get("os");
  if (asked === "mac" || asked === "linux" || asked === "windows") return asked;
  const ua = navigator.userAgent;
  if (/Windows/i.test(ua)) return "windows";
  if (/Android/i.test(ua)) return "mac"; // a phone is not where it gets installed; show the common one
  if (/Linux|X11|CrOS/i.test(ua)) return "linux";
  return "mac";
}

onMounted(() => {
  os.value = detect();
});

async function copy() {
  try {
    await navigator.clipboard.writeText(pick.value.command);
    copied.value = true;
    window.setTimeout(() => (copied.value = false), 1800);
  } catch {
    copied.value = false;
  }
}
</script>

<template>
  <p class="hk-install-os">
    <span aria-hidden="true">&#9679;</span> For {{ pick.name }}: {{ pick.how }}
  </p>
  <div class="hk-install-line" aria-label="Install hikma">
    <code :title="pick.command"><span aria-hidden="true">$</span> {{ pick.command }}</code>
    <button type="button" class="hk-copy-button" aria-live="polite" @click="copy">
      {{ copied ? "Copied ✓" : "Copy" }}
    </button>
  </div>
  <p class="hk-install-foot">
    <a :href="withBase('/getting-started')">Other ways to install</a>
    <a class="hk-built" href="https://alkhatib.tech" target="_blank" rel="noopener">Developed by <b>Hasan Alkhatib</b></a>
  </p>
</template>
