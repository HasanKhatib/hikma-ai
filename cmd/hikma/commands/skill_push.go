package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/preflight"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// ghRun runs the gh CLI with optional stdin and returns trimmed stdout.
// Tests replace it with a fake.
var ghRun = func(stdin string, args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	out, err := cmd.Output()
	if err != nil {
		msg := err.Error()
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			msg = strings.TrimSpace(string(ee.Stderr))
		}
		return "", fmt.Errorf("gh %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(string(out)), nil
}

// forkWait is how long push waits for a new fork to become available.
var (
	forkWait  = 60 * time.Second
	forkSleep = time.Sleep
	pushNow   = time.Now
)

func ghGet(endpoint string, jq string) (string, error) {
	args := []string{"api", endpoint}
	if jq != "" {
		args = append(args, "--jq", jq)
	}
	return ghRun("", args...)
}

func ghSend(method, endpoint, jq string, body any) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	args := []string{"api", endpoint, "--method", method, "--input", "-"}
	if jq != "" {
		args = append(args, "--jq", jq)
	}
	return ghRun(string(data), args...)
}

func isNotFound(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "Not Found"))
}

// pushFile is one file to publish.
type pushFile struct {
	path    string // path inside the registry repo
	content []byte
	exec    bool
}

// pushPlan describes one skill publication.
type pushPlan struct {
	upstream config.Registry
	name     string
	files    []pushFile
	// codeowners, when non-empty, is the owner to add for skills/<name>/.
	codeowners string
}

func readPushFiles(skillDir, name string) ([]pushFile, error) {
	var files []pushFile
	err := filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}
		files = append(files, pushFile{
			path:    "skills/" + name + "/" + filepath.ToSlash(rel),
			content: data,
			exec:    info.Mode()&0o111 != 0,
		})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	return files, err
}

// pushSkill publishes a skill to the registry as a pull request. With write
// access it pushes a branch to the registry; otherwise it pushes to your fork.
// It never force-pushes: an open PR for the skill gets a new commit, and a
// leftover branch without an open PR is left alone in favor of a new branch.
func pushSkill(w io.Writer, plan pushPlan) (string, error) {
	upstream := plan.upstream.FullName()

	baseBranch, err := ghGet("repos/"+upstream, ".default_branch")
	if err != nil {
		return "", fmt.Errorf("get default branch: %w", err)
	}

	writeRepo, headOwner := upstream, ""
	canPush, err := ghGet("repos/"+upstream, ".permissions.push")
	if err != nil {
		return "", fmt.Errorf("check access to %s: %w", upstream, err)
	}
	if canPush != "true" {
		fmt.Fprintf(w, "No write access to %s; pushing through your fork.\n", upstream)
		fork, err := ensureFork(w, upstream, baseBranch)
		if err != nil {
			return "", err
		}
		writeRepo, headOwner = fork, strings.SplitN(fork, "/", 2)[0]
	}

	branch := "skill/" + plan.name
	existingPR := ""
	head, exists, err := branchHead(writeRepo, branch)
	if err != nil {
		return "", err
	}
	if exists {
		existingPR, err = findOpenPR(upstream, headOwner, branch)
		if err != nil {
			return "", err
		}
		if existingPR == "" {
			// The old branch has no open PR (merged or closed); do not reuse or rewrite it.
			branch = fmt.Sprintf("%s-%s", branch, pushNow().UTC().Format("20060102-150405"))
			exists = false
		}
	}
	if !exists {
		head, _, err = branchHead(writeRepo, baseBranch)
		if err != nil {
			return "", err
		}
		if head == "" {
			return "", fmt.Errorf("cannot find branch %s in %s", baseBranch, writeRepo)
		}
	}
	if existingPR != "" {
		fmt.Fprintf(w, "Updating branch %s on %s...\n", branch, writeRepo)
	} else {
		fmt.Fprintf(w, "Creating branch %s on %s...\n", branch, writeRepo)
	}

	baseTree, err := ghGet(fmt.Sprintf("repos/%s/git/commits/%s", writeRepo, head), ".tree.sha")
	if err != nil {
		return "", fmt.Errorf("read base commit: %w", err)
	}

	fmt.Fprintln(w, "Uploading files...")
	var entries []map[string]string
	for _, f := range plan.files {
		sha, err := ghSend("POST", "repos/"+writeRepo+"/git/blobs", ".sha", map[string]string{
			"content": base64.StdEncoding.EncodeToString(f.content), "encoding": "base64",
		})
		if err != nil {
			return "", fmt.Errorf("create blob for %s: %w", f.path, err)
		}
		mode := "100644"
		if f.exec {
			mode = "100755"
		}
		entries = append(entries, map[string]string{"path": f.path, "mode": mode, "type": "blob", "sha": sha})
		fmt.Fprintf(w, "  %s\n", strings.TrimPrefix(f.path, "skills/"+plan.name+"/"))
	}

	if plan.codeowners != "" {
		entry, err := codeownersEntry(w, plan, baseBranch)
		if err != nil {
			return "", err
		}
		if entry != nil {
			sha, err := ghSend("POST", "repos/"+writeRepo+"/git/blobs", ".sha", map[string]string{
				"content": base64.StdEncoding.EncodeToString([]byte(entry.content)), "encoding": "base64",
			})
			if err != nil {
				return "", fmt.Errorf("create CODEOWNERS blob: %w", err)
			}
			entries = append(entries, map[string]string{"path": ".github/CODEOWNERS", "mode": "100644", "type": "blob", "sha": sha})
			fmt.Fprintf(w, "  .github/CODEOWNERS  (+%s)\n", entry.line)
		}
	}

	tree, err := ghSend("POST", "repos/"+writeRepo+"/git/trees", ".sha", map[string]any{"base_tree": baseTree, "tree": entries})
	if err != nil {
		return "", fmt.Errorf("create tree: %w", err)
	}
	if tree == baseTree {
		fmt.Fprintln(w, "No changes to push.")
		return existingPR, nil
	}

	action := "add"
	if existingPR != "" || skillExistsOnBranch(upstream, baseBranch, plan.name) {
		action = "update"
	}
	message := fmt.Sprintf("feat(skills): %s %s", action, plan.name)
	commit, err := ghSend("POST", "repos/"+writeRepo+"/git/commits", ".sha", map[string]any{
		"message": message, "tree": tree, "parents": []string{head},
	})
	if err != nil {
		return "", fmt.Errorf("create commit: %w", err)
	}

	if exists {
		_, err = ghSend("PATCH", fmt.Sprintf("repos/%s/git/refs/heads/%s", writeRepo, branch), "", map[string]any{"sha": commit, "force": false})
	} else {
		_, err = ghSend("POST", "repos/"+writeRepo+"/git/refs", "", map[string]string{"ref": "refs/heads/" + branch, "sha": commit})
	}
	if err != nil {
		return "", fmt.Errorf("update branch: %w", err)
	}

	if existingPR != "" {
		fmt.Fprintf(w, "Updated PR: %s\n", existingPR)
		return existingPR, nil
	}
	fmt.Fprintln(w, "Opening PR...")
	headRef := branch
	if headOwner != "" {
		headRef = headOwner + ":" + branch
	}
	body := fmt.Sprintf("%s the `%s` skill.\n\nCreated via `hikma skill push`.", strings.ToUpper(action[:1])+action[1:]+"s", plan.name)
	pr, err := ghSend("POST", "repos/"+upstream+"/pulls", ".html_url", map[string]string{
		"title": message, "head": headRef, "base": baseBranch, "body": body,
	})
	if err != nil {
		return "", fmt.Errorf("create PR: %w", err)
	}
	fmt.Fprintf(w, "  PR: %s\n", pr)
	return pr, nil
}

