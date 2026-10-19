---
layout: home
footer: false
---

<script setup>
import { withBase } from "vitepress";
</script>

<div class="hk-hero-wrap">
  <HeroScene />
  <div class="hk-dock">
    <div class="hk-headline">
      <h1>CLI buddy to help you <em>manage your skills.</em></h1>
      <p class="hk-hero-tagline">Setup your GitHub registry, push skills to it from wherever, create skills from templates, and more around it.</p>
    </div>
    <div class="hk-getit">
      <InstallLine />
      <div class="hk-hero-actions">
        <a class="hk-button hk-button-brand" :href="withBase('/getting-started')">Get started <span aria-hidden="true">&#8594;</span></a>
        <a class="hk-text-link" :href="withBase('/commands/')">Commands</a>
      </div>
    </div>
  </div>
</div>
