# hikma skill import

Bring a skill you found anywhere into your own registry, after a review of what is in it.

```bash
hikma skill import <source> [<name>] [flags]
```

Requires the GitHub CLI (`gh`), signed in, to write to the registry. Reading a public source needs only `git`.

## Forms

```bash
hikma skill import owner/repo code-review                                    # a skill in a repo
hikma skill import owner/repo/skills/code-review                             # an exact path skips discovery
hikma skill import https://github.com/owner/repo/tree/main/skills/code-review  # a link copied from the browser
hikma skill import https://github.com/owner/repo/blob/main/skills/code-review/SKILL.md
hikma skill import owner/repo                                                # pick from the skills in that repo
hikma skill import ./local/checkout my-skill                                 # a folder on your machine
```

The source is always required: `hikma skill import code-review` is an error that tells you to use `install` (to install from your registry) or to name a repository. Sources are github.com repositories and local folders. Other hosts, including GitHub Enterprise, are not supported yet and get a clear message, not a clone error.

A tree URL can name a branch with slashes in it (`feature/skills`). Hikma matches the URL against the branches and tags of the repository, longest first. If it cannot ask the remote it reads the first part as the ref and says so.

## What happens

1. Hikma reads the source and finds the skills in it. Nothing is written to your machine except a temporary download.
2. It checks the skill with the same rules as [`registry validate`](/commands/registry-validate).
3. **It shows a review**: the files and their sizes, the licence, and anything worth reading first. Then it asks.
4. **It shows the target**: the registry and where the setting came from, the source commit, and whether the skill will be committed or opened as a pull request, in plain words.
5. The skill is published **the way [`skill push`](/commands/skill-push) publishes**, so it follows the registry's rules. Nothing is force-pushed.
6. It prints the next command, such as `hikma skill install code-review`.

All checks and the review finish before the first call to GitHub, so a skill that fails them never touches your registry.

## The review

```text
Importing pdf
  from:     anthropics/skills @ dbd4588 (skills/pdf)
  read as:  git objects at dbd4588
  files:    12, 57.3 KiB
    SKILL.md  7.9 KiB
    scripts/extract_form_structure.py  3.9 KiB  (executable)
    ...
  licence:  Proprietary. LICENSE.txt has complete terms (license in SKILL.md)
  Read these before you continue. Skills run with your agent's permissions:
  scripts and executables: scripts/check_bounding_boxes.py, ...
```

The review flags:

- **Scripts and executables**: anything under `scripts/`, files with the executable bit, and files that start with `#!`.
- **Binary files**, and **links in `SKILL.md` to paths outside the skill folder**.
- **Lines in `SKILL.md` that pipe a download into a shell** (`curl ... | sh`).
- **The licence**: the `license` field in `SKILL.md`, a `LICENSE` file in the skill folder, or one at the root of the repository (which is not copied). With none of these the review and the confirmation both warn that you may not have the right to copy and share the skill.
- **What was skipped**: symlinks, submodules, and a file named `.hikma-origin.json`.

::: warning The review is a pointer, not a scanner
Skills can contain scripts, and instructions that your agent follows with your permissions. Provenance shows where a skill came from, not that it is safe. Read `SKILL.md` and everything the review lists before you merge. Hikma does not look for malicious content.
:::

## Flags

| Flag | Description |
|---|---|
| `--ref <ref>` | Branch, tag, or commit to read. For a URL, put the ref in the URL |
| `--as <name>` | Import under a different name. Rewrites the `name:` line in `SKILL.md`, because the spec requires the name to match the folder |
| `--force` | Replace a skill of the same name that the registry already has |
| `--pr` | Open a pull request even if you can write to the registry |
| `-y`, `--yes` | Skip the confirmation (required without a terminal) |
| `--dry-run` | Show the review and the plan and write nothing. The registry is not contacted, and without a configured registry it still reviews the skill |
| `--json` | Print the plan, and the result, as one JSON document. Needs `--yes` or `--dry-run`. The prose goes to stderr |
| `--registry <source>` | Registry to import into, for this command |

`--yes` skips the question and nothing else: it does not hide the licence warning, and what the confirmation says about how the skill is published is still printed, so it shows in a job log.

## How it is published

Import asks GitHub, when it needs to, and does not store the answer:

| Situation | What import does |
|---|---|
| You can write, and no rule requires pull requests | Commits to the registry's default branch, whatever it is called (`main`, `master`, ...), with no review |
| A rule on the default branch requires pull requests | Opens a pull request on its own and says why |
| You cannot write to the registry | Opens a pull request from your fork |
| You pass `--pr` | Always opens a pull request, from a branch named `import/<name>` |

