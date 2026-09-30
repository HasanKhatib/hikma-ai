# Registry Guide

A Hikma AI registry is a separate repository that stores reusable skills. The CLI repository should not contain a top-level `skills/` registry folder.

## Minimal Shape

```text
skills/
  index.json
  my-skill/
    SKILL.md
```

## Configure A Registry

```bash
hikma config registry hasankhatib/ai
```

You can also override the configured registry for one command:

```bash
hikma skill list --registry hasankhatib/ai
```

## Supported Registry Values

```text
hasankhatib/ai
https://github.com/hasankhatib/ai.git
git@github.com:hasankhatib/ai.git
```

## Index File

The `skills/index.json` file should contain:

```json
{
  "generated": "2026-09-30T00:00:00Z",
  "skills": [
    {
      "name": "agentkan",
      "description": "Maintain an agentkan roadmap board.",
      "owner": "hasankhatib",
      "email": "",
      "last_validated": "2026-09-30",
      "compatibility": "Claude Code"
    }
  ]
}
```
