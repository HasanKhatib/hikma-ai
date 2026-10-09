package commands

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/hasankhatib/hikma-ai/internal/config"
	"github.com/hasankhatib/hikma-ai/internal/preflight"
	"github.com/hasankhatib/hikma-ai/internal/scaffold"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newSkillCmd() *cobra.Command {
	var flagRegistry string
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Install, create, update, and publish skills",
	}
	cmd.PersistentFlags().StringVar(&flagRegistry, "registry", "", "registry to use for this command (owner/repo, git URL, or local path)")

	cmd.AddCommand(
		newSkillListCmd(&flagRegistry),
		newSkillInfoCmd(&flagRegistry),
		newSkillInstallCmd(&flagRegistry),
		newSkillUpdateCmd(),
		newSkillCreateCmd(),
		newSkillPushCmd(&flagRegistry),
	)

	return cmd
}

func newSkillCreateCmd() *cobra.Command {
	var flagOwner, flagTeam, flagDescription string
	var flagAgent string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Scaffold a new skill into the active agent's skill directory",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill create <name> [flags]\n\nExample: hikma skill create my-skill --description \"short description\" --team my-team")
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts exactly one skill name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := checkSkillName(name); err != nil {
				return err
			}

			isAgentOverride := flagAgent != ""
			selection, err := config.ResolveSelection(flagAgent)
			if err != nil {
				return fmt.Errorf("error: %w\nrun 'hikma doctor' to check your environment", err)
			}
			printActiveSelection(selection, isAgentOverride)

			targetDir, err := resolveSkillDirForAgent(name, flagAgent)
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}

			if _, err := os.Stat(targetDir); err == nil {
				ui.UserError("%s already exists at %s/\n  choose a different name or remove the existing directory", name, targetDir)
				return fmt.Errorf("already exists")
			}

			// Resolve author from gh; flags override resolved values.
			owner, email := resolveGHUser()
			if flagOwner != "" {
				owner = flagOwner
			}

			fmt.Printf("Resolving author from gh... %s\n", owner)
			fmt.Println()

			created, err := scaffold.CreateSkill(scaffold.SkillOptions{
				SkillName:   name,
				Owner:       owner,
				Email:       email,
				Team:        flagTeam,
				Description: flagDescription,
				Date:        time.Now().Format("2006-01-02"),
				TargetDir:   targetDir,
			})
			if err != nil {
				return fmt.Errorf("scaffold skill: %w", err)
			}

			fmt.Printf("Created %s/\n", targetDir)
			for _, p := range created {
				rel := strings.TrimPrefix(p, targetDir+string(filepath.Separator))
				fmt.Printf("  %s\n", rel)
			}

			fmt.Printf("\nOwner: %s", owner)
			if email != "" {
				fmt.Printf(" (%s)", email)
			}
			fmt.Println()
			fmt.Printf("\nEdit %s/SKILL.md, then run 'hikma skill push %s'.\n", targetDir, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&flagOwner, "owner", "", "override owner (default: resolved from gh)")
	cmd.Flags().StringVar(&flagTeam, "team", "", "team name to write into SKILL.md metadata")
	cmd.Flags().StringVar(&flagDescription, "description", "", "short description to pre-fill in SKILL.md")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "AI agent (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func newSkillPushCmd(flagRegistry *string) *cobra.Command {
	var flagAgent string
	var assumeYes, withCodeowners bool

	cmd := &cobra.Command{
		Use:   "push <name>",
		Short: "Open a pull request to publish a skill to the registry",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill push <name>\n\nRun 'hikma skill create <name>' to scaffold a new skill first.")
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts exactly one skill name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := normalizeSkillName(args[0])
			if err := checkSkillName(name); err != nil {
				return err
			}
			activeRegistry, origin, err := resolvePushRegistry(*flagRegistry)
			if err != nil {
				return err
			}
			pushTarget = &activeRegistry

			result := preflight.Check()
			if !result.OK() {
				return preflightError(result)
			}

			isAgentOverride := flagAgent != ""
			selection, err := config.ResolveSelection(flagAgent)
			if err != nil {
				return fmt.Errorf("error: %w\nrun 'hikma doctor' to check your environment", err)
			}
			printActiveSelection(selection, isAgentOverride)

			skillDir, err := resolveSkillDirForAgent(name, flagAgent)
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}

			if _, err := os.Stat(skillDir); os.IsNotExist(err) {
				ui.UserError("skill '%s' not found at %s/\n  run 'hikma skill create %s' to scaffold it first", name, skillDir, name)
				return fmt.Errorf("skill not found locally")
			}

			// Validate: no unfilled placeholders.
			fmt.Printf("Validating %s...\n", name)
			violations, err := scaffold.ValidatePlaceholders(skillDir)
			if err != nil {
				return fmt.Errorf("validate %s: %w", name, err)
			}
			if len(violations) > 0 {
				for _, v := range violations {
					fmt.Fprintf(os.Stderr, "  FAIL  %s\n", v)
				}
				ui.UserError("fill in all placeholders in %s/SKILL.md before pushing", name)
				return fmt.Errorf("validation failed")
			}
			fmt.Println("Validating...  ok")

			// Gate: always confirm which registry receives the skill.
			if err := confirmPush(cmd, name, skillDir, activeRegistry, origin, assumeYes); err != nil {
				return err
			}

			// Detect whether this is a first push or an update.
			branch := fmt.Sprintf("skill/%s", name)
			isUpdate := ghAPIBranchExists(branch)
			action := "add"
			if isUpdate {
				action = "update"
			}

			// Step 1: get HEAD SHA of the default branch.
			if isUpdate {
				fmt.Println("Updating branch " + branch + " on " + activeRegistry.FullName() + "...")
			} else {
				fmt.Println("Creating branch " + branch + " on " + activeRegistry.FullName() + "...")
			}
			baseBranch, err := ghAPIString(activeRegistry.GitHubAPIRepo(), "--jq", ".default_branch")
			if err != nil {
				ui.Error("get default branch: %s", err)
				return fmt.Errorf("push failed")
			}
			headSHA, err := ghAPIString(
				activeRegistry.GitHubAPIRepo()+"/git/refs/heads/"+baseBranch,
				"--jq", ".object.sha",
			)
			if err != nil {
				ui.Error("get HEAD SHA: %s", err)
				return fmt.Errorf("push failed")
			}

			// Step 2: create a blob for each file.
			fmt.Println("Uploading files...")
			var blobs []blobEntry
			err = walkSkillFiles(skillDir, func(rel, content string) error {
				sha, blobErr := ghAPICreateBlob(content)
				if blobErr != nil {
					return fmt.Errorf("create blob for %s: %w", rel, blobErr)
				}
				registryPath := fmt.Sprintf("skills/%s/%s", name, rel)
				blobs = append(blobs, blobEntry{path: registryPath, sha: sha})
				fmt.Printf("  %s\n", rel)
				return nil
			})
			if err != nil {
				return err
			}

			// Step 2b (opt-in): update CODEOWNERS with the skill owner.
			if withCodeowners {
				owner, err := readSkillOwner(skillDir)
				if err != nil {
					ui.Error("read skill owner: %s", err)
					return fmt.Errorf("push failed")
				}
				if owner == "" {
					user, userErr := ghUser()
					if userErr != nil {
						ui.Error("get current user: %s", userErr)
						return fmt.Errorf("push failed")
					}
					owner = user
				}
				if owner != "" {
					fmt.Println("Updating CODEOWNERS...")
					codeowners, err := fetchRegistryFileContent(".github/CODEOWNERS")
					if err != nil {
						ui.Error("fetch CODEOWNERS: %s", err)
						return fmt.Errorf("push failed")
					}
					entry := fmt.Sprintf("skills/%s/ @%s", name, owner)
					if !strings.Contains(codeowners, entry) {
						if len(codeowners) > 0 && !strings.HasSuffix(codeowners, "\n") {
							codeowners += "\n"
						}
						codeowners += entry + "\n"
						codeownersSHA, blobErr := ghAPICreateBlob(codeowners)
						if blobErr != nil {
							ui.Error("create CODEOWNERS blob: %s", blobErr)
							return fmt.Errorf("push failed")
						}
						blobs = append(blobs, blobEntry{path: ".github/CODEOWNERS", sha: codeownersSHA})
						fmt.Printf("  .github/CODEOWNERS  (+%s)\n", entry)
					}
				}
			}

			// Step 3: create tree.
			treeSHA, err := ghAPICreateTree(headSHA, blobs)
			if err != nil {
				ui.Error("create tree: %s", err)
				return fmt.Errorf("push failed")
			}

			// Step 4: create commit.
			commitSHA, err := ghAPICreateCommit(
				fmt.Sprintf("feat(skills): %s %s", action, name),
				treeSHA,
				headSHA,
			)
			if err != nil {
				ui.Error("create commit: %s", err)
				return fmt.Errorf("push failed")
			}

			// Step 5: create or force-update branch.
			if err := ghAPIEnsureBranch(branch, commitSHA); err != nil {
				ui.Error("%s", err)
				return fmt.Errorf("push failed")
			}

			// Step 6: open or find existing PR.
			if isUpdate {
				fmt.Println("Updating PR...")
			} else {
				fmt.Println("Opening PR...")
			}
			prBody := fmt.Sprintf("Adds the `%s` skill to the registry.\n\nCreated via `hikma skill push`.", name)
			if isUpdate {
				prBody = fmt.Sprintf("Updates the `%s` skill in the registry.\n\nCreated via `hikma skill push`.", name)
			}
			prURL, err := ghAPICreatePR(
				fmt.Sprintf("feat(skills): %s %s", action, name),
				branch,
				prBody,
			)
			if err != nil {
				ui.Error("create PR: %s", err)
				return fmt.Errorf("push failed")
			}

			fmt.Printf("  PR: %s\n", prURL)
			fmt.Println()
			if isUpdate {
				fmt.Println("A maintainer or skill owner will review and merge your update.")
			} else {
				fmt.Println("A maintainer will review and merge your skill.")
			}
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

