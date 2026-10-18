# Versioning and deprecation policy

What you can rely on when you upgrade hikma.

## Semantic versioning

hikma follows [semantic versioning](https://semver.org). From 1.0, within a major version:

- **Commands, flags, and their meaning** do not change or disappear. New commands and flags can be added.
- **Exit codes** (`0`, `1`, `2`, `3`) keep their meaning. See [Exit codes and JSON](/reference/output).
- **`--json` documents** keep every existing field, with its name and meaning. New fields can appear, so ignore fields you do not know. Each document has a `version`.
- **The lockfile and config schemas** only grow. An older file is read and upgraded the next time hikma writes it. A file written by a newer hikma is refused, never rewritten.
- **Human-readable output** (tables, colors, wording) can change in any release. Use `--json` and exit codes in scripts.

Things that are not part of the contract: the banner, the menu text, error message wording, and the exact layout of any human-readable output.

## Deprecation

A feature is removed only in a major version, after at least one minor release that warns about it.

- The warning goes to **stderr**, so piped output and `--json` stay clean.
- It names the replacement and the version that removes the feature.
- The change is listed under *Deprecated* in the [changelog](https://github.com/HasanKhatib/hikma-ai/blob/main/CHANGELOG.md).
- Once you have switched to the replacement, the warning stops.

Security fixes are the one exception: if a behavior is unsafe, it can be tightened in a minor or patch release, and the changelog says so.

## Before 1.0

Releases below 1.0 can change the contracts above when there is a good reason, and the changelog says what changed. 0.6 and later follow these rules where they can, so the move to 1.0 changes as little as possible.

## Agents and `gh skill`

Agent folder layouts and `gh skill` behavior come from tools hikma does not control. hikma tracks them in a table, checks them before each major release, and treats a change on their side as a bug fix or a new feature in hikma, not a breaking change.