This is the same decision `skill push` makes, shown in [How publishing decides](/guides/registry#how-publishing-decides). `hikma init` and `hikma doctor` print which of these applies to your registry.

::: warning In CI
With write access and `--yes`, an import commits third-party content straight to the default branch, and nothing waits for a review. A script that should publish through review passes `--pr`, or the registry requires pull requests with a branch rule or ruleset, which import then follows. The confirmation line always says which of the two will happen. The review is printed first either way, so a job log shows what was brought in.
:::

## What is copied

Files are copied **byte for byte**, read from git objects at the commit, so Windows line-ending conversion, `.gitattributes` filters, and Git LFS cannot change them. Two things differ from upstream, and both are listed in the review and recorded in the origin file:

- **`metadata.github-*` is removed from `SKILL.md`.** `gh skill install` writes these keys to remember where a skill came from. If they stayed, `gh skill update` on a machine that installed from your registry could fetch the upstream skill and skip your review. `gh skill publish` removes the same keys.
- **With `--as`, the `name:` line** is rewritten.

If the keys are in a form hikma cannot remove safely (such as `metadata: {github-repo: ...}`), the import stops and says so.

Executable bits come from the git file mode, so they survive Windows. Local folders are read from the working tree instead, and the review says so.

### Limits

An import takes at most **100 files** and **5 MiB** in total. There is no override in this release. Entries that could leave the skill folder are refused, and a Git LFS pointer stops the import, because the real file is not in the repository. Two files whose names differ only in letter case are refused, because they would collide on macOS and Windows.

## Where it came from

`skills/<name>/.hikma-origin.json` is added to the registry with the skill:

```json
{
  "version": 1,
  "source": "owner/repo",
  "ref": null,
  "commit": "dbd4588...",
  "tree": "9f2c1e...",
  "path": "skills/pdf",
  "license": "MIT",
  "files": { "SKILL.md": "sha256:..." },
  "changes": [
    { "file": "SKILL.md", "reason": "strip-github-metadata", "detail": "removed metadata.github-repo", "upstream": "sha256:..." }
  ]
}
```

- `source` never holds credentials or a local path: it is `owner/repo`, or `local` for a folder. `ref` is the ref you asked for, or `null`.
- `commit` and `tree` (the git tree id of the skill folder) are recorded for a remote source, and for a clean local git tree. For a folder with uncommitted changes, or one that is not a git repository, they are `null` and the review says why.
- `files` has the hash of every stored file except this one. `changes` lists each difference from upstream with the upstream hash, so edited-by-import can be told from edited-later.
- `license` is the text of the `license` field, as written, not an SPDX identifier.

The file is advisory. It records a source and does not prove the copy is safe. A skill installed from your registry carries it into the agent folder like any other file. See the [policy](/reference/policy#the-origin-file) for how the format may change.

## Name collisions

Hikma checks the registry before uploading anything. If it already has a skill of that name the import stops and shows both sides:

```text
error: org/reg already has a skill called "alpha", so nothing was imported
  registry has: SKILL.md, old.md
  import has:   SKILL.md, scripts, .hikma-origin.json
import it beside the existing one: hikma skill import owner/repo alpha --as <name>
or replace the existing one: add --force
```

`--force` replaces it, and removes files of the old copy that the new one does not have. With both flags, `--as` picks the name and `--force` allows replacing that name. A name that differs only in letter case from an existing skill is always refused, because both would collide on macOS and Windows: use `--as`.

## In scripts and CI

- Nothing prompts without a terminal. Without `--yes` the command prints the plan and exits non-zero.
- Without a name and without a terminal, the skills in the source are printed one per line as `name<TAB>path<TAB>description`, and the exit code is `2`. Pass a name or an exact path.
- `--dry-run --json` is the way to check what an import would do. The [document](/reference/output#json-output) has `version`, `source`, `skill`, `files`, `skipped`, `changes`, `license`, `review`, `warnings`, and `target`.
- Calls to GitHub that are rate limited (HTTP 429, or a secondary limit) are retried after a minute, then two, then four, and then fail with a message that says how many files were uploaded and that nothing was committed. [`skill push`](/commands/skill-push) retries the same way.

## Private sources and other situations

| Situation | What happens |
|---|---|
| Private upstream | Git is tried first with your credential helper, then the GitHub CLI. Hikma never asks for a password: `GIT_TERMINAL_PROMPT=0`. |
| Offline or behind a proxy | `git` honours your proxy settings. Reading and review happen before any call to GitHub, and the error names the step that failed. |
| A monorepo | Give an exact path or a tree URL. Hikma fetches the tree without file contents and reads only the skill it needs. |
| Two skills with the same name | They are listed with their paths. Pick one by path. |
| Folders that start with a dot | Skipped when looking for skills, such as `.claude/skills`. Name the exact path to import from one. |
| A skill in your own agent folder | It is already a local skill: Hikma points you at [`skill push`](/commands/skill-push). |
| A folder with uncommitted changes | Read from the working tree. The commit and tree are `null`, and the review says so. |

Where Hikma looks for skills: `SKILL.md` at the root, `<name>/`, `skills/<name>/`, `skills/<scope>/<name>/`, and `plugins/<scope>/skills/<name>/`. A namespaced skill (`skills/author/code-review`) is imported under its last segment and its full path is recorded.

## Exit status

`0` on success, `1` on failure, `2` for wrong usage. See [Exit codes and JSON](/reference/output).

## Not in this release

`import --update` (compare with the upstream again, using the recorded hashes), `--all` for a whole repository, hosts other than github.com, and installing right after importing. For searching and previewing public skills, use [`gh skill`](/commands/skill-adopt#using-hikma-with-gh-skill).