// checkSkillName validates a name for create and push against the configured naming convention.
func checkSkillName(name string) error {
	naming, err := config.NamingConvention()
	if err != nil {
		return err
	}
	return config.CheckSkillName(naming, name)
}

// resolveGHUser asks gh for the current user's login and primary email.
// On any error it returns "unknown" and "".
func resolveGHUser() (login, email string) {
	out, err := exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return "unknown", ""
	}
	login = strings.Trim(strings.TrimSpace(string(out)), `"`)

	outEmail, err := exec.Command("gh", "api", "user/emails",
		"--jq", "[.[] | select(.primary==true)][0].email").Output()
	if err != nil {
		return login, ""
	}
	email = strings.Trim(strings.TrimSpace(string(outEmail)), `"`)
	if email == "null" {
		email = ""
	}
	return login, email
}

// walkSkillFiles walks skillDir and calls fn for each file with its relative
// slash-separated path and string content.
func walkSkillFiles(skillDir string, fn func(rel, content string) error) error {
	return filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		// Use forward slashes for the registry path.
		rel := filepath.ToSlash(strings.TrimPrefix(path, skillDir+string(filepath.Separator)))
		return fn(rel, string(data))
	})
}

// ghAPIString runs gh api <endpoint> with extra args and returns trimmed output.
func ghAPIString(endpoint string, extraArgs ...string) (string, error) {
	args := append([]string{"api", endpoint}, extraArgs...)
	out, err := exec.Command("gh", args...).Output()
	if err != nil {
		return "", fmt.Errorf("gh api %s: %w", endpoint, err)
	}
	return strings.Trim(strings.TrimSpace(string(out)), `"`), nil
}

