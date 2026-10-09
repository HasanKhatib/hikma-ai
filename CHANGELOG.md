# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims to follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [0.1.0] - 2026-10-09

First public release.

### Added
- Install skills from any source: GitHub `owner/repo`, a git URL, or a local path, with `--ref` to pin a version; pick from a list when no skill name is given.
- `.hikma/lock.json` records each install's source, commit, and file hashes. `hikma skill update` uses the recorded source, blocks on local edits, and flags changes to `scripts/`.
- Layered configuration (flag, environment, project `.hikma/config.json`, user config) with `hikma config list|get|set|unset|path`, showing where each value comes from.
- `hikma init` sets up one or more agents (Claude, Codex, Copilot, OpenCode): instruction files, shared project config, and skills in each agent's folder.
- `hikma skill push` confirms the target registry, works without write access through a fork, and never force-pushes.
- `hikma registry validate` checks skills against the skill format and your naming setting (`loose` or `kebab-case`).
- `hikma doctor`; shell completions.
- Homebrew cask and an install script for Windows (Git Bash) and other platforms, with checksum verification.

[Unreleased]: https://github.com/HasanKhatib/hikma-ai/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/HasanKhatib/hikma-ai/releases/tag/v0.1.0
