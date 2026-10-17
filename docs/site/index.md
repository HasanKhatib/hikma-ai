---
layout: home
footer: false
---

<script setup>
import { withBase } from "vitepress";
</script>

<div class="hk-hero-wrap">
<div class="hk-hero">
  <div class="hk-hero-copy">
    <p class="hk-eyebrow"><bdi lang="ar" dir="rtl">حكمة</bdi> Hikma</p>
    <h1>Skills follow you. <em>Define your source, use one CLI.</em></h1>
    <p class="hk-hero-tagline">Publish to a registry you own, install into Claude, Codex, Copilot, OpenCode, or your own agent. Plain Git.</p>
    <div class="hk-hero-actions">
      <a class="hk-button hk-button-brand" :href="withBase('/getting-started')">Get started <span aria-hidden="true">&#8594;</span></a>
      <a class="hk-text-link" :href="withBase('/commands/')">Commands</a>
    </div>
    <InstallLine />
  </div>
  <div class="hk-hero-image">
    <HeroScene />
    <p class="hk-flow"><code>hikma init</code><i aria-hidden="true">&#8594;</i><code>skill install</code><i aria-hidden="true">&#8594;</i><code>skill push</code></p>
  </div>
</div>
</div>
