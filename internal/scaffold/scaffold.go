// Package scaffold handles template copying and variable interpolation for
// hikma scaffold and skill create operations.
package scaffold

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hasankhatib/hikma-ai/internal/config"
)

// FS holds all templates embedded at compile time.
// Referenced as templates/scaffold/** and templates/skill/** within the FS.
// The "all:" prefix is required to include hidden directories (e.g. .agents/).
//
//go:embed all:templates
var FS embed.FS

// Options controls what scaffold writes and where.
type Options struct {
	ProjectName string
	Owner       string
	Technology  string
	Agents      []config.Agent // agents to write instruction files for
	Registry    string         // optional; mentioned in the generated instructions
	Path        string
	Force       bool
	DryRun      bool
}

// Result holds the outcome of a scaffold run.
type Result struct {
	Created     []string
	Overwritten []string
	Skipped     []string
}

// Scaffold writes the instruction files for opts.Agents into opts.Path. Each
// agent contributes the files named in its layout; shared files such as
// AGENTS.md are written once. With DryRun set nothing is written and
// Result.Created lists what would have been.
func Scaffold(opts Options) (Result, error) {
	var result Result

	if len(opts.Agents) == 0 {
		return result, fmt.Errorf("no agent selected")
	}
	for _, a := range opts.Agents {
		if !config.ValidAgent(a) {
			return result, fmt.Errorf("unsupported agent %q", a)
		}
	}

	var skillPaths []string
	seen := map[string]bool{}
	for _, sel := range uniqueSelections(opts.Agents) {
		if !seen[sel.Layout.SkillsDir] {
			seen[sel.Layout.SkillsDir] = true
			skillPaths = append(skillPaths, sel.Layout.SkillsDir)
		}
	}

	data := struct {
		ProjectName string
		Owner       string
		Technology  string
		Registry    string
		SkillPaths  string
	}{
		ProjectName: opts.ProjectName,
		Owner:       opts.Owner,
		Technology:  opts.Technology,
		Registry:    opts.Registry,
		SkillPaths:  "`" + strings.Join(skillPaths, "`, `") + "`",
	}

	for _, name := range config.InstructionFiles(opts.Agents) {
		dst := filepath.Join(opts.Path, name)
		if opts.DryRun {
			result.Created = append(result.Created, dst)
			continue
		}
		if err := writeTemplateFile("templates/scaffold/"+name, dst, data, opts.Force, &result); err != nil {
			return result, err
		}
	}
	return result, nil
}

func uniqueSelections(agents []config.Agent) []config.Selection {
	sels := make([]config.Selection, len(agents))
	for i, a := range agents {
		sels[i] = config.SelectionFor(a)
	}
	return sels
}

// walkAndWrite recursively writes all files from srcRoot (in embed.FS) into dstRoot.
func walkAndWrite(srcRoot, dstRoot string, data interface{}, force bool, result *Result) error {
	return fs.WalkDir(FS, srcRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(dstRoot, rel)

		if d.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		return writeTemplateFile(path, dst, data, force, result)
	})
}

// writeTemplateFile reads a template from the embedded FS, interpolates data, and writes to dst.
func writeTemplateFile(src, dst string, data interface{}, force bool, result *Result) error {
	existed := false
	if _, err := os.Stat(dst); err == nil {
		existed = true
	}

	if existed && !force {
		result.Skipped = append(result.Skipped, dst)
		return nil
	}

	raw, err := FS.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read template %s: %w", src, err)
	}

	tmpl, err := template.New(filepath.Base(src)).Parse(string(raw))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", src, err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create directory for %s: %w", dst, err)
	}

	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create file %s: %w", dst, err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("execute template %s: %w", src, err)
	}

	if existed {
		result.Overwritten = append(result.Overwritten, dst)
	} else {
		result.Created = append(result.Created, dst)
	}
	return nil
}

// SkillOptions controls what CreateSkill writes and where.
type SkillOptions struct {
	SkillName   string
	Owner       string
	Email       string
	Team        string
	Description string
	Date        string
	TargetDir   string
}

// CreateSkill writes a new skill directory from the embedded skill template
// into opts.TargetDir, interpolating the provided variables.
// It returns the list of files created.
func CreateSkill(opts SkillOptions) ([]string, error) {
	data := struct {
		SkillName   string
		Owner       string
		Email       string
		Team        string
		Description string
		Date        string
	}{
		SkillName:   opts.SkillName,
		Owner:       opts.Owner,
		Email:       opts.Email,
		Team:        opts.Team,
		Description: opts.Description,
		Date:        opts.Date,
	}

	var result Result
	if err := walkAndWrite("templates/skill", opts.TargetDir, data, false, &result); err != nil {
		return nil, err
	}

	return result.Created, nil
}

// ValidatePlaceholders checks that no unresolved template token or generated
// replacement instruction remains under skillDir. It returns a list of
// "file:line: text" strings, one per occurrence.
func ValidatePlaceholders(skillDir string) ([]string, error) {
	var violations []string
	markers := []string{
		"{{.",
		"Replace this description.",
		"Describe required tools",
		"Replace with scenarios and keywords",
		"Replace with clear steps for the agent.",
		"What this skill must not do",
	}

	err := filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			for _, marker := range markers {
				if !strings.Contains(line, marker) {
					continue
				}
				rel, _ := filepath.Rel(skillDir, path)
				violations = append(violations, fmt.Sprintf("%s:%d: %s", rel, i+1, strings.TrimSpace(line)))
				break
			}
		}
		return nil
	})

	return violations, err
}
