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
	"github.com/hasankhatib/hikma-ai/internal/registry"
	"github.com/hasankhatib/hikma-ai/internal/scaffold"
	"github.com/hasankhatib/hikma-ai/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newSkillCmd() *cobra.Command {
	var flagRegistry string
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage skills from the configured registry",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flagRegistry == "" {
				return nil
			}
			if _, err := config.ParseRegistry(flagRegistry); err != nil {
				return err
			}
			return os.Setenv("HIKMA_REGISTRY", flagRegistry)
		},
	}
	cmd.PersistentFlags().StringVar(&flagRegistry, "registry", "", "registry to use for this command (owner/repo or GitHub URL)")

	cmd.AddCommand(
		newSkillListCmd(),
		newSkillInfoCmd(),
		newSkillInstallCmd(),
		newSkillUpdateCmd(),
		newSkillCreateCmd(),
		newSkillPushCmd(),
	)

	return cmd
}

func newSkillListCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available skills in the registry",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			result := preflight.Check()
			if !result.OK() {
				return preflightError(result)
			}

			status := io.Writer(os.Stdout)
			if asJSON {
				status = os.Stderr
			}
			skills, err := loadOrFetchIndex(status)
			if err != nil {
				return err
			}

			if asJSON {
				return printJSON(skills)
			}

			printSkillTable(skills)
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newSkillInfoCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show details for a skill",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill info <name>\n\nRun 'hikma skill list' to see available skills.")
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts exactly one skill name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := normalizeSkillName(args[0])

			result := preflight.Check()
			if !result.OK() {
				return preflightError(result)
			}

			status := io.Writer(os.Stdout)
			if asJSON {
				status = os.Stderr
			}
			skills, err := loadOrFetchIndex(status)
			if err != nil {
				return err
			}

			skill, found, err := findSkillOrRefresh(skills, name, status)
			if err != nil {
				ui.Error("refresh registry index: %s", err)
				return fmt.Errorf("index unavailable")
			}
			if !found {
				ui.UserError("skill '%s' not found in registry\n  run 'hikma skill list' to see available skills", name)
				return fmt.Errorf("skill not found")
			}

			if asJSON {
				return printJSON(skill)
			}

			printSkillDetail(skill)
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return registryCompletions()
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newSkillInstallCmd() *cobra.Command {
	var force bool
	var flagAgent string

	cmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install a skill from the registry into the active agent's skill directory",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill install <name>\n\nRun 'hikma skill list' to see available skills.")
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts exactly one skill name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			name := normalizeSkillName(args[0])

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

			// Option A: check index first for a user-friendly error message.
			skills, err := loadOrFetchIndex(os.Stdout)
			if err != nil {
				return err
			}
			if _, found, err := findSkillOrRefresh(skills, name, os.Stdout); err != nil {
				ui.Error("refresh registry index: %s", err)
				return fmt.Errorf("index unavailable")
			} else if !found {
				fmt.Println("Skill not found in index; checking registry path directly...")
			}

			targetDir, err := resolveSkillDirForAgent(name, flagAgent)
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}

			// Idempotency check.
			if _, err := os.Stat(targetDir); err == nil {
				if !force {
					ui.UserError("%s is already installed at %s/\n  use --force to reinstall", name, targetDir)
					return fmt.Errorf("already installed")
				}
			}

			fmt.Printf("Fetching %s from registry...\n", name)

			written, err := fetchAndReplaceSkill(name, targetDir)
			if err != nil {
				ui.Error("fetch %s: %s", name, err)
				return fmt.Errorf("fetch failed")
			}

			for _, p := range written {
				fmt.Printf("  -> %s/%s\n", targetDir, p)
			}

			fmt.Printf("\nOpen %s/SKILL.md to review.\n", targetDir)
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return registryCompletions()
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "reinstall even if already installed")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "AI agent (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func newSkillUpdateCmd() *cobra.Command {
	var all bool
	var force bool
	var flagAgent string

	cmd := &cobra.Command{
		Use:   "update [<name>]",
		Short: "Update installed skills to the latest registry version",
		Args: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")
			agent, _ := cmd.Flags().GetString("agent")
			if !all && len(args) == 0 {
				return fmt.Errorf("missing required argument: <name>\n\nUsage:\n  hikma skill update <name>\n  hikma skill update --all\n\nProvide a skill name or pass --all to update every installed skill.")
			}
			if all && len(args) > 0 {
				return fmt.Errorf("--all and a skill name are mutually exclusive\n\nUsage:\n  hikma skill update <name>\n  hikma skill update --all")
			}
			if all && agent != "" {
				return fmt.Errorf("--all cannot be combined with --agent")
			}
			if len(args) > 1 {
				return fmt.Errorf("accepts at most one skill name")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			result := preflight.Check()
			if !result.OK() {
				return preflightError(result)
			}

			if len(args) == 1 {
				isAgentOverride := flagAgent != ""
				selection, err := config.ResolveSelection(flagAgent)
				if err != nil {
					return fmt.Errorf("error: %w\nrun 'hikma doctor' to check your environment", err)
				}
				printActiveSelection(selection, isAgentOverride)
				return updateSkill(normalizeSkillName(args[0]), force, flagAgent)
			}

			// --all: scan both skill base directories and update each unique skill.
			basePaths := []string{
				config.SkillBasePath(config.ProfileDefault),
				config.SkillBasePath(config.ProfileClaude),
			}
			seen := map[string]bool{}
			var updated, skipped int
			for _, base := range basePaths {
				entries, err := os.ReadDir(base)
				if os.IsNotExist(err) {
					continue
				}
				if err != nil {
					return fmt.Errorf("read %s/: %w", base, err)
				}
				for _, e := range entries {
					if !e.IsDir() {
						continue
					}
					localDir := filepath.Join(base, e.Name())
					if seen[localDir] {
						continue
					}
					seen[localDir] = true
					err := updateSkillDir(e.Name(), localDir, force)
					if err != nil {
						ui.UserError("skipping %s: %s", localDir, err)
						skipped++
					} else {
						updated++
					}
				}
			}
			fmt.Printf("\nDone: %d updated, %d skipped.\n", updated, skipped)
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return installedSkillCompletions()
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "update all installed skills across all supported skill directories")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite local changes")
	cmd.Flags().StringVar(&flagAgent, "agent", "", "AI agent (copilot, codex, opencode, claude)")
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// updateSkill updates the named skill using the active agent's skill directory.
func updateSkill(name string, force bool, flags ...string) error {
	flagAgent := ""
	if len(flags) > 0 {
		flagAgent = flags[0]
	}
	localDir, err := resolveSkillDirForAgent(name, flagAgent)
	if err != nil {
		return err
	}
	return updateSkillDir(name, localDir, force)
}

// updateSkillDir performs the update flow for a skill at a known local path.
func updateSkillDir(name, localDir string, force bool) error {
	// Check that it is installed at all.
	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		ui.UserError("skill '%s' is not installed at %s/\n  run 'hikma skill install %s' to install it first", name, localDir, name)
		return fmt.Errorf("not installed")
	}

	if !force {
		changed, err := registry.DetectLocalChanges(name, localDir)
		if err != nil {
			ui.Error("detect local changes for '%s': %s", name, err)
			return fmt.Errorf("detect changes failed")
		}
		if changed {
			ui.UserError("skill '%s' has local changes\n  use --force to overwrite local changes", name)
			return fmt.Errorf("local changes detected")
		}
	}

	fmt.Printf("Updating %s...\n", name)

	written, err := fetchAndReplaceSkill(name, localDir)
	if err != nil {
		ui.Error("fetch %s: %s", name, err)
		return fmt.Errorf("fetch failed")
	}

	for _, p := range written {
		fmt.Printf("  -> %s/%s\n", localDir, p)
	}

	lastValidated := parseLastValidated(localDir)
	if lastValidated != "" {
		fmt.Printf("Updated %s (last_validated: %s).\n", name, lastValidated)
	} else {
		fmt.Printf("Updated %s.\n", name)
	}
	return nil
}

