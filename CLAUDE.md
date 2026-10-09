# Hikma AI

Hikma AI is a public, neutral CLI for scaffolding, sharing, and installing reusable AI-agent skills through user-configured registries.

## Agent Scope

- Claude is the only agent used to develop this repository. Do not add opencode or other agent configuration for working on this repo.
- The `hikma` CLI itself is agent-neutral: it must install skills and scaffold files into the right folder for each supported agent (for example `.claude/skills` for Claude, `.agents/skills` for others). Agent choice is part of `hikma init`.
- Keep public files free of organization-specific references, private registry URLs, credentials, and internal project names.
- Prefer the CLI command name `hikma` in examples unless the naming decision changes on the board.
- Keep this repository CLI-only. Do not add a top-level bundled skill registry folder to this repo.
- Treat `hasankhatib/ai` as the planned personal registry example, not as a hardcoded default.

## Board

The project roadmap lives in `docs/board/` as an agentkan board.

- `docs/board/roadmap.json` is the live epic and task board.
- `docs/board/next.json` is the current focus, critical path, and risk list.
- `docs/board/archive.json` stores completed epics after human archive action.
- `docs/board/epics/<ID>.md` contains long-form context for each epic.

At the start of a session, read `docs/board/next.json` and the active or next epic bodies. When editing the board, use the local `agentkan` skill and validate with:

```bash
npx agentkan validate docs/board
```

Use the viewer when a human wants to operate the board:

```bash
npx agentkan serve docs/board
```

## Board Rules

- AI proposes and the human disposes: do not archive epics, delete epics, or mark epics done unless explicitly asked.
- Keep epic IDs stable forever.
- Keep `goal` and `exit` fields to one line in JSON; put longer context in the epic markdown file.
- Update `docs/board/next.json` when the current focus changes.
- Run board validation after every board edit.

## Current Product Boundary

The public MVP lets users:

- Initialize AI-agent instructions in a repository.
- Configure an external skill registry, such as `hasankhatib/ai`.
- List, install, create, update, and publish skills against that registry.
- Choose an agent during `hikma init` and get skills and instructions in that agent's folder layout.

This repo now contains the CLI source. Keep future changes aligned with `docs/migration-plan.md` and never add a top-level registry `skills/` directory here.

## Non-Goals

- No hosted registry or marketplace; registries are plain repos.
- No support for pushing to non-GitHub hosts.
- No package managers beyond Homebrew; other platforms use the install script.
- No skill search or preview, and no user-scope (global) installs: use `gh skill` for those. Hikma is the registry workflow layer: repo setup, a configured registry, a gated push, a lockfile, and policy.
- No telemetry, auto-update, or TUI until there is clear demand.
