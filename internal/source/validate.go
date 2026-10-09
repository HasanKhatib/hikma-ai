package source

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

var specName = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// specFields are the frontmatter keys defined by the Agent Skills spec.
// Other keys work in some agents (Claude Code adds several) but claude.ai
// uploads and the Skills API reject them.
var specFields = map[string]bool{
	"name": true, "description": true, "license": true,
	"compatibility": true, "metadata": true, "allowed-tools": true,
}

func warnf(format string, a ...any) Issue {
	return Issue{Severity: SeverityWarning, Message: fmt.Sprintf(format, a...)}
}

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
	issues = append(issues, checkSpec(string(data), fm)...)
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

// checkSpec applies the Agent Skills format rules (agentskills.io/specification)
// beyond presence of name and description.
func checkSpec(content string, fm frontmatter) []Issue {
	var issues []Issue
	if n := strings.TrimSpace(fm.Name); n != "" {
		if utf8.RuneCountInString(n) > 64 {
			issues = append(issues, errorf("name is longer than 64 characters"))
		}
		if !specName.MatchString(n) {
			issues = append(issues, warnf("name %q is outside the Agent Skills spec (lowercase letters, digits, single hyphens); some agents reject it. Use `hikma config set naming kebab-case` to enforce it", n))
		}
	}
	if utf8.RuneCountInString(strings.TrimSpace(fm.Description)) > 1024 {
		issues = append(issues, errorf("description is longer than 1024 characters"))
	}

	var raw map[string]any
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if end := strings.Index(content[4:], "\n---"); end >= 0 {
		_ = yaml.Unmarshal([]byte(content[4:4+end]), &raw)
	}
	if c, ok := raw["compatibility"]; ok {
		if cs, isStr := c.(string); !isStr || cs == "" || utf8.RuneCountInString(cs) > 500 {
			issues = append(issues, errorf("compatibility must be a string of 1-500 characters"))
		}
	}
	if m, ok := raw["metadata"]; ok {
		mm, isMap := m.(map[string]any)
		if !isMap {
			issues = append(issues, errorf("metadata must be a map of string keys to string values"))
		}
		for k, v := range mm {
			if _, isStr := v.(string); !isStr {
				issues = append(issues, errorf("metadata.%s must be a string (quote numbers, dates, and booleans)", k))
			}
		}
	}
	var extra []string
	for k := range raw {
		if !specFields[k] {
			extra = append(extra, k)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		issues = append(issues, warnf("fields outside the Agent Skills spec: %s (claude.ai uploads and the Skills API reject them)", strings.Join(extra, ", ")))
	}
	if lines := strings.Count(content, "\n") + 1; lines > 500 {
		issues = append(issues, warnf("SKILL.md is %d lines; the spec recommends under 500 (move detail into references/)", lines))
	}
	return issues
}
