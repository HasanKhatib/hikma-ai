<script setup lang="ts">
import { ref } from "vue";
import { withBase } from "vitepress";

const command = "brew install HasanKhatib/tap/hikma";
const copied = ref(false);

async function copy() {
  try {
    await navigator.clipboard.writeText(command);
    copied.value = true;
    window.setTimeout(() => (copied.value = false), 1800);
  } catch {
    copied.value = false;
  }
}
</script>

<template>
  <div class="hk-install-line" aria-label="Install hikma">
    <code><span aria-hidden="true">$</span> {{ command }}</code>
    <button type="button" class="hk-copy-button" aria-live="polite" @click="copy">
      {{ copied ? "Copied ✓" : "Copy" }}
    </button>
  </div>
  <p class="hk-install-alt">
    Other ways:
    <a href="https://github.com/HasanKhatib/hikma-ai/blob/main/scripts/install.sh" target="_blank" rel="noopener">install script</a>
    (read it first), <code>go install</code>, or
    <a :href="withBase('/getting-started')">all options</a>.
  </p>
</template>
