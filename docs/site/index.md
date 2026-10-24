---
layout: home
footer: false
---

<script setup>
import { withBase } from "vitepress";
</script>

<div class="hk-runway" aria-hidden="true"></div>
<div class="hk-hero-wrap">
  <HeroScene />
  <div class="hk-headline">
    <p class="hk-prompt" aria-hidden="true">~/hikma <span>$</span> <i>&#9612;</i></p>
    <h1>Your AI <em>skills follow you.</em></h1>
    <p class="hk-hero-tagline">Publish them to a GitHub repo you own. Install them into your preferred harness.</p>
  </div>
  <div class="hk-dock">
    <div class="hk-dock-copy">
      <div class="hk-hero-actions">
        <a class="hk-button hk-button-brand" :href="withBase('/getting-started')">Get started <span aria-hidden="true">&#8594;</span></a>
        <a class="hk-text-link" :href="withBase('/commands/')">Commands</a>
      </div>
    </div>
    <div class="hk-getit">
      <InstallLine />
    </div>
  </div>
</div>