// blobEntry pairs a registry file path with its blob SHA.
type blobEntry struct {
	path string
	sha  string
}

// ghAPICreateBlob creates a git blob on the configured registry and returns its SHA.
func ghAPICreateBlob(content string) (string, error) {
	api, err := activeRegistryAPI()
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	bodyBytes, err := json.Marshal(map[string]string{
		"encoding": "base64",
		"content":  encoded,
	})
	if err != nil {
		return "", err
	}
	c := exec.Command("gh", "api",
		api+"/git/blobs",
		"--method", "POST",
		"--input", "-",
		"--jq", ".sha",
	)
	c.Stdin = strings.NewReader(string(bodyBytes))
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("create blob: %w", err)
	}
	return strings.Trim(strings.TrimSpace(string(out)), `"`), nil
}

// ghAPICreateTree creates a git tree on the configured registry and returns its SHA.
func ghAPICreateTree(baseSHA string, blobs []blobEntry) (string, error) {
	api, err := activeRegistryAPI()
	if err != nil {
		return "", err
	}
	type treeItem struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
	}
	var items []treeItem
	for _, b := range blobs {
		items = append(items, treeItem{Path: b.path, Mode: "100644", Type: "blob", SHA: b.sha})
	}
	body, err := json.Marshal(map[string]interface{}{
		"base_tree": baseSHA,
		"tree":      items,
	})
	if err != nil {
		return "", err
	}
	c := exec.Command("gh", "api",
		api+"/git/trees",
		"--method", "POST",
		"--input", "-",
		"--jq", ".sha",
	)
	c.Stdin = strings.NewReader(string(body))
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("create tree: %w", err)
	}
	return strings.Trim(strings.TrimSpace(string(out)), `"`), nil
}

