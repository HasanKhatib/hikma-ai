<script setup lang="ts">
import { ref } from "vue";

type Method = "brew" | "script" | "source";

const tabs: { id: Method; label: string; prompt: string; commands: string[] }[] = [
  {
    id: "brew",
    label: "Homebrew",
    prompt: "$",
    commands: ["brew install HasanKhatib/tap/hikma"],
  },
  {
    id: "script",
    label: "Script",
    prompt: "$",
    commands: [
      "curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash",
    ],
  },
  {
    id: "source",
    label: "Go",
    prompt: "$",
    commands: ["go install github.com/hasankhatib/hikma-ai/cmd/hikma@latest"],
  },
];

const hints: Record<Method, string> = {
  brew: "macOS and Linux",
  script: "macOS, Linux, Windows Git Bash",
  source: "any platform with Go",
};

const method = ref<Method>("brew");
const copied = ref(false);

function select(next: Method) {
  method.value = next;
  copied.value = false;
}

function handleTabKey(event: KeyboardEvent) {
  if (event.key !== "ArrowLeft" && event.key !== "ArrowRight") return;
  event.preventDefault();
  const index = tabs.findIndex((t) => t.id === method.value);
  const step = event.key === "ArrowRight" ? 1 : -1;
  select(tabs[(index + step + tabs.length) % tabs.length].id);
  const tablist = (event.currentTarget as HTMLElement).closest('[role="tablist"]');
  requestAnimationFrame(() =>
    tablist?.querySelector<HTMLElement>('[aria-selected="true"]')?.focus(),
  );
}

async function copyCommand() {
  const tab = tabs.find((t) => t.id === method.value)!;
  try {
    await navigator.clipboard.writeText(tab.commands.join("\n"));
    copied.value = true;
    window.setTimeout(() => (copied.value = false), 1800);
  } catch {
    copied.value = false;
  }
}
</script>

<template>
  <div class="hk-install-panel" aria-label="Install hikma">
    <div class="hk-install-bar">
      <div class="hk-install-tabs" role="tablist" aria-label="Choose how to install">
        <button
          v-for="tab in tabs"
          :id="`hk-tab-${tab.id}`"
          :key="tab.id"
          role="tab"
          :aria-selected="method === tab.id"
          aria-controls="hk-install-command"
          :tabindex="method === tab.id ? 0 : -1"
          @click="select(tab.id)"
          @keydown="handleTabKey"
        >
          {{ tab.label }}
        </button>
      </div>
      <button class="hk-copy-button" type="button" @click="copyCommand">
        {{ copied ? "Copied" : "Copy" }}
      </button>
    </div>

    <div
      id="hk-install-command"
      class="hk-install-command"
      role="tabpanel"
      :aria-labelledby="`hk-tab-${method}`"
    >
      <div class="hk-install-heading">
        <p class="hk-install-label">Install hikma</p>
        <p class="hk-install-hint">{{ hints[method] }}</p>
      </div>
      <div class="hk-install-lines">
        <div
          v-for="tab in tabs.filter((t) => t.id === method)"
          :key="tab.id"
          class="hk-install-line-group"
        >
          <div v-for="command in tab.commands" :key="command" class="hk-install-line">
            <span>{{ tab.prompt }}</span>
            <code>{{ command }}</code>
          </div>
        </div>
      </div>
    </div>

    <div class="hk-install-next">
      <span><i></i> Then run</span>
      <code>hikma doctor</code>
    </div>
  </div>
</template>
