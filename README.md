# Hikma AI

Hikma AI is a repo-native CLI for scaffolding AI-agent instructions and installing reusable skills from a registry you control.

This repository contains the CLI only. It does not contain a bundled skill registry. Point Hikma AI at a separate registry repository, for example `hasankhatib/ai`.

## Install

**macOS and Linux (Homebrew):**

```bash
brew install HasanKhatib/tap/hikma
```

**Windows (Git Bash) and anywhere else:** run the install script. It downloads the matching release, verifies its checksum, and installs to `~/.local/bin`.

```bash
curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash

# or, signed in with the GitHub CLI:
gh api -H "Accept: application/vnd.github.raw+json" \
  repos/HasanKhatib/hikma-ai/contents/scripts/install.sh | bash

# pin a version:
curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash -s -- v0.1.0
```

Set `HIKMA_INSTALL_DIR` to install somewhere else. On Windows the script needs Git Bash (included with Git for Windows); make sure the install directory is on your `PATH`.

**From source:**

```bash
go install github.com/hasankhatib/hikma-ai/cmd/hikma@latest
```

Reading skills needs only `git`; the `gh` CLI is needed just for `hikma skill push`.

## Quickstart

Install a skill from any repo, no setup needed:

```bash
hikma skill install owner/repo my-skill   # install one skill
hikma skill install owner/repo            # pick from the skills in that repo
```

Or set a registry once and use bare names:

```bash
hikma config set registry hasankhatib/ai
hikma skill list
hikma skill install agentkan
```

Create a skill and publish it to your registry:

```bash
hikma skill create my-skill --description "What this skill helps with"
hikma skill push my-skill        # shows the target registry and asks to confirm
hikma registry validate .        # check every skill in a registry checkout
```

Set a repo up for your agents in one step:

```bash
hikma init --agent claude,codex --registry hasankhatib/ai --skill agentkan
```

`init` writes each agent's instruction files, records the agents and registry in `.hikma/config.json` (commit it so teammates share them), and installs the skills into every selected agent's folder. After that, `hikma skill install <name>` installs into all of them.

| Agent | Instructions | Skills folder |
|---|---|---|
| `claude` | `AGENTS.md`, `CLAUDE.md` | `.claude/skills` |
| `codex`, `copilot`, `opencode` | `AGENTS.md` | `.agents/skills` |

Agents that share a folder are written once. For one-off use, set a personal default with `hikma config set agent claude` or pass `--agent`.

## How it works

- **Install from anywhere.** A source is a GitHub `owner/repo`, any git URL, or a local path. Use `--ref` to pin a branch, tag, or commit. Installing does not enforce a naming convention.
- **Push goes to your registry.** `hikma skill push` always targets your configured registry (a GitHub `owner/repo`), prints it with where the setting came from, and asks you to confirm (`--yes` in scripts). With write access it pushes a branch to the registry; without it, to your fork. It never force-pushes: an open PR for the skill gets a new commit, and a skill that was already merged gets a fresh branch.
- **Installs are tracked.** `.hikma/lock.json` records each skill's source, commit, and file hashes. `hikma skill update` pulls from the recorded source, refuses to overwrite local edits without `--force`, and warns when `scripts/` changed.

## Skill repositories

Any repo with skills works. Hikma looks for `skills/<name>/SKILL.md`, then `<name>/SKILL.md` at the top level, then a single `SKILL.md` at the root. No index file is needed.

```text
skills/
  my-skill/
    SKILL.md
```

A registry is just a repo you chose as your default source and push target. Registry values can be:

```text
hasankhatib/ai
https://github.com/hasankhatib/ai.git
git@github.com:hasankhatib/ai.git
./path/to/local/registry
```

Configuration precedence:

1. Command flag: `--registry`.
2. Environment variable: `HIKMA_REGISTRY`.
3. Project config: `.hikma/config.json` (commit it to share a registry with your team).
4. User config: `hikma config set registry <owner/repo>`.

The same order applies to `agent` (`--agent`, `HIKMA_AGENT`) and `agents` (`HIKMA_AGENTS`); a project's `agents` list wins over a personal `agent` default. See where each value comes from with `hikma config list`.

## Skill Naming

By default skill names are not restricted. To require kebab-case names when creating and pushing skills:

```bash
hikma config set naming kebab-case   # or: loose (default)
```

Installing never enforces a naming convention.

## Commands

```bash
hikma init [--agent claude,codex] [--registry <source>] [--skill <name>]...
hikma config list [--json]
hikma config get <key>
hikma config set <key> <value> [--project]
hikma config unset <key> [--project]
hikma config path
hikma config agent [copilot|codex|opencode|claude]
hikma config registry [owner/repo|url]
hikma skill list [owner/repo]
hikma skill info [owner/repo] <name>
hikma skill install [owner/repo] [name] [--ref <ref>] [--agent claude] [--force]
hikma skill update <name|--all> [--force]
hikma skill create <name>
hikma skill push <name> [--registry owner/repo] [--yes] [--codeowners]
hikma registry validate [source] [--ref <ref>]
hikma doctor
```

## Development

```bash
go test ./...
go build ./...
npx agentkan validate docs/board
```

## Release

Releases are tag-based. Pushing a `v*` tag runs GoReleaser, publishes binaries to GitHub Releases, and updates the Homebrew cask in `HasanKhatib/homebrew-tap` (needs the `HOMEBREW_TAP_GITHUB_TOKEN` repo secret). Pre-release tags such as `v0.1.0-rc.1` do not update the tap.

```bash
git tag v0.1.0
git push origin v0.1.0
```

Try a release locally with `goreleaser release --snapshot --clean`.

## Documentation Site

The repository can keep GitHub Pages docs for open-source reference material. A separate Cloudflare Pages site under `alkhatib.tech` can link to or mirror the same docs.
