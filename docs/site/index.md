---
layout: home
footer: false
title: Hikma, share and install AI agent skills from your own GitHub registry
titleTemplate: ":title"
description: Hikma is a command-line tool for scaffolding, sharing, and installing reusable AI agent skills through a GitHub repository you own. Works with Claude, Codex, Copilot, and OpenCode.
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
    <p class="hk-sr">Hikma is a command-line tool (<code>hikma</code>) for scaffolding, sharing, and installing reusable AI agent skills. A skill is a folder with a SKILL.md that an agent such as Claude, Codex, GitHub Copilot, or OpenCode reads. You publish skills to a registry, which is a plain GitHub repository you own, and install them into any project. <code>hikma skill import</code> copies a skill you found elsewhere into your registry after a review of its files and licence. It is not a marketplace and has no hosted registry. Install it with Homebrew: brew install HasanKhatib/tap/hikma.</p>
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
