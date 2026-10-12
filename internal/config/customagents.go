package config

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
)

// CustomAgent is a user-defined agent from the `customAgents` config key.
// Both paths are slash-separated and relative to the project root.
type CustomAgent struct {
	SkillsDir       string `json:"skillsDir"`
	InstructionFile string `json:"instructionFile,omitempty"`
}

var layoutNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// ValidateCustomAgents checks every custom agent. Config files are committed, so
// they are untrusted input: a definition must never point outside the project or
// into the repository's own metadata.
func ValidateCustomAgents(agents map[string]CustomAgent) error {
	names := make([]string, 0, len(agents))
	for name := range agents {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		l := agents[name]
		if !layoutNameRE.MatchString(name) {
			return fmt.Errorf("customAgents.%s: name must be lowercase letters, digits, and hyphens", name)
		}
		if ValidAgent(Agent(name)) {
			return fmt.Errorf("customAgents.%s: name is a built-in agent; choose another name", name)
		}
		if l.SkillsDir == "" {
			return fmt.Errorf("customAgents.%s.skillsDir is required", name)
		}
		if err := checkLayoutPath(l.SkillsDir); err != nil {
			return fmt.Errorf("customAgents.%s.skillsDir: %w", name, err)
		}
		if l.InstructionFile != "" {
			if err := checkLayoutPath(l.InstructionFile); err != nil {
				return fmt.Errorf("customAgents.%s.instructionFile: %w", name, err)
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
// the custom agents from user and project config.
type LayoutSet struct {
	custom map[Agent]Layout
	names  []Agent // custom names, sorted
}

// Layouts resolves the layout set. A project definition overrides a user
// definition with the same name. An invalid definition in either file is an error.
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
	for _, src := range []map[string]CustomAgent{user.CustomAgents, proj.CustomAgents} {
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

func customToLayout(l CustomAgent) Layout {
	out := Layout{Description: "custom agent using " + l.SkillsDir, SkillsDir: l.SkillsDir}
	if l.InstructionFile != "" {
		out.InstructionFiles = []string{l.InstructionFile}
	}
	return out
}

// Valid reports whether a is a built-in agent or a configured custom agent.
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

// Names lists built-in agents in display order, then custom agents by name.
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
	index := map[string]int{}
	for _, a := range agents {
		sel := s.Selection(a)
		sel.Reads = append([]string(nil), sel.Layout.AlsoReads...)
		if i, seen := index[sel.Layout.SkillsDir]; seen {
			out[i].Reads = append(out[i].Reads, sel.Reads...)
			continue
		}
		index[sel.Layout.SkillsDir] = len(out)
		out = append(out, sel)
	}
	return out
}

// ValidateCustomAgent checks one definition, with the same rules as on load.
func ValidateCustomAgent(name string, a CustomAgent) error {
	return ValidateCustomAgents(map[string]CustomAgent{name: a})
}

// ScopePath returns the config file a write for scope targets.
func ScopePath(scope Scope) (string, error) { return scopePath(scope) }

// CustomAgentEntry is a configured custom agent and where it is defined.
type CustomAgentEntry struct {
	Name   string
	Agent  CustomAgent
	Source Source
}

// ListCustomAgents returns the custom agents in effect, by name. A project
// definition hides a user definition with the same name.
func ListCustomAgents() ([]CustomAgentEntry, error) {
	user, err := Load()
	if err != nil {
		return nil, err
	}
	proj, err := LoadProject()
	if err != nil {
		return nil, err
	}
	byName := map[string]CustomAgentEntry{}
	for name, a := range user.CustomAgents {
		byName[name] = CustomAgentEntry{name, a, SourceUser}
	}
	for name, a := range proj.CustomAgents {
		byName[name] = CustomAgentEntry{name, a, SourceProject}
	}
	out := make([]CustomAgentEntry, 0, len(byName))
	for _, e := range byName {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// CustomAgentIn returns the definition of name in the config file for scope.
func CustomAgentIn(scope Scope, name string) (CustomAgent, bool, error) {
	path, err := scopePath(scope)
	if err != nil {
		return CustomAgent{}, false, err
	}
	c, err := loadFile(path)
	if err != nil {
		return CustomAgent{}, false, err
	}
	a, ok := c.CustomAgents[name]
	return a, ok, nil
}

// AddCustomAgent validates and writes a definition to the config file for
// scope, and returns that file's path. It refuses to replace an existing
// definition unless force is set. With use set it also adds name to the
// `agents` list in the same file.
func AddCustomAgent(scope Scope, name string, a CustomAgent, force, use bool) (string, error) {
	if err := ValidateCustomAgent(name, a); err != nil {
		return "", err
	}
	path, err := scopePath(scope)
	if err != nil {
		return "", err
	}
	c, err := loadFile(path)
	if err != nil {
		return "", err
	}
	if _, exists := c.CustomAgents[name]; exists && !force {
		return "", fmt.Errorf("custom agent %q is already defined in %s; use --force to replace it", name, path)
	}
	if c.CustomAgents == nil {
		c.CustomAgents = map[string]CustomAgent{}
	}
	c.CustomAgents[name] = a
	if use {
		present := false
		for _, existing := range c.Agents {
			if existing == name {
				present = true
			}
		}
		if !present {
			c.Agents = append(c.Agents, name)
		}
	}
	return path, saveFile(path, c)
}

// RemoveCustomAgent deletes a definition from the config file for scope. It
// also drops name from the `agents` list and clears `agent` when it names it,
// and returns what it removed. An emptied config file is deleted.
func RemoveCustomAgent(scope Scope, name string) (path string, removed []string, err error) {
	path, err = scopePath(scope)
	if err != nil {
		return "", nil, err
	}
	c, err := loadFile(path)
	if err != nil {
		return "", nil, err
	}
	if _, ok := c.CustomAgents[name]; !ok {
		return path, nil, fmt.Errorf("custom agent %q is not defined in %s", name, path)
	}
	delete(c.CustomAgents, name)
	removed = append(removed, "customAgents."+name)
	var kept []string
	for _, a := range c.Agents {
		if a == name {
			removed = append(removed, "agents entry "+name)
			continue
		}
		kept = append(kept, a)
	}
	c.Agents = kept
	if string(c.Agent) == name {
		c.Agent = ""
		removed = append(removed, "agent")
	}
	if c.isEmpty() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return path, removed, fmt.Errorf("cannot remove empty config file: %w", err)
		}
		return path, removed, nil
	}
	return path, removed, saveFile(path, c)
}
