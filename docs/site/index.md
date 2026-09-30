# Hikma AI

Hikma AI is a CLI for adding AI-agent instructions and reusable skills to repositories.

The CLI does not ship with a skill registry. Configure a separate registry repository, such as `hasankhatib/ai`, and use Hikma AI to install from it.

## Quickstart

```bash
go install github.com/hasankhatib/hikma-ai/cmd/hikma@latest
hikma config agent claude
hikma config registry hasankhatib/ai
hikma init --agent claude
hikma skill list
```

## Links

- [Commands](commands/index.md)
- [Registry Guide](guides/registry.md)
- [Release Checklist](../migration-plan.md#release-readiness-checklist)
