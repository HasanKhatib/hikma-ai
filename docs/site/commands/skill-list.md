# hikma skill list

List the skills in a registry or any repository.

```bash
hikma skill list [<owner/repo>] [--json]
```

Without an argument it reads the configured registry. Skills are found without an index file; see [Skill format](/guides/skill-format#where-skills-live).

```text
$ hikma skill list owner/repo
NAME                       DESCRIPTION                                         OWNER
deploy-helper              Deploy a service and check health after rollout     platform
release-notes              Draft release notes from merged pull requests       docs
```

`--json` prints the skills as JSON and sends status messages to stderr, so the output is safe to pipe.
