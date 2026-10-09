# hikma skill create

Scaffold a new skill.

```bash
hikma skill create <name> [flags]
```

Creates `<skills folder>/<name>/` for your active agent with a `SKILL.md` template and empty `scripts/`, `references/`, and `assets/` folders.

```bash
hikma skill create deploy-helper --description "Deploy a service and verify health"
```

## Flags

| Flag | Description |
|---|---|
| `--description <text>` | Pre-fill the description |
| `--owner <name>` | Owner (default: your GitHub login via `gh`) |
| `--team <name>` | Team name for the metadata |
| `--agent <list>` | Agents, comma-separated |

The name is checked against your [`naming`](/commands/config#naming) setting. The template contains placeholder text that must be replaced before you can push; `skill push` and `registry validate` both reject unfilled placeholders.

Next, edit `SKILL.md` (see [Skill format](/guides/skill-format)), then run [`hikma skill push`](/commands/skill-push).
