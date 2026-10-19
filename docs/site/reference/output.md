# Exit codes and JSON output

What scripts and CI can rely on. From 1.0, these only change in a major version.

## Exit codes

| Code | Meaning | Examples |
|---|---|---|
| `0` | Success | Everything matched, or the command did what you asked |
| `1` | The command failed | A source could not be fetched, a file could not be written, a skill is not installed |
| `2` | Wrong usage | Unknown command or flag, missing argument, `sync --json` without `--yes` or `--dry-run` |
| `3` | Installed skills differ from the lockfile | [`skill verify`](/commands/skill-verify) found a difference, or [`sync --frozen`](/commands/sync) cannot reproduce the lockfile exactly |

Code `3` lets CI tell drift (something to review) from breakage (something to fix). The codes are the same on Linux, macOS, and Windows.

```bash
hikma skill verify
case $? in
  0) echo "clean" ;;
  3) echo "drift: review with hikma skill diff" ;;
  *) echo "could not check" ;;
esac
```

## JSON output

Commands with `--json` print one JSON document on stdout. Progress, warnings, and errors go to stderr, so the output is safe to pipe.

Every document has a `version` field, currently `1`. Within a major version, fields can be added but existing fields keep their names and meaning, so ignore fields you do not know. Arrays are always present, empty (`[]`) rather than `null`.

| Command | Document |
|---|---|
| `hikma skill verify --json` | `{ version, ok, checked, differ, skills: [{ path, name, status, added, modified, removed }] }` where `status` is `ok`, `modified`, or `missing` |
| `hikma skill diff <name> --json` | `{ version, skills: [{ path, name, source, commit, missing, localEdits: { added, modified, removed }, upstream: { commit, upToDate, scriptsChanged, added, modified, removed } }] }` |
| `hikma sync --json` | `{ version, ok, dryRun, skills: [{ path, name, source, commit, state, result, error? }] }` where `state` is `ok`, `restore`, or `modified`, and `result` is `up-to-date`, `planned`, `restored`, `skipped`, `failed`, or `not-run` |
| `hikma skill list --installed --json` | `{ version, skills: [{ scope, name, path, status, source, ref, commit, origin, alsoAt }] }` where `status` is `ok`, `modified`, `missing`, or `untracked` |
| `hikma skill list [<owner/repo>] --json` | `{ version, skills: [{ name, description, owner, path }] }` |
| `hikma skill import <source> [<name>] --json` | `{ version, dryRun, source: { source, ref, commit, tree, path, readFrom }, skill: { name, description }, files: [{ path, size, executable }], skipped: [{ path, reason }], changes: [{ file, reason, detail }], license: { value, note }, review: { scripts, binaries, outsideLinks, runLines }, warnings, target: { registry, origin, how } \| null, result: { url, mode } \| null }`. Needs `--yes` or `--dry-run`. `ref`, `commit`, `tree`, and `license.value` are `null` when unknown; `result` is `null` for a dry run, and `mode` is `pull-request` or `commit` |
| `hikma skill info <name> --json` | `{ version, skill: { name, description, owner, path } }` |
| `hikma config list --json` | `{ version, entries: [{ key, value, source }] }` |

`sync --json` never prompts, so it needs `--yes` (restore) or `--dry-run` (plan only). `skill diff --json` cannot be combined with `--patch`.

With `skill verify --json` and `sync --json` the document is printed even when the command exits `3` or `1`, so you can read what differed.

## Files with a version

The lockfile (`.hikma/lock.json`) and config files carry their own `version`. A file with a newer version than your `hikma` understands is refused with an error, and never rewritten. See [the lockfile](/commands/sync#versions) and [config](/commands/config#config-file-version).
