package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Agent represents the user-facing AI agent selection.
type Agent string

const (
	AgentCopilot  Agent = "copilot"
	AgentCodex    Agent = "codex"
	AgentOpenCode Agent = "opencode"
	AgentClaude   Agent = "claude"
)

// Layout says where an agent reads skills and instructions from. Supporting a
// new agent means adding one entry to the layouts table (and, for a new
// instruction file, one template under internal/scaffold/templates/scaffold).
type Layout struct {
	Description      string
	SkillsDir        string   // slash-separated, relative to the project root
	InstructionFiles []string // files written by hikma init, in templates/scaffold
}

// ValidAgents lists all recognised agents in display order.
var ValidAgents = []Agent{AgentCopilot, AgentCodex, AgentOpenCode, AgentClaude}

// Skill and instruction locations follow each tool's own documentation:
// Claude Code reads .claude/skills; Codex, GitHub Copilot, and OpenCode all read
// the shared .agents/skills location (and AGENTS.md).
var layouts = map[Agent]Layout{
	AgentCopilot:  {"GitHub Copilot using AGENTS.md and .agents/skills", ".agents/skills", []string{"AGENTS.md"}},
	AgentCodex:    {"Codex CLI using AGENTS.md and .agents/skills", ".agents/skills", []string{"AGENTS.md"}},
	AgentOpenCode: {"OpenCode using AGENTS.md and .agents/skills", ".agents/skills", []string{"AGENTS.md"}},
	AgentClaude:   {"Claude Code using CLAUDE.md and .claude/skills", ".claude/skills", []string{"AGENTS.md", "CLAUDE.md"}},
}

// LayoutFor returns the layout for a, which must be a valid agent.
func LayoutFor(a Agent) Layout { return layouts[a] }

// Selection is a resolved agent and its layout.
type Selection struct {
	Agent  Agent
	Layout Layout
}

// SelectionFor builds the selection for a valid agent.
func SelectionFor(a Agent) Selection { return Selection{Agent: a, Layout: LayoutFor(a)} }

// SkillDir returns where the named skill lives for this agent.
func (s Selection) SkillDir(name string) string {
	return filepath.Join(filepath.FromSlash(s.Layout.SkillsDir), name)
}

// ValidAgent reports whether a is a supported agent.
func ValidAgent(a Agent) bool {
	_, ok := layouts[a]
	return ok
}

func agentList() string {
	names := make([]string, len(ValidAgents))
	for i, a := range ValidAgents {
		names[i] = string(a)
	}
	return strings.Join(names, ", ")
}

// ParseAgents splits a comma-separated agent list and validates every entry.
func ParseAgents(value string) ([]Agent, error) {
	var agents []Agent
	seen := map[Agent]bool{}
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		a := Agent(part)
		if !ValidAgent(a) {
			return nil, fmt.Errorf("unsupported agent %q - supported values: %s", part, agentList())
		}
		if !seen[a] {
			seen[a] = true
			agents = append(agents, a)
		}
	}
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents given - supported values: %s", agentList())
	}
	return agents, nil
}

// uniqueByDir keeps the first selection for each skills directory, since
// several agents share .agents/skills.
func uniqueByDir(agents []Agent) []Selection {
	var out []Selection
	seen := map[string]bool{}
	for _, a := range agents {
		sel := SelectionFor(a)
		if seen[sel.Layout.SkillsDir] {
			continue
		}
		seen[sel.Layout.SkillsDir] = true
		out = append(out, sel)
	}
	return out
}

// ResolveTargets returns every agent selection a command should act on, one
// per distinct skills directory. Order of precedence: the --agent flag (a
// comma-separated list), HIKMA_AGENT, the `agents` list from project or user
// config, then the single `agent` setting.
func ResolveTargets(flagAgent string) ([]Selection, error) {
	if flagAgent != "" {
		agents, err := ParseAgents(flagAgent)
		if err != nil {
			return nil, err
		}
		return uniqueByDir(agents), nil
	}
	if env := envValue(KeyAgent); env != "" {
		agents, err := ParseAgents(env)
		if err != nil {
			return nil, err
		}
		return uniqueByDir(agents), nil
	}
	list, err := Lookup(KeyAgents)
	if err != nil {
		return nil, err
	}
	if list.Value != "" {
		agents, err := ParseAgents(list.Value)
		if err != nil {
			return nil, fmt.Errorf("%w (from %s config; run 'hikma config set agents <list>')", err, list.Source)
		}
		return uniqueByDir(agents), nil
	}
	sel, err := ResolveSelection("")
	if err != nil {
		return nil, err
	}
	return []Selection{sel}, nil
}

// InstructionFiles returns the de-duplicated instruction files for the given agents.
func InstructionFiles(agents []Agent) []string {
	var files []string
	seen := map[string]bool{}
	for _, a := range agents {
		for _, f := range LayoutFor(a).InstructionFiles {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	return files
}
