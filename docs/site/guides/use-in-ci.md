# Use Hikma in CI

Install a pinned `hikma`, restore the skills a repository was committed with, and fail the job when they drift. Everything here is a flag or an exit code that is documented on its own page; this page puts them in one place.

## Install a pinned version

```bash
curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | HIKMA_REQUIRE_ATTESTATION=1 bash -s -- v1.7.1
```

- Pass a tag so every run uses the same binary. Without one the script installs the latest release.
- `HIKMA_REQUIRE_ATTESTATION=1` fails the install instead of skipping the attestation check when `gh` is missing.
- It installs to `~/.local/bin` unless `HIKMA_INSTALL_DIR` says otherwise. Make sure that directory is on `PATH`.

See [Getting started](/getting-started) for the other ways to install.

## Restore and check skills

```bash
hikma sync --frozen --yes
```

`sync` restores every skill in `.hikma/lock.json` from its recorded source at its recorded commit and checks the files against the recorded hashes. `--frozen` makes any drift a failure instead of a skip, the way `npm ci` does for a lockfile. See [`hikma sync`](/commands/sync).

To check what is already installed without fetching anything:

```bash
hikma skill verify --json
```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | The command failed |
| `2` | Wrong usage |
| `3` | Installed skills differ from the lockfile |

Code `3` tells drift (review it) from breakage (fix it). The full table is in [Exit codes and JSON](/reference/output).

## Never wait for input

Hikma prompts only when standard input is a terminal. In CI it never waits: it prints the plan or fails and names the flag to pass, usually `--yes`. Pass `--no-input` to make that explicit, and `--no-color` or `NO_COLOR=1` for plain logs. See [Flags and terminal behavior](/reference/flags).

## Read the result in a script

Commands with `--json` print one versioned JSON document on stdout and send everything else to stderr, so it is safe to pipe. `sync --json` needs `--yes` or `--dry-run`. The document is printed even when the exit code is `1` or `3`.

```bash
hikma sync --frozen --yes --json | jq '.skills[] | select(.result == "failed")'
```

## Credentials

Hikma never reads or stores tokens. It runs `gh` and `git`, which use `GH_TOKEN` or `GITHUB_TOKEN` and your git configuration. A private registry needs a token that can read it. See [Configuration and environment](/reference/configuration).

## A GitHub Actions job

```yaml
jobs:
  skills:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Install hikma
        run: |
          curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | HIKMA_REQUIRE_ATTESTATION=1 bash -s -- v1.7.1
          echo "$HOME/.local/bin" >> "$GITHUB_PATH"
        env:
          GH_TOKEN: ${{ github.token }}
      - name: Skills match the lockfile
        run: hikma sync --frozen --yes
        env:
          GH_TOKEN: ${{ github.token }}
```
