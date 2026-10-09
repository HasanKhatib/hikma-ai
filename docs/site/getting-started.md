# Getting started

This page takes you from nothing to a repository with agent instructions and a skill installed.

## 1. Install

::: code-group

```bash [Homebrew]
brew install HasanKhatib/tap/hikma
```

```bash [Script]
curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash
```

:::

The script downloads the matching release, checks it against `checksums.txt`, and installs to `~/.local/bin`. Set `HIKMA_INSTALL_DIR` to change that, or pass a version: `... | bash -s -- v0.1.0`. On Windows run it in Git Bash and make sure the install directory is on your `PATH`.

Check the install:

```bash
hikma --version
hikma doctor
```

`doctor` needs `git`. The GitHub CLI (`gh`) is optional and only used by [`skill push`](/commands/skill-push) and for private GitHub repositories.

## 2. Install a skill from any repository

No setup is needed to install from a repository you name:

```bash
hikma skill install owner/repo            # pick from the skills in that repo
hikma skill install owner/repo my-skill   # install one by name
hikma skill install owner/repo my-skill --ref v1.0   # pin a tag, branch, or commit
```

Skills land in your agent's folder (`.agents/skills` by default, `.claude/skills` for Claude) and are recorded in `.hikma/lock.json`. See [skill install](/commands/skill-install).

## 3. Set up a whole repository

To prepare a repository for one or more agents:

```bash
hikma init --agent claude,codex --registry owner/repo --skill my-skill
```

This writes `AGENTS.md`, saves the agents and registry in `.hikma/config.json`, and installs the skill into `.claude/skills` and `.agents/skills`. Commit `.hikma/` so teammates share the setup. See [init](/commands/init) and [Set up a team repo](/guides/team-setup).

## 4. Use your own registry

Set a registry once and use bare names:

```bash
hikma config set registry owner/repo
hikma skill list
hikma skill install my-skill
```

## 5. Create and publish a skill

```bash
hikma skill create my-skill --description "What it helps with"
# edit .claude/skills/my-skill/SKILL.md (or .agents/skills/my-skill/SKILL.md)
hikma skill push my-skill
```

`push` validates the skill, shows the target registry and asks you to confirm, then opens a pull request. See [skill push](/commands/skill-push).

## Next

- [Command reference](/commands/)
- [Skill format](/guides/skill-format)
- [Run a registry](/guides/registry)
