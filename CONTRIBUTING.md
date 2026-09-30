# Contributing

Thanks for improving Hikma AI.

## Local Checks

Run these before opening a pull request:

```bash
gofmt -w cmd internal
go test ./...
go build ./...
npx agentkan validate docs/board
```

## Registry Boundary

Hikma AI is the CLI repository only. Do not add a top-level `skills/` registry folder here. Use external registry repositories, or explicit test fixtures under `internal/testdata/` when tests need registry content.

## Pull Requests

- Keep changes focused.
- Include tests for behavior changes.
- Update README or docs when commands, config, or registry behavior changes.