func skillExistsOnBranch(repo, branch, name string) bool {
	_, err := ghGet(fmt.Sprintf("repos/%s/contents/skills/%s/SKILL.md?ref=%s", repo, name, branch), ".sha")
	return err == nil
}

// ensureFork returns the full name of your fork of upstream, creating it if
// needed and syncing its default branch.
func ensureFork(w io.Writer, upstream, baseBranch string) (string, error) {
	fork, err := ghSend("POST", "repos/"+upstream+"/forks", ".full_name", map[string]string{})
	if err != nil {
		return "", fmt.Errorf("fork %s: %w", upstream, err)
	}
	deadline := pushNow().Add(forkWait)
	for {
		if _, err := ghGet("repos/"+fork, ".full_name"); err == nil {
			break
		}
		if pushNow().After(deadline) {
			return "", fmt.Errorf("fork %s is not ready yet; try again in a minute", fork)
		}
		forkSleep(2 * time.Second)
	}
	// Best effort: bring the fork's default branch up to date with upstream.
	if _, err := ghSend("POST", "repos/"+fork+"/merge-upstream", "", map[string]string{"branch": baseBranch}); err != nil {
		fmt.Fprintf(w, "note: could not sync %s with %s: %v\n", fork, upstream, err)
	}
	fmt.Fprintf(w, "Using fork %s\n", fork)
	return fork, nil
}

