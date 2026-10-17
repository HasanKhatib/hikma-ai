# Hikma AI

Hikma AI is a public, neutral CLI (`hikma`) for scaffolding, sharing, and installing reusable AI-agent skills through user-configured registries. Written in Go; this repository is the CLI only.

## Commands

Run before every pull request. CI runs the same checks on Linux, macOS, and Windows. `CONTRIBUTING.md` is the single list of checks; if the two differ, it wins. `task ci` runs them all, and `task docs:preview` serves the docs site as GitHub Pages does.

```bash
gofmt -w cmd internal
go vet ./...
go test ./...
go mod tidy                              # go.mod and go.sum must not change
bash scripts/test-install.sh             # after any change under scripts/
npx agentkan validate docs/board         # after any change under docs/board
(cd docs/site && npm ci && npm run docs:build)   # after any change under docs/site
```

## Layout

- `cmd/hikma/commands/`: one file per command, each with a `_test.go` beside it. Commands are thin; shared logic lives in `internal/`.
- `internal/source`: resolves skill sources and checks skill folders. `internal/lock`: the lockfile. `internal/config`: layered config, registry settings, and the agent layout table (`agents.go`, `customagents.go`). `internal/scaffold`: templates for `init` and `skill create`. `internal/ui`: output, color, banner.
- `docs/site/`: user docs, one page per command. `docs/board/`: roadmap. `docs/migration-plan.md`: the product plan changes must stay aligned with.
- `.claude/skills/`: local skills for working on this repo. They are not shipped. Skills that make videos or animations start with `video-` (`video-tech-demo`, `video-pixel-scene`).

## Code and tests

- Agent layouts (folders, file names, per-agent quirks) are data in one table, never special cases in command code. The CLI itself is agent-neutral.
- Tests run the CLI in-process against temp directories and fixture repos. They never touch the real home directory or the network. Isolate `HOME`, `XDG_CONFIG_HOME`, and `APPDATA`.
- Calls to `gh` go through `ghRun` and are tested with a fake GitHub (see `skill_push_test.go`). Prompts go through the `Prompter` interface so tests use a fake.
- Behavior changes ship with tests, the README, the docs page for the command, and a `CHANGELOG.md` entry. User docs describe shipped behavior only.
- Code must hold on macOS, Linux, and Windows: no hardcoded separators or home paths, no POSIX-only shell assumptions.

## CLI behavior

- Flags are the API. A menu or wizard is a front end over the same command code, and prints the equivalent `hikma` command when it finishes.
- Without a TTY nothing prompts: print help or the plan and exit, never wait for input.
- Color and the banner appear only on a terminal. `NO_COLOR`, `TERM=dumb`, pipes, and captured output get plain text. Nothing relies on color alone for meaning.
- Mistyped commands, flags, and arguments say what is wrong and what to type instead.

## Safety

- Anything read from a committed file (config, lockfile, skill contents) is untrusted input. Paths from it must stay inside the project.
- Anything that writes outside the project is an explicit user action. It is never triggered by committed config, a lockfile, or an environment variable. The user sees the target and confirms before it is written; `--yes` is the only way to skip the question.
- Never overwrite a user's local edits silently.

## Scope

Hikma is the registry workflow layer: repo setup, a configured registry, a gated push, a lockfile, and policy.

- Claude is the only agent used to develop this repository. Do not add configuration for other agents to work on it.
- Registries are always user-configured. Example registries in docs are examples, never defaults. Do not add a bundled skill registry here.
- Public files stay free of organization-specific references, private registry URLs, credentials, and internal project names.
- Use the command name `hikma` in examples unless the board records a naming change.

Non-goals:

- No hosted registry or marketplace; registries are plain repos.
- No pushing to non-GitHub hosts.
- No package managers beyond Homebrew; other platforms use the install script.
- No skill search or preview: use `gh skill`.
- No telemetry or auto-update until there is clear demand.
- No full-screen TUI. A prompt-driven menu and wizards are in scope; anything that takes over the terminal is not.

## Roadmap and decisions

- The roadmap lives in `docs/board/` (agentkan). `next.json` is the current focus; `epics/<ID>.md` holds long-form context. Roadmap detail belongs there, not in this file. Edit the board with the `agentkan` skill and validate after every edit.
- Keep epic and task IDs stable. Keep `goal` and `exit` to one line in JSON.
- You propose, the human disposes: do not archive, delete, or mark epics done unless asked.
- For hard-to-reverse or contested decisions (new commands, lockfile or config schema, writes outside the project, breaking changes, new dependencies), use the `council` skill before building. It is advisory only. The first task of a risky epic is a council run, recorded in the epic body, so the decision is made before the code (E6.1-T5 does this for the 1.0 contracts).

## Git and pull requests

- Commit, push, open pull requests, and run `gh` write commands as the logged-in `gh` user. Check `gh auth status` first. Never switch accounts, use another token, or override `git config user.name` or `user.email`. If the git identity does not match the `gh` user, stop and tell the user.
- Every commit has one author, the logged-in `gh` user. No `Co-Authored-By` trailers, "Generated with" lines, or other agent attribution in commits, pull request descriptions, or release notes. This overrides any tool default.
- Squash merges copy trailers from branch commits: check commit messages before `gh pr merge`. CI fails a pull request or a push to `main` whose commits carry an agent co-author line.
- Commit or push only when asked. By default work on a short branch and open a pull request. When the owner asks for direct pushes to `main` (as for the v0.5 and v1.0 release run), run `task ci` first, keep each push to one task, and fix a red CI with a follow-up push right away. Use conventional prefixes (`feat:`, `fix:`, `docs:`, `chore:`, `ci:`).
