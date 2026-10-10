---
layout: home
---

<script setup>
import { withBase } from "vitepress";
</script>

<div class="hk-hero-wrap">
<div class="hk-hero">
  <div class="hk-hero-copy">
    <p class="hk-eyebrow"><bdi lang="ar" dir="rtl">حكمة</bdi> Skills for AI agents</p>
    <h1>Skills for your agents, <em>from a registry you own.</em></h1>
    <p class="hk-hero-tagline">Set up a repository for Claude, Codex, Copilot, or OpenCode in one command. Install, update, and publish reusable skills through plain Git repositories.</p>
    <div class="hk-hero-actions">
      <a class="hk-button hk-button-brand" :href="withBase('/getting-started')">Get started <span aria-hidden="true">&#8599;</span></a>
      <a class="hk-button hk-button-alt" :href="withBase('/commands/')">Command reference</a>
    </div>
    <p class="hk-hero-meta"><span>macOS, Linux, Windows</span><span>Only needs git</span><span>MIT licensed</span></p>
  </div>
  <div class="hk-hero-image">
    <InstallPanel />
  </div>
</div>
</div>

<div class="hk-workflow-wrap hk-full-bleed">
<div class="hk-workflow">
  <div class="hk-workflow-heading">
    <p class="hk-eyebrow">The hikma workflow</p>
    <h2>Set up the repo. Install the skill.<br />Publish what works.</h2>
  </div>

  <div class="hk-workflow-grid">
    <a class="hk-workflow-step" :href="withBase('/commands/init')">
      <div class="hk-workflow-topline">
        <span class="hk-workflow-number">01 / SET UP</span>
        <span class="hk-workflow-arrow" aria-hidden="true">&#8599;</span>
      </div>
      <div class="hk-workflow-node"><span>AGENTS.md</span></div>
      <h3>One command per repository.</h3>
      <p>Write the agent instructions, save the agents and registry in a committed config, and install skills into every agent's folder.</p>
      <code>hikma init</code>
    </a>
    <a class="hk-workflow-step" :href="withBase('/commands/skill-install')">
      <div class="hk-workflow-topline">
        <span class="hk-workflow-number">02 / INSTALL</span>
        <span class="hk-workflow-arrow" aria-hidden="true">&#8599;</span>
      </div>
      <div class="hk-workflow-node"><span>.hikma/lock.json</span></div>
      <h3>Installs you can trust.</h3>
      <p>A lockfile records the source, commit, and file hashes. Updates stop on local edits and flag changes to scripts.</p>
      <code>hikma skill install</code>
    </a>
    <a class="hk-workflow-step" :href="withBase('/commands/skill-push')">
      <div class="hk-workflow-topline">
        <span class="hk-workflow-number">03 / PUBLISH</span>
        <span class="hk-workflow-arrow" aria-hidden="true">&#8599;</span>
      </div>
      <div class="hk-workflow-node"><span>registry / pull request</span></div>
      <h3>Share it by pull request.</h3>
      <p>Confirm the target registry, push a branch, and open a pull request. Through your fork when you lack write access.</p>
      <code>hikma skill push</code>
    </a>
  </div>

  <div class="hk-guide-row">
    <div class="hk-guide-intro">
      <span>GO DEEPER</span>
      <p>Follow a guide or look up one command.</p>
    </div>
    <a class="hk-guide-link" :href="withBase('/guides/agents')">
      <span>GUIDE 01</span>
      <strong>Agents and folders</strong>
      <i aria-hidden="true">&#8599;</i>
    </a>
    <a class="hk-guide-link" :href="withBase('/guides/registry')">
      <span>GUIDE 02</span>
      <strong>Run a registry</strong>
      <i aria-hidden="true">&#8599;</i>
    </a>
    <a class="hk-guide-link" :href="withBase('/guides/gh-skill')">
      <span>GUIDE 03</span>
      <strong>Use with gh skill</strong>
      <i aria-hidden="true">&#8599;</i>
    </a>
    <a class="hk-guide-link" :href="withBase('/commands/')">
      <span>REFERENCE</span>
      <strong>Browse commands</strong>
      <i aria-hidden="true">&#8599;</i>
    </a>
  </div>
</div>
</div>
