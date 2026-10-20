# hikma upgrade

Upgrade hikma to the latest release.

```bash
hikma upgrade             # show both versions, upgrade, check the result
hikma upgrade --dry-run   # show both versions and what would run; change nothing
```

It shows the version you have and the latest release, and when they differ it upgrades at once, with no question:

```text
$ hikma upgrade
Current version: 1.2.0
Latest release:  1.3.0
Upgrading with Homebrew (brew update, then brew upgrade HasanKhatib/tap/hikma)...
  1.2.0 -> 1.3.0
Done.
```

When you already have the latest, it says so and changes nothing. A development build (`dev`) has no version to compare with, so it upgrades.

hikma works out how it was installed:

- **Homebrew:** it runs `brew update` and then `brew upgrade HasanKhatib/tap/hikma`.
- **Anything else** (the install script, a downloaded binary): it downloads the install script and runs it. The script checks the archive against the release checksums and, when `gh` is signed in, its build attestation, then replaces the binary in the folder it already lives in.

## Checking the result

A release is published before the Homebrew formula catches up. After the upgrade, hikma checks that the version now installed is the one it announced. If Homebrew still provides the old one, it says so instead of printing "Done.", and suggests trying again in a few minutes.

## Notes

- Running `hikma upgrade` is the confirmation: it replaces the hikma you ran it with, and says which release it is installing. Use `--dry-run` to look first. `--yes` is still accepted so existing scripts keep working, and does nothing.
- Without a terminal it does not wait for anything, so it can run in a script.
- If GitHub cannot be reached to learn the latest release, nothing is changed, and the error names the command to run yourself.
- On Windows the install script needs Git Bash.
