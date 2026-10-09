# A team registry with review

A team shares skills through a registry repository where every change is reviewed.

## Registry setup

1. Create the repository, with branch protection and required reviews on the default branch.
2. Add a CI job that checks every pull request:

   ```yaml
   name: Validate skills
   on: pull_request
   jobs:
     validate:
       runs-on: ubuntu-latest
       steps:
         - uses: actions/checkout@v7
         - name: Install hikma
           run: curl -fsSL https://raw.githubusercontent.com/HasanKhatib/hikma-ai/main/scripts/install.sh | bash
         - run: ~/.local/bin/hikma registry validate .
   ```

3. Choose a naming rule and commit it with the registry's docs: `hikma config set naming kebab-case`.

## Contributing a skill

Contributors do not need write access:

```bash
hikma skill create release-notes --description "Draft release notes from merged PRs"
# edit and test the skill in a project
hikma skill push release-notes
```

`push` forks the registry when needed and opens a pull request from the fork. Reviewers see the diff; CI validates the format. Use `--codeowners` to add the skill's owner to `.github/CODEOWNERS`.

## Consuming the registry

In each project repository:

```bash
hikma init --agent claude,codex --registry your-org/skills --skill release-notes
```

Commit `.hikma/`. Everyone who clones the project gets the same agents, registry, and skills, and `hikma skill update --all` brings in reviewed changes, showing exactly which files changed.
