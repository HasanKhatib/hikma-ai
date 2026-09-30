# AGENTS.md

## Project

{{.ProjectName}} - {{.Technology}}
Owner: {{.Owner}}

## Rules

- Never modify production config without explicit instruction
- Never commit secrets or credentials
- Never run destructive operations without explicit confirmation

## Stack

- Technology: {{.Technology}}

## Build and run

<!-- Fill in the commands to build, test, and run this project. -->

## Key invariants

<!-- List things that must always be true: API contracts, data constraints, etc. -->

## Skills

Load skills from `{{.SkillPath}}` when the task matches their trigger keywords.
Run `hikma skill list` to see available skills.

## Boundaries

- Do not modify infrastructure or deployment config without explicit instruction
- Do not commit secrets, tokens, or credentials
- Do not push to main directly - open a PR