// ghAPICreateCommit creates a git commit and returns its SHA.
func ghAPICreateCommit(message, treeSHA, parentSHA string) (string, error) {
	api, err := activeRegistryAPI()
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(map[string]interface{}{
		"message": message,
		"tree":    treeSHA,
		"parents": []string{parentSHA},
	})
	if err != nil {
		return "", err
	}
	c := exec.Command("gh", "api",
		api+"/git/commits",
		"--method", "POST",
		"--input", "-",
		"--jq", ".sha",
	)
	c.Stdin = strings.NewReader(string(body))
	out, err := c.Output()
	if err != nil {
		return "", fmt.Errorf("create commit: %w", err)
	}
	return strings.Trim(strings.TrimSpace(string(out)), `"`), nil
}

// ghAPIBranchExists returns true if branch already exists in the registry.
func ghAPIBranchExists(branch string) bool {
	api, err := activeRegistryAPI()
	if err != nil {
		return false
	}
	out, err := exec.Command("gh", "api",
		fmt.Sprintf("%s/git/refs/heads/%s", api, branch),
		"--jq", ".ref",
	).Output()
	if err != nil {
		return false
	}
	ref := strings.Trim(strings.TrimSpace(string(out)), `"`)
	return ref != "" && ref != "null"
}

// ghAPIEnsureBranch creates branch pointing to commitSHA if it does not exist,
// or force-updates it if it does.
func ghAPIEnsureBranch(branch, commitSHA string) error {
	api, err := activeRegistryAPI()
	if err != nil {
		return err
	}
	if ghAPIBranchExists(branch) {
		body, err := json.Marshal(map[string]interface{}{
			"sha":   commitSHA,
			"force": true,
		})
		if err != nil {
			return err
		}
		c := exec.Command("gh", "api",
			fmt.Sprintf("%s/git/refs/heads/%s", api, branch),
			"--method", "PATCH",
			"--input", "-",
		)
		c.Stdin = strings.NewReader(string(body))
		if _, err := c.Output(); err != nil {
			stderr := ""
			if ee, ok := err.(*exec.ExitError); ok {
				stderr = string(ee.Stderr)
			}
			return fmt.Errorf("update branch: %s", strings.TrimSpace(stderr))
		}
		return nil
	}
	body, err := json.Marshal(map[string]interface{}{
		"ref": "refs/heads/" + branch,
		"sha": commitSHA,
	})
	if err != nil {
		return err
	}
	c := exec.Command("gh", "api",
		api+"/git/refs",
		"--method", "POST",
		"--input", "-",
	)
	c.Stdin = strings.NewReader(string(body))
	if _, err := c.Output(); err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		return fmt.Errorf("create branch: %s", strings.TrimSpace(stderr))
	}
	return nil
}

