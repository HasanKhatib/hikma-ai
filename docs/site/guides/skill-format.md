# Skill format

A skill is a folder with a `SKILL.md` file, and optional supporting folders:

```text
my-skill/
  SKILL.md          required
  scripts/          optional: helper scripts
  references/       optional: long reference material
  assets/           optional: templates and other files
```

## SKILL.md

`SKILL.md` starts with YAML frontmatter, then instructions for the agent:

```markdown
---
name: my-skill
description: "Deploy a service and verify health. Use when asked to roll out or roll back."
metadata:
  owner: platform
  team: "platform-engineering"
---

# my-skill

## When to use
- Rolling out a new version

## Instructions
- Run the health check after each step
```

| Field | Required | Notes |
|---|---|---|
| `name` | yes | Must match the folder name |
| `description` | yes | What the skill does and when to use it. Agents use this to decide when to load it. |
| `metadata.owner`, `metadata.team` | no | Shown by `skill list`; used for `--codeowners`. `metadata` values must be strings, so quote numbers and dates |
| `license`, `compatibility`, `allowed-tools` | no | Defined by the spec. Most skills do not need `compatibility` |

Keep `SKILL.md` under 500 lines and move long material into `references/`. Agents load only `name` and `description` up front, then the full file when a skill matches, so a sharp description matters more than anything else. Fields outside the spec work in some agents but are rejected by claude.ai uploads and the Skills API; `registry validate` warns about them.

[`hikma skill create`](/commands/skill-create) generates a template with these fields.

## Where skills live

Hikma finds skills without an index file. It checks, in order:

1. `skills/<name>/SKILL.md`
2. `<name>/SKILL.md` at the repository root
3. a single `SKILL.md` at the repository root

The first layout that finds skills wins. The same layout is used by `gh skill`, so one repository works with both tools.

## Names

By default any name that is a single path segment is accepted. To require lowercase letters, digits, and hyphens when creating and publishing:

```bash
hikma config set naming kebab-case
```

Installing never enforces a naming rule.

## Validating

```bash
hikma registry validate .
```

See [registry validate](/commands/registry-validate).
