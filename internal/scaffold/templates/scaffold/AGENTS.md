# AGENTS.md

## Project

{{.ProjectName}}{{if .Technology}} - {{.Technology}}{{end}}
{{- if .Owner}}
Owner: {{.Owner}}
{{- end}}

## Rules

- Never modify production config without explicit instruction
- Never commit secrets or credentials
- Never run destructive operations without explicit confirmation

## Build and run

<!-- Fill in the commands to build, test, and run this project. -->

## Key invariants

<!-- List things that must always be true: API contracts, data constraints, etc. -->

## Skills

Skills are installed with the `hikma` CLI into {{.SkillPaths}}. Load a skill when the task matches its description.

- Installed skills and where they came from are recorded in `.hikma/lock.json`.
- Install one with `hikma skill install <owner/repo> <name>`{{if .Registry}}, or `hikma skill install <name>` to use the project registry `{{.Registry}}`{{end}}.

## Boundaries

- Do not modify infrastructure or deployment config without explicit instruction
- Do not commit secrets, tokens, or credentials
- Do not push to main directly - open a PR
