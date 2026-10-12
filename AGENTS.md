# Hikma AI

Hikma AI is a public, neutral CLI for scaffolding, sharing, and installing reusable AI-agent skills through user-configured registries.

## Agent Scope

- Claude is the only agent used to develop this repository. Do not add configuration for other agents to work on this repo.
- The `hikma` CLI itself is agent-neutral: it installs skills and scaffolds files into the right folder for each supported agent. Agent layouts are data in one table, never special cases in command code.
- Keep public files free of organization-specific references, private registry URLs, credentials, and internal project names.
- Prefer the CLI command name `hikma` in examples unless the naming decision changes on the board.
- Keep this repository CLI-only. Do not add a bundled skill registry to it.
- Registries are always user-configured. Example registries in docs are examples, never defaults.

## Board

The project roadmap lives in `docs/board/` as an agentkan board.

- `roadmap.json` is the live epic and task board.
- `next.json` is the current focus, critical path, and risk list.
- `archive.json` stores completed epics after human archive action.
- `epics/<ID>.md` contains long-form context for each epic.

At the start of a session, read `next.json` and the active or next epic bodies. Use the local `agentkan` skill to edit the board, and validate after every edit:

```bash
npx agentkan validate docs/board
```

Use `npx agentkan serve docs/board` when a human wants to operate the board.

## Board Rules

- AI proposes and the human disposes: do not archive, delete, or mark epics done unless explicitly asked.
- Keep epic and task IDs stable forever.
- Keep `goal` and `exit` to one line in JSON; put longer context in the epic markdown file.
- Update `next.json` when the current focus changes.
- Roadmap detail (versions, epic IDs, planned features) belongs on the board, not in this file.

## Engineering Principles

- Anything read from a committed file (config, lockfile, skill contents) is untrusted input. Paths from it must stay inside the project.
- Anything that writes outside the project must be an explicit user action. It is never triggered by committed config, a lockfile, or an environment variable, and the user sees the target before it is written.
- Never overwrite a user's local edits silently.
- Tests must not touch the real home directory or the network. Use temp directories and fixture repos.
- User docs describe shipped behavior only.

## Product Boundary

Hikma is the registry workflow layer: repo setup, a configured registry, a gated push, a lockfile, and policy. Users can:

- Initialize agent instructions in a repository for their chosen agents.
- Configure an external skill registry.
- List, install, create, update, and publish skills against that registry.
- Restore and verify installed skills from the lockfile.

Keep changes aligned with `docs/migration-plan.md`.

## Non-Goals

- No hosted registry or marketplace; registries are plain repos.
- No support for pushing to non-GitHub hosts.
- No package managers beyond Homebrew; other platforms use the install script.
- No skill search or preview: use `gh skill` for those.
- No telemetry, auto-update, or TUI until there is clear demand.
