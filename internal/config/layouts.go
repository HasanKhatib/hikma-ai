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

// LayoutSet is every layout available to a command: the built-in agents plus
// the custom layouts from user and project config.
type LayoutSet struct {
	custom map[Agent]Layout
	names  []Agent // custom names, sorted
}

// Layouts resolves the layout set. A project layout overrides a user layout
// with the same name. Invalid layouts in either file are an error.
func Layouts() (LayoutSet, error) {
	user, err := Load()
	if err != nil {
		return LayoutSet{}, err
	}
	proj, err := LoadProject()
	if err != nil {
		return LayoutSet{}, err
	}
	set := LayoutSet{custom: map[Agent]Layout{}}
	for _, src := range []map[string]CustomLayout{user.Layouts, proj.Layouts} {
		for name, l := range src {
			set.custom[Agent(name)] = customToLayout(l)
		}
	}
	for a := range set.custom {
		set.names = append(set.names, a)
	}
	sort.Slice(set.names, func(i, j int) bool { return set.names[i] < set.names[j] })
	return set, nil
}

func customToLayout(l CustomLayout) Layout {
	out := Layout{Description: "custom layout using " + l.SkillsDir, SkillsDir: l.SkillsDir}
	if l.InstructionFile != "" {
		out.InstructionFiles = []string{l.InstructionFile}
	}
	return out
}

// Valid reports whether a is a built-in agent or a configured layout.
func (s LayoutSet) Valid(a Agent) bool {
	if ValidAgent(a) {
		return true
	}
	_, ok := s.custom[a]
	return ok
}

// Layout returns the layout for a, which must be valid.
func (s LayoutSet) Layout(a Agent) Layout {
	if l, ok := s.custom[a]; ok {
		return l
	}
	return LayoutFor(a)
}

// Selection builds the selection for a valid agent.
func (s LayoutSet) Selection(a Agent) Selection { return Selection{Agent: a, Layout: s.Layout(a)} }

// Names lists built-in agents in display order, then custom layouts by name.
func (s LayoutSet) Names() []Agent {
	return append(append([]Agent{}, ValidAgents...), s.names...)
}

// SkillsDirs returns every skills folder in the set.
func (s LayoutSet) SkillsDirs() map[string]bool {
	dirs := map[string]bool{}
	for _, a := range s.Names() {
		dirs[s.Layout(a).SkillsDir] = true
	}
	return dirs
}

// InstructionFiles returns the de-duplicated instruction files for the agents.
func (s LayoutSet) InstructionFiles(agents []Agent) []string {
	var files []string
	seen := map[string]bool{}
	for _, a := range agents {
		for _, f := range s.Layout(a).InstructionFiles {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	return files
}

func (s LayoutSet) list() string {
	names := make([]string, 0, len(s.names)+len(ValidAgents))
	for _, a := range s.Names() {
		names = append(names, string(a))
	}
	return strings.Join(names, ", ")
}

// Parse splits a comma-separated agent list and validates every entry.
func (s LayoutSet) Parse(value string) ([]Agent, error) {
	var agents []Agent
	seen := map[Agent]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		a := Agent(part)
		if !s.Valid(a) {
			return nil, fmt.Errorf("unsupported agent %q - supported values: %s", part, s.list())
		}
		if !seen[a] {
			seen[a] = true
			agents = append(agents, a)
		}
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents given - supported values: %s", s.list())
	}
	return agents, nil
}

// UniqueByDir keeps the first selection for each skills directory, since
// several agents share .agents/skills.
func (s LayoutSet) UniqueByDir(agents []Agent) []Selection {
	var out []Selection
	seen := map[string]bool{}
	for _, a := range agents {
		sel := s.Selection(a)
		if seen[sel.Layout.SkillsDir] {
			continue
		}
		seen[sel.Layout.SkillsDir] = true
		out = append(out, sel)
	}
	return out
}