// ghAPIFindOpenPR returns the HTML URL of the first open PR targeting branch,
// or an error if none is found.
func ghAPIFindOpenPR(branch string) (string, error) {
	r, err := pushRegistry()
	if err != nil {
		return "", err
	}
	out, err := exec.Command("gh", "api",
		fmt.Sprintf("%s/pulls?head=%s:%s&state=open", r.GitHubAPIRepo(), r.Owner, branch),
		"--jq", ".[0].html_url",
	).Output()
	if err != nil {
		return "", fmt.Errorf("find open PR: %w", err)
	}
	url := strings.Trim(strings.TrimSpace(string(out)), `"`)
	if url == "" || url == "null" {
		return "", fmt.Errorf("no open PR found for branch %s", branch)
	}
	return url, nil
}

// ghAPICreatePR opens a pull request and returns its HTML URL. If a PR already
// exists for the branch it returns the URL of the existing PR instead.
func ghAPICreatePR(title, branch, body string) (string, error) {
	api, err := activeRegistryAPI()
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(map[string]interface{}{
		"title": title,
		"head":  branch,
		"base":  "main",
		"body":  body,
	})
	if err != nil {
		return "", err
	}
	c := exec.Command("gh", "api",
		api+"/pulls",
		"--method", "POST",
		"--input", "-",
		"--jq", ".html_url",
	)
	c.Stdin = strings.NewReader(string(payload))
	out, err := c.Output()
	if err != nil {
		stderr := ""
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		if strings.Contains(stderr, "A pull request already exists") || strings.Contains(string(out), "already exists") {
			return ghAPIFindOpenPR(branch)
		}
		return "", fmt.Errorf("create PR: %s", strings.TrimSpace(stderr))
	}
	return strings.Trim(strings.TrimSpace(string(out)), `"`), nil
}

// normalizeSkillName accepts copied registry paths such as
// @skills/my-skill/ while preserving bare skill names.
func normalizeSkillName(input string) string {
	name := strings.TrimSpace(input)
	name = strings.TrimPrefix(name, "@")
	name = filepath.ToSlash(strings.Trim(name, "/"))

	parts := strings.Split(name, "/")
	for i, part := range parts {
		if part == "skills" && i+1 < len(parts) && parts[i+1] != "" {
			return parts[i+1]
		}
	}

	return name
}

// printJSON marshals v to indented JSON and prints it.
func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// preflightError formats preflight failures into a single error.
func preflightError(result preflight.Result) error {
	var sb strings.Builder
	for _, f := range result.Failures {
		sb.WriteString(fmt.Sprintf("error: %s\n", f.Err))
		for _, fix := range f.Fix {
			sb.WriteString(fmt.Sprintf("  %s\n", fix))
		}
	}
	sb.WriteString("run 'hikma doctor' for details")
	return fmt.Errorf("%s", sb.String())
}

