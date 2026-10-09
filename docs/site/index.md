---
layout: home

hero:
  name: Hikma
  text: Skills for your AI agents, from a registry you own
  tagline: Set up a repository for Claude, Codex, Copilot, or OpenCode in one command, and install, update, and publish reusable skills through plain Git repositories.
  actions:
    - theme: brand
      text: Get started
      link: /getting-started
    - theme: alt
      text: Command reference
      link: /commands/
    - theme: alt
      text: View on GitHub
      link: https://github.com/HasanKhatib/hikma-ai

features:
  - title: One command sets up a repo
    details: "`hikma init` writes each agent's instruction files, saves the agents and registry in a committed config, and installs skills into every agent's folder."
  - title: Your registry is a setting
    details: Bare names install from your registry. `hikma skill push` opens a pull request into it, through your fork when you lack write access, after you confirm the target.
  - title: Installs you can trust
    details: A lockfile records the source, commit, and file hashes. Updates stop on local edits and flag changes to scripts.
  - title: Policy for your registry
    details: Choose a naming rule and run `hikma registry validate` to check every skill against the format.
  - title: Plain Git, light requirements
    details: Reading skills needs only `git`. Registries are ordinary repositories; there is nothing to host.
  - title: Works alongside gh skill
    details: Both tools read the same `skills/<name>/SKILL.md` layout, so a repository that works with one works with the other.
---

## Install

::: code-group

```bash [Homebrew]
brew install HasanKhatib/tap/hikma
```

```bash [Script (macOS, Linux, Windows Git Bash)]
curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash
```

```bash [From source]
go install github.com/hasankhatib/hikma-ai/cmd/hikma@latest
```

:::

Then follow [Getting started](/getting-started).
