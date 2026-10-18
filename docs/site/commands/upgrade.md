# hikma upgrade

Upgrade hikma to the latest release.

```bash
hikma upgrade          # show the plan, ask, upgrade
hikma upgrade --yes    # skip the question
```

hikma works out how it was installed:

- **Homebrew:** it runs `brew upgrade HasanKhatib/tap/hikma`.
- **Anything else** (the install script, a downloaded binary): it downloads the install script and runs it. The script checks the archive against the release checksums and, when `gh` is signed in, its build attestation, then replaces the binary in the folder it already lives in.

The plan is printed first and nothing changes until you confirm. Without a terminal and without `--yes`, hikma prints the plan and exits with an error instead of waiting. On Windows the script needs Git Bash.