// branchHead returns the commit SHA at the tip of branch in repo.
func branchHead(repo, branch string) (string, bool, error) {
	sha, err := ghGet(fmt.Sprintf("repos/%s/git/ref/heads/%s", repo, branch), ".object.sha")
	if err != nil {
		if isNotFound(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("read branch %s in %s: %w", branch, repo, err)
	}
	return sha, true, nil
}

// findOpenPR returns the URL of an open PR from branch (on headOwner's fork, or
// in upstream itself when headOwner is empty), or "".
func findOpenPR(upstream, headOwner, branch string) (string, error) {
	owner := headOwner
	if owner == "" {
		owner = strings.SplitN(upstream, "/", 2)[0]
	}
	out, err := ghGet(fmt.Sprintf("repos/%s/pulls?head=%s:%s&state=open", upstream, owner, branch), ".[0].html_url")
	if err != nil {
		return "", fmt.Errorf("find open PR: %w", err)
	}
	if out == "" || out == "null" {
		return "", nil
	}
	return out, nil
}

type codeownersChange struct{ content, line string }

func codeownersEntry(w io.Writer, plan pushPlan, baseBranch string) (*codeownersChange, error) {
	fmt.Fprintln(w, "Updating CODEOWNERS...")
	current := ""
	out, err := ghGet(fmt.Sprintf("repos/%s/contents/.github/CODEOWNERS?ref=%s", plan.upstream.FullName(), baseBranch), ".content")
	switch {
	case err == nil:
		decoded, derr := base64.StdEncoding.DecodeString(strings.Trim(out, `"`))
		if derr != nil {
			return nil, fmt.Errorf("decode CODEOWNERS: %w", derr)
		}
		current = string(decoded)
	case !isNotFound(err):
		return nil, fmt.Errorf("fetch CODEOWNERS: %w", err)
	}
	line := fmt.Sprintf("skills/%s/ @%s", plan.name, plan.codeowners)
	if strings.Contains(current, line) {
		return nil, nil
	}
	if current != "" && !strings.HasSuffix(current, "\n") {
		current += "\n"
	}
	return &codeownersChange{content: current + line + "\n", line: line}, nil
}

// readSkillOwner returns metadata.owner (or metadata.team) from SKILL.md.
func readSkillOwner(skillDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return "", fmt.Errorf("read SKILL.md: %w", err)
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", nil
	}
	rest := content[4:]
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return "", nil
	}
	var doc struct {
		Metadata struct {
			Owner string `yaml:"owner"`
			Team  string `yaml:"team"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(rest[:end]), &doc); err != nil {
		return "", fmt.Errorf("parse SKILL.md frontmatter: %w", err)
	}
	if doc.Metadata.Owner != "" {
		return doc.Metadata.Owner, nil
	}
	return doc.Metadata.Team, nil
}

func newSkillPushCmd(flagRegistry *string) *cobra.Command {
	var flagAgent string
	var assumeYes, withCodeowners bool

	cmd := &cobra.Command{
		Use:   "push <name>",
		Short: "Open a pull request to publish a skill to your registry",
		Long: `Publish a local skill to your configured registry as a pull request.

The target registry is always shown, and you must confirm it (--yes skips the
prompt). With write access the branch is pushed to the registry; otherwise it is
pushed to your fork. Nothing is force-pushed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			name := normalizeSkillName(args[0])
			if err := checkSkillName(name); err != nil {
				return err
			}
			registry, origin, err := resolvePushRegistry(*flagRegistry)
			if err != nil {
				return err
			}
			if result := preflight.Check(); !result.OK() {
				return preflightError(result)
			}

			skillDir, err := resolveSkillDirForAgent(name, flagAgent)
			if err != nil {
				return err
			}
			if !dirExists(skillDir) {
				ui.UserError("skill '%s' not found at %s/\n  run 'hikma skill create %s' to scaffold it first", name, skillDir, name)
				return fmt.Errorf("skill not found locally")
			}

			fmt.Fprintf(out, "Validating %s...\n", name)
			issues, err := validateSkillDir(skillDir, name)
			if err != nil {
				return err
			}
			if printIssues(cmd.ErrOrStderr(), skillDir, issues) {
				return fmt.Errorf("validation failed: fix the problems above before pushing")
			}
			fmt.Fprintln(out, "Validating...  ok")

			if err := confirmPush(cmd, name, skillDir, registry, origin, assumeYes); err != nil {
				return err
			}

			files, err := readPushFiles(skillDir, name)
			if err != nil {
				return err
			}
			plan := pushPlan{upstream: registry, name: name, files: files}
			if withCodeowners {
				owner, err := readSkillOwner(skillDir)
				if err != nil {
					return err
				}
				if owner == "" {
					if owner, err = ghGet("user", ".login"); err != nil {
						return fmt.Errorf("get current user: %w", err)
					}
				}
				plan.codeowners = owner
			}
			if _, err := pushSkill(out, plan); err != nil {
				return fmt.Errorf("push failed: %w", err)
			}
			fmt.Fprintln(out, "\nA maintainer will review and merge your skill.")
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return installedSkillCompletions()
		},
	}

	cmd.Flags().StringVar(&flagAgent, "agent", "", "AI agent (copilot, codex, opencode, claude)")
	cmd.Flags().BoolVarP(&assumeYes, "yes", "y", false, "skip the confirmation prompt")
	cmd.Flags().BoolVar(&withCodeowners, "codeowners", false, "also add the skill owner to .github/CODEOWNERS in the registry")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// resolvePushRegistry returns the GitHub registry that push writes to and where that choice came from.
func resolvePushRegistry(flagRegistry string) (config.Registry, string, error) {
	src, origin, err := resolveSource(flagRegistry)
	if err != nil {
		return config.Registry{}, "", err
	}
	if src.Kind != source.KindGitHub {
		return config.Registry{}, "", fmt.Errorf("push needs a GitHub registry (owner/repo); %s is a %s source", src.Display, src.Kind)
	}
	return config.Registry{Owner: src.Owner, Repo: src.Repo, Raw: src.Raw}, origin, nil
}
