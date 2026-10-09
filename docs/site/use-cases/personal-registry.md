# A personal registry

You write skills for yourself and want them in every project.

## 1. Create the repository

Create a repository on GitHub, for example `your-user/skills`, with a `skills/` folder.

## 2. Point Hikma at it

```bash
hikma config set registry your-user/skills
```

## 3. Write a skill

In any project:

```bash
hikma skill create commit-style --description "Write commit messages in my preferred style"
# edit .agents/skills/commit-style/SKILL.md
hikma skill push commit-style
```

`push` shows `your-user/skills`, asks you to confirm, and opens a pull request. Merge it.

## 4. Use it everywhere

```bash
cd another-project
hikma init --agent claude --skill commit-style
```

Later, pull in your improvements:

```bash
hikma skill update --all
```