// parseLastValidated reads SKILL.md from skillDir and returns the value of the
// last_validated field from the YAML frontmatter, or an empty string if the
// field is absent or the file cannot be read.
func parseLastValidated(skillDir string) string {
	data, err := os.ReadFile(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		return ""
	}
	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}
	rest := content[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return ""
	}
	for _, line := range strings.Split(rest[:end], "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "last_validated:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "last_validated:"))
		}
	}
	return ""
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
			if !isKebabCase(name) {
				return fmt.Errorf("name must be kebab-case (lowercase letters, digits, hyphens): got %q", name)
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

func newSkillPushCmd() *cobra.Command {
	var flagAgent string

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
			activeRegistry, err := config.ResolveRegistry("")
			if err != nil {
				return err
			}

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

			// Detect whether this is a first push or an update.
			branch := fmt.Sprintf("skill/%s", name)
			isUpdate := ghAPIBranchExists(branch)
			action := "add"
			if isUpdate {
				action = "update"
			}

			// Step 1: get HEAD SHA of main.
			if isUpdate {
				fmt.Println("Updating branch " + branch + " on " + activeRegistry.FullName() + "...")
			} else {
				fmt.Println("Creating branch " + branch + " on " + activeRegistry.FullName() + "...")
			}
			headSHA, err := ghAPIString(
				activeRegistry.GitHubAPIRepo()+"/git/refs/heads/main",
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

			// Step 2b: update CODEOWNERS with the skill owner when present.
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
	_ = cmd.RegisterFlagCompletionFunc("agent", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"copilot", "codex", "opencode", "claude"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

// registryCompletions returns skill names (with descriptions) from the local
// registry cache for shell completion. Falls back to no completions on error.
func registryCompletions() ([]string, cobra.ShellCompDirective) {
	skills, fresh, err := registry.LoadCachedIndex()
	if err != nil || !fresh {
		// If cache is stale or missing, try a fresh fetch - but don't block the
		// completion on network latency. Return empty on any error.
		var fetchErr error
		skills, fetchErr = registry.FetchIndex()
		if fetchErr != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
	}
	names := make([]string, len(skills))
	for i, s := range skills {
		names[i] = s.Name + "\t" + s.Description
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// installedSkillCompletions returns de-duplicated skill names from both supported skill directories.
func installedSkillCompletions() ([]string, cobra.ShellCompDirective) {
	seen := make(map[string]bool)
	for _, base := range []string{config.SkillBasePath(config.ProfileDefault), config.SkillBasePath(config.ProfileClaude)} {
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				seen[e.Name()] = true
			}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, cobra.ShellCompDirectiveNoFileComp
}

// isKebabCase returns true if s contains only lowercase letters, digits, and hyphens.
func isKebabCase(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return true
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
	r, err := config.ResolveRegistry("")
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

// loadOrFetchIndex returns the skill index from cache if fresh, or fetches and caches it.
func loadOrFetchIndex(status io.Writer) ([]registry.Skill, error) {
	skills, fresh, err := registry.LoadCachedIndex()
	if err != nil {
		ui.Error("load cached index: %s", err)
		return nil, fmt.Errorf("index unavailable")
	}
	if fresh {
		return skills, nil
	}

	fmt.Fprintln(status, "Fetching registry index...")
	skills, err = fetchIndex()
	if err != nil {
		ui.Error("fetch registry index: %s", err)
		return nil, fmt.Errorf("index unavailable")
	}

	// Cache best-effort - don't fail the command if caching fails.
	_ = registry.CacheIndex(skills)
	return skills, nil
}

var (
	fetchIndex = registry.FetchIndex
	fetchSkill = registry.FetchSkill
)

// fetchAndReplaceSkill downloads into a sibling staging directory before replacing targetDir.
func fetchAndReplaceSkill(name, targetDir string) ([]string, error) {
	parent := filepath.Dir(targetDir)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return nil, fmt.Errorf("create skill parent directory: %w", err)
	}
	stagingDir, err := os.MkdirTemp(parent, "."+filepath.Base(targetDir)+"-staging-")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	written, err := fetchSkill(name, stagingDir)
	if err != nil {
		return nil, err
	}

	backupDir := stagingDir + "-backup"
	targetExists := false
	if _, err := os.Stat(targetDir); err == nil {
		targetExists = true
		if err := os.Rename(targetDir, backupDir); err != nil {
			return nil, fmt.Errorf("stage existing skill directory: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect existing skill directory: %w", err)
	}

	if err := os.Rename(stagingDir, targetDir); err != nil {
		if targetExists {
			if restoreErr := os.Rename(backupDir, targetDir); restoreErr != nil {
				return nil, fmt.Errorf("replace skill directory: %w (restore failed: %v)", err, restoreErr)
			}
		}
		return nil, fmt.Errorf("replace skill directory: %w", err)
	}
	if targetExists {
		if err := os.RemoveAll(backupDir); err != nil {
			return nil, fmt.Errorf("remove previous skill directory: %w", err)
		}
	}
	return written, nil
}

// findSkill returns the skill with the given name and true, or zero value and false.
func findSkill(skills []registry.Skill, name string) (registry.Skill, bool) {
	for _, s := range skills {
		if s.Name == name {
			return s, true
		}
	}
	return registry.Skill{}, false
}

func findSkillOrRefresh(skills []registry.Skill, name string, status io.Writer) (registry.Skill, bool, error) {
	if skill, found := findSkill(skills, name); found {
		return skill, true, nil
	}

	fmt.Fprintln(status, "Refreshing registry index...")
	refreshed, err := fetchIndex()
	if err != nil {
		return registry.Skill{}, false, err
	}
	_ = registry.CacheIndex(refreshed)

	skill, found := findSkill(refreshed, name)
	return skill, found, nil
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

// printSkillTable prints skills as an aligned table.
func printSkillTable(skills []registry.Skill) {
	if len(skills) == 0 {
		fmt.Println("No skills found in registry.")
		return
	}

	const nameW = 25
	const descW = 50

	fmt.Printf("%-*s  %-*s  %s\n", nameW, "NAME", descW, "DESCRIPTION", "OWNER")
	for _, s := range skills {
		desc := s.Description
		if len(desc) > descW {
			desc = desc[:descW-3] + "..."
		}
		fmt.Printf("%-*s  %-*s  %s\n", nameW, s.Name, descW, desc, s.Owner)
	}
}

// printSkillDetail prints the full metadata block for one skill.
func printSkillDetail(s registry.Skill) {
	fmt.Printf("Name:           %s\n", s.Name)
	fmt.Printf("Description:    %s\n", s.Description)
	fmt.Printf("Owner:          %s\n", s.Owner)
	if s.Email != "" {
		fmt.Printf("Email:          %s\n", s.Email)
	}
	fmt.Printf("Last validated: %s\n", s.LastValidated)
	fmt.Printf("Compatibility:  %s\n", s.Compatibility)
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
	r, err := config.ResolveRegistry("")
	if err != nil {
		return "", err
	}
	return r.GitHubAPIRepo(), nil
}

func resolveSkillDir(name, flagAgent string) (string, error) {
	return resolveSkillDirForAgent(name, flagAgent)
}

func resolveSkillDirForAgent(name, flagAgent string) (string, error) {
	selection, err := config.ResolveSelection(flagAgent)
	if err != nil {
		return "", err
	}
	return config.SkillDir(name, selection.Profile), nil
}

// printActiveSelection writes the active agent and resolved paths to stdout.
func printActiveSelection(selection config.Selection, isOverride bool) {
	suffix := ""
	if isOverride {
		suffix = " (override)"
	}
	fmt.Printf("Active agent: %s%s\n", selection.Agent, suffix)
	fmt.Printf("Skill path: %s\n", config.SkillBasePath(selection.Profile))
}

func scaffoldOutput(profile config.Profile) string {
	if profile == config.ProfileClaude {
		return "AGENTS.md, CLAUDE.md"
	}
	return "AGENTS.md"
}
