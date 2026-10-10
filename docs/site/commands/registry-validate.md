# hikma registry validate

Check every skill in a registry against the skill format.

```bash
hikma registry validate [<source>] [--ref <ref>]
```

The source is `owner/repo`, a git URL, or a path (`.` for the repository you are in). Without an argument the configured registry is used.

| Flag | Description |
|---|---|
| `--ref <ref>` | Branch, tag, or commit to validate |
| `--format <text\|github>` | `github` also prints workflow annotations (`::error file=...`) so problems appear on the files in a pull request |

See [Run a registry](/guides/registry#check-it-on-every-pull-request) for the GitHub Action that does this for you.

## Checks

For each skill it verifies that:

- `SKILL.md` starts with YAML frontmatter
- the frontmatter has a `name` that matches the folder name
- the frontmatter has a `description`
- the folder name follows your [`naming`](/commands/config#naming) setting
- no template placeholders are left (such as "Replace this description.")
- the frontmatter follows the [Agent Skills spec](https://agentskills.io/specification): `name` up to 64 characters, `description` up to 1024, `compatibility` up to 500, and `metadata` values that are all strings

These are reported as warnings, not failures:

- symlinks, which are ignored when a skill is installed or pushed
- a `name` outside the spec's lowercase-and-hyphens form (enforce it with [`naming`](/commands/config#naming))
- frontmatter fields outside the spec, such as `when_to_use`. Claude Code accepts them, but claude.ai uploads and the Skills API reject them
- a `SKILL.md` over 500 lines, which the spec recommends splitting into `references/`

## Example

```text
$ hikma registry validate .
  ok    deploy-helper
  FAIL  release-notes
          error: frontmatter name "release-note" does not match folder "release-notes"
  FAIL  triage
          error: frontmatter is missing `description`

3 skill(s) checked, 2 failed.
```

The command exits non-zero when any skill fails, which makes it a good pull request check for a registry. `hikma skill push` runs the same checks before it publishes.

See [Skill format](/guides/skill-format).
