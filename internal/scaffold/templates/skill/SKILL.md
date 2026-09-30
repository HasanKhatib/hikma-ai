---
name: {{.SkillName}}
description: "{{if .Description}}{{.Description}}{{else}}Replace this description. Say what the skill does and when to use it (include trigger keywords).{{end}}"
metadata:
  owner: {{.Owner}}
  email: {{.Email}}
  team: "{{.Team}}"
  policy: default
  last_validated: {{.Date}}
compatibility: "Describe required tools (e.g. gh CLI) and where the skill applies (Copilot, opencode, Claude Code)."
---

# {{.SkillName}}

## When to use

- Replace with scenarios and keywords that should load this skill.

## Instructions

- Replace with clear steps for the agent.
- Link to `references/` for long canonical material.

## Boundaries

- What this skill must not do (secrets, prod changes, etc.).
