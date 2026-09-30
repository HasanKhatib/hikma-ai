# Hikma AI

Hikma AI is a repo-native CLI for scaffolding AI-agent instructions and installing reusable skills from a registry you control.

This repository contains the CLI only. It does not contain a bundled skill registry. Point Hikma AI at a separate registry repository, for example `hasankhatib/ai`.

## Install

From source:

```bash
go install github.com/hasankhatib/hikma-ai/cmd/hikma@latest
```

From a release archive, download the matching asset from GitHub Releases and place the `hikma` binary on your `PATH`.

## Quickstart

Configure your preferred agent and registry:

```bash
hikma config agent claude
hikma config registry hasankhatib/ai
```

Initialize a repository:

```bash
hikma init --agent claude --registry hasankhatib/ai
```

Browse and install skills:

```bash
hikma skill list
hikma skill install agentkan
```

Create a new local skill:

```bash
hikma skill create my-skill --description "What this skill helps with"
```

## Registry Model

Hikma AI expects a separate GitHub repository with a `skills/` directory and an index file:

```text
skills/
  index.json
  my-skill/
    SKILL.md
```

Registry values can be provided as:

```text
hasankhatib/ai
https://github.com/hasankhatib/ai.git
git@github.com:hasankhatib/ai.git
```

Configuration precedence:

1. Command flag: `--registry`.
2. Environment variable: `HIKMA_REGISTRY`.
3. User config: `hikma config registry <owner/repo>`.

## Commands

```bash
hikma init
hikma config agent [copilot|codex|opencode|claude]
hikma config registry [owner/repo|url]
hikma skill list [--registry owner/repo]
hikma skill info <name> [--registry owner/repo]
hikma skill install <name> [--agent claude] [--registry owner/repo]
hikma skill update <name|--all> [--registry owner/repo]
hikma skill create <name>
hikma skill push <name> [--registry owner/repo]
hikma doctor
hikma tap
```

## Development

```bash
go test ./...
go build ./...
npx agentkan validate docs/board
```

## Release

Releases are tag-based. Pushing a `v*` tag runs GoReleaser and publishes binaries to GitHub Releases.

```bash
git tag v1.0.0
git push origin v1.0.0
```

## Documentation Site

The repository can keep GitHub Pages docs for open-source reference material. A separate Cloudflare Pages site under `alkhatib.tech` can link to or mirror the same docs.
