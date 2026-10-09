# hikma registry validate

Check every skill in a registry against the skill format.

```bash
hikma registry validate [<source>] [--ref <ref>]
```

The source is `owner/repo`, a git URL, or a path (`.` for the repository you are in). Without an argument the configured registry is used.

## Checks

For each skill it verifies that:

- `SKILL.md` starts with YAML frontmatter
- the frontmatter has a `name` that matches the folder name
- the frontmatter has a `description`
- the folder name follows your [`naming`](/commands/config#naming) setting
- no template placeholders are left (such as "Replace this description.")

Symlinks are reported as warnings because they are ignored when a skill is installed or pushed.

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
