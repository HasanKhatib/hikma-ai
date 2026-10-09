# hikma skill info

Show details for one skill.

```bash
hikma skill info [<owner/repo>] <name> [--ref <ref>] [--json]
```

Prints the name, description, owner, and source. With a repository but no name it offers a picker when run in a terminal.

```bash
hikma skill info deploy-helper
hikma skill info owner/repo deploy-helper --ref v1.2
```
