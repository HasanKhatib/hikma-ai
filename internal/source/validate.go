package source

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Severity of a validation issue.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

// Issue is one problem found while validating a skill.
type Issue struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

func errorf(format string, a ...any) Issue {
	return Issue{Severity: SeverityError, Message: fmt.Sprintf(format, a...)}
}

// CheckSkill validates one skill directory against the Agent Skills format.
// dirName is the folder name the skill is published under; a root-level skill
// (a repo with a single SKILL.md) passes isRoot and skips the name match.
func CheckSkill(dir, dirName string, isRoot bool) []Issue {
	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		return []Issue{errorf("SKILL.md is missing")}
	}
	fm, ok := parseFrontmatter(string(data))
	if !ok {
		return []Issue{errorf("SKILL.md must start with YAML frontmatter between --- lines")}
	}
	var issues []Issue
	switch {
	case strings.TrimSpace(fm.Name) == "":
		issues = append(issues, errorf("frontmatter is missing `name`"))
	case !isRoot && fm.Name != dirName:
		issues = append(issues, errorf("frontmatter name %q does not match folder %q", fm.Name, dirName))
	}
	if strings.TrimSpace(fm.Description) == "" {
		issues = append(issues, errorf("frontmatter is missing `description`"))
	}
	symlinks := 0
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.Type()&fs.ModeSymlink != 0 {
			symlinks++
		}
		return nil
	})
	if symlinks > 0 {
		issues = append(issues, Issue{Severity: SeverityWarning, Message: fmt.Sprintf("%d symlink(s) are ignored when the skill is installed or pushed", symlinks)})
	}
	return issues
}

// HasErrors reports whether any issue is an error.
func HasErrors(issues []Issue) bool {
	for _, i := range issues {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}