// readSkillOwner parses the YAML frontmatter of SKILL.md in skillDir and
// returns the CODEOWNERS owner string. It prefers metadata.owner and falls back
// to metadata.team. Returns an empty string without error if neither field is present.
func readSkillOwner(skillDir string) (string, error) {
	skillFile := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(skillFile)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", skillFile, err)
	}

	// Extract frontmatter between the first pair of --- delimiters.
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return "", nil
	}
	rest := content[3:]
	end := strings.Index(rest, "\n---")
	if end == -1 {
		return "", nil
	}
	frontmatter := rest[:end]

	var doc struct {
		Metadata struct {
			Owner string `yaml:"owner"`
			Team  string `yaml:"team"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &doc); err != nil {
		return "", fmt.Errorf("parse SKILL.md frontmatter: %w", err)
	}

	if doc.Metadata.Owner != "" {
		return doc.Metadata.Owner, nil
	}
	return doc.Metadata.Team, nil
}

// fetchRegistryFileContent fetches the raw text content of a file from the
// registry repo (e.g. ".github/CODEOWNERS"). Returns empty string without
// error if the file does not exist (404).
func fetchRegistryFileContent(repoPath string) (string, error) {
	api, err := activeRegistryAPI()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(
		"gh", "api",
		fmt.Sprintf("%s/contents/%s", api, repoPath),
		"--jq", ".content",
	).Output()
	if err != nil {
		// Treat 404 as an empty file rather than a hard failure.
		if strings.Contains(err.Error(), "404") {
			return "", nil
		}
		return "", fmt.Errorf("fetch %s: %w", repoPath, err)
	}
	encoded := strings.ReplaceAll(strings.TrimSpace(string(out)), "\n", "")
	encoded = strings.Trim(encoded, `"`)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode %s: %w", repoPath, err)
	}
	return string(decoded), nil
}

func activeRegistryAPI() (string, error) {
	r, err := pushRegistry()
	if err != nil {
		return "", err
	}
	return r.GitHubAPIRepo(), nil
}

func resolveSkillDir(name, flagAgent string) (string, error) {
	return resolveSkillDirForAgent(name, flagAgent)
}

// resolveSkillDirForAgent returns the existing directory for name across the
// selected agents, or the first selected agent's directory when it does not exist yet.
func resolveSkillDirForAgent(name, flagAgent string) (string, error) {
	targets, err := config.ResolveTargets(flagAgent)
	if err != nil {
		return "", err
	}
	for _, t := range targets {
		if dir := t.SkillDir(name); dirExists(dir) {
			return dir, nil
		}
	}
	return targets[0].SkillDir(name), nil
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// printActiveSelection writes the active agent and resolved paths to stdout.
func printActiveSelection(selection config.Selection, isOverride bool) {
	fmt.Printf("Active agent: %s%s\n", selection.Agent, overrideSuffix(isOverride))
	fmt.Printf("Skill path: %s\n", selection.Layout.SkillsDir)
}

// pushTarget is the registry chosen for the current push, set before any API call.
var pushTarget *config.Registry

func pushRegistry() (config.Registry, error) {
	if pushTarget != nil {
		return *pushTarget, nil
	}
	return config.ResolveRegistry("")
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

// confirmPush shows the target registry and asks the user to confirm.
func confirmPush(cmd *cobra.Command, name, skillDir string, r config.Registry, origin string, assumeYes bool) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "\nYou are pushing to registry: %s (%s)\n", r.FullName(), origin)
	fmt.Fprintf(out, "  skill: %s (%s/)\n", name, skillDir)
	if assumeYes {
		return nil
	}
	if !isInteractive() {
		return fmt.Errorf("confirmation required: re-run with --yes to push to %s", r.FullName())
	}
	var ok bool
	if err := survey.AskOne(&survey.Confirm{Message: "Push to " + r.FullName() + "?", Default: false}, &ok); err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("push cancelled")
	}
	return nil
}

// ghUser returns the login name of the authenticated GitHub user.
func ghUser() (string, error) {
	out, err := exec.Command("gh", "api", "user", "--jq", ".login").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
