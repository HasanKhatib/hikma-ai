# Security Policy

## Reporting a vulnerability

Please report security issues privately through GitHub: open a [private security advisory](https://github.com/HasanKhatib/hikma-ai/security/advisories/new) on this repository. Do not open a public issue for a vulnerability.

Include the version (`hikma --version`), your operating system, and steps to reproduce. You can expect an acknowledgement within a few days. Fixes are released as patch versions and credited in the changelog unless you prefer otherwise.

## Supported versions

Only the latest release receives security fixes.

## What counts

Hikma installs files from repositories you choose, and skills can contain scripts that your AI agent may run. These are in scope:

- a way to make `hikma` write files outside the intended skills folder (path traversal, symlinks)
- installing or updating from a source other than the one the user named or recorded in `.hikma/lock.json`
- the push flow writing to a registry other than the one shown at the confirmation prompt
- the install script or release artifacts failing to verify what they download
- leaking credentials in output or logs

Trusting the contents of a skill is up to you: review a skill and its `scripts/` before using it. `hikma skill update` lists changed files and flags changes to `scripts/`.

## Keep secrets out of reports

Do not publish tokens, private registry URLs, or credentials in issues, discussions, pull requests, or examples.
