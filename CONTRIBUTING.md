# Contributing

Thanks for improving Hikma AI.

## Local checks

Run these before opening a pull request (CI runs them on Linux, macOS, and Windows):

```bash
gofmt -w cmd internal
go vet ./...
go test ./...
go mod tidy          # go.mod and go.sum must not change
bash scripts/test-install.sh   # if you touched scripts/
npx agentkan validate docs/board   # if you touched docs/board
```

## Registry boundary

Hikma AI is the CLI repository only. Do not add a top-level `skills/` registry folder here. Tests build their registries on the fly in temporary directories; do not commit registry content.

## Tests

- Command tests run the CLI in-process against a temporary git repository; see `cmd/hikma/commands/skill_install_test.go`.
- Calls to the `gh` CLI go through one function (`ghRun`) and are tested with a fake GitHub; see `skill_push_test.go`.
- Config tests isolate `HOME`, `XDG_CONFIG_HOME`, and `APPDATA` so they never touch your real configuration.

## Pull requests

- Keep changes focused and open them against `main`.
- Include tests for behavior changes.
- Update the README and `docs/site` when commands, config, or registry behavior changes.
- Use clear commit messages; conventional prefixes (`feat:`, `fix:`, `docs:`, `ci:`) are welcome.

## Planning

Roadmap and decisions live in `docs/board/`. See `CLAUDE.md` for the board rules and the project's non-goals.
