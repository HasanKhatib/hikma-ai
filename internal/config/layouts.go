package config

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// CustomLayout is a user-defined agent layout from the `layouts` config key.
// Both paths are slash-separated and relative to the project root.
type CustomLayout struct {
	SkillsDir       string `json:"skillsDir"`
	InstructionFile string `json:"instructionFile,omitempty"`
}

var layoutNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidateLayouts checks every custom layout. Config files are committed, so
// they are untrusted input: a layout must never point outside the project or
// into the repository's own metadata.
func ValidateLayouts(layouts map[string]CustomLayout) error {
	names := make([]string, 0, len(layouts))
	for name := range layouts {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		l := layouts[name]
		if !layoutNameRE.MatchString(name) {
			return fmt.Errorf("layouts.%s: name must be lowercase letters, digits, and hyphens", name)
		}
		if ValidAgent(Agent(name)) {
			return fmt.Errorf("layouts.%s: name is a built-in agent; choose another name", name)
		}
		if l.SkillsDir == "" {
			return fmt.Errorf("layouts.%s.skillsDir is required", name)
		}
		if err := checkLayoutPath(l.SkillsDir); err != nil {
			return fmt.Errorf("layouts.%s.skillsDir: %w", name, err)
		}
		if l.InstructionFile != "" {
			if err := checkLayoutPath(l.InstructionFile); err != nil {
				return fmt.Errorf("layouts.%s.instructionFile: %w", name, err)
			}
		}
	}
	return nil
}

// checkLayoutPath requires a clean, slash-separated path that stays inside the
// project and avoids .git and .hikma at any depth. The comparison ignores case
// and trailing dots or spaces because macOS and Windows resolve those to the
// same folder.
func checkLayoutPath(p string) error {
	if strings.ContainsAny(p, "\\\x00:") {
		return fmt.Errorf("%q must use forward slashes and no drive letter", p)
	}
	if strings.HasPrefix(p, "/") {
		return fmt.Errorf("%q must be relative", p)
	}
	if p == "." || path.Clean(p) != p {
		return fmt.Errorf("%q must be a clean relative path", p)
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return fmt.Errorf("%q must not contain ..", p)
		}
		name := strings.ToLower(strings.TrimRight(seg, ". "))
		if name == "" {
			return fmt.Errorf("%q has an invalid segment %q", p, seg)
		}
		if name == ".git" || name == ".hikma" {
			return fmt.Errorf("%q must not be inside %s", p, seg)
		}
	}
	return nil
}
