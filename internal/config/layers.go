package config

import (
	"encoding/json"
	"fmt"
	"github.com/hasankhatib/hikma-ai/internal/source"
	"os"
	"path/filepath"
	"strings"
)

// Source says which configuration layer a value came from.
type Source string

const (
	SourceFlag    Source = "flag"
	SourceEnv     Source = "env"
	SourceProject Source = "project"
	SourceUser    Source = "user"
	SourceDefault Source = "default"
)

// Scope selects which config file a write targets.
type Scope string

const (
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
)

// Config keys understood by hikma.
const (
	KeyAgent    = "agent"
	KeyAgents   = "agents"
	KeyRegistry = "registry"
	KeyNaming   = "naming"
)

// Naming conventions for skill names.
const (
	NamingLoose     = "loose"
	NamingKebabCase = "kebab-case"
)

// Keys lists every supported config key in display order.
var Keys = []string{KeyAgent, KeyAgents, KeyRegistry, KeyNaming}

var envVars = map[string]string{
	KeyAgent:    "HIKMA_AGENT",
	KeyAgents:   "HIKMA_AGENTS",
	KeyRegistry: "HIKMA_REGISTRY",
	KeyNaming:   "HIKMA_NAMING",
}

// Entry is a resolved config value and the layer it came from.
type Entry struct {
	Key    string
	Value  string
	Source Source
}

// EnvVar returns the environment variable that overrides key.
func EnvVar(key string) string { return envVars[key] }

func envValue(key string) string { return os.Getenv(envVars[key]) }

// ProjectConfigPath returns the nearest .hikma/config.json at or above the
// working directory, stopping at the repository root or the home directory. It returns "" when none exists.
func ProjectConfigPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	home, _ := os.UserHomeDir()
	for {
		if home != "" && dir == home {
			return "", nil
		}
		candidate := filepath.Join(dir, ".hikma", "config.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return "", nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// ProjectWritePath is where `config set --project` writes: the existing project
// config if one is found, otherwise .hikma/config.json in the working directory.
func ProjectWritePath() (string, error) {
	if p, err := ProjectConfigPath(); err != nil || p != "" {
		return p, err
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("cannot determine working directory: %w", err)
	}
	return filepath.Join(dir, ".hikma", "config.json"), nil
}

// LoadProject reads the project config. It returns an empty Config when none exists.
func LoadProject() (Config, error) {
	path, err := ProjectConfigPath()
	if err != nil || path == "" {
		return Config{}, err
	}
	return loadFile(path)
}

func loadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, fmt.Errorf("cannot read config file %s: %w", path, err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("cannot parse config file %s: %w", path, err)
	}
	return c, nil
}

func saveFile(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}
	return nil
}

func (c Config) get(key string) string {
	switch key {
	case KeyAgent:
		return string(c.Agent)
	case KeyAgents:
		return strings.Join(c.Agents, ",")
	case KeyRegistry:
		return c.Registry
	case KeyNaming:
		return c.Naming
	}
	return ""
}

func (c *Config) put(key, value string) {
	switch key {
	case KeyAgent:
		c.Agent = Agent(value)
	case KeyAgents:
		c.Agents = nil
		for _, a := range strings.Split(value, ",") {
			if a = strings.TrimSpace(a); a != "" {
				c.Agents = append(c.Agents, a)
			}
		}
	case KeyRegistry:
		c.Registry = value
	case KeyNaming:
		c.Naming = value
	}
}

func validateKey(key string) error {
	for _, k := range Keys {
		if k == key {
			return nil
		}
	}
	return fmt.Errorf("unknown config key %q - supported keys: %s", key, strings.Join(Keys, ", "))
}

func validateValue(key, value string) error {
	switch key {
	case KeyAgent:
		if !ValidAgent(Agent(value)) {
			return fmt.Errorf("unsupported agent %q - supported values: %s", value, agentList())
		}
	case KeyAgents:
		if _, err := ParseAgents(value); err != nil {
			return err
		}
	case KeyRegistry:
		if _, err := source.Parse(value); err != nil {
			return err
		}
	case KeyNaming:
		if value != NamingLoose && value != NamingKebabCase {
			return fmt.Errorf("unsupported naming %q - supported values: %s, %s", value, NamingLoose, NamingKebabCase)
		}
	}
	return nil
}

// Lookup resolves key through env, project config, user config, then the default.
// Command flags are applied by callers before Lookup.
func Lookup(key string) (Entry, error) {
	if err := validateKey(key); err != nil {
		return Entry{}, err
	}
	if v := os.Getenv(envVars[key]); v != "" {
		return Entry{Key: key, Value: v, Source: SourceEnv}, nil
	}
	proj, err := LoadProject()
	if err != nil {
		return Entry{}, err
	}
	if v := proj.get(key); v != "" {
		return Entry{Key: key, Value: v, Source: SourceProject}, nil
	}
	user, err := Load()
	if err != nil {
		return Entry{}, err
	}
	if v := user.get(key); v != "" {
		return Entry{Key: key, Value: v, Source: SourceUser}, nil
	}
	switch key {
	case KeyAgent:
		return Entry{Key: key, Value: string(AgentCopilot), Source: SourceDefault}, nil
	case KeyNaming:
		return Entry{Key: key, Value: NamingLoose, Source: SourceDefault}, nil
	}
	return Entry{Key: key, Source: SourceDefault}, nil
}

// List resolves every supported key.
func List() ([]Entry, error) {
	entries := make([]Entry, 0, len(Keys))
	for _, k := range Keys {
		e, err := Lookup(k)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func scopePath(scope Scope) (string, error) {
	switch scope {
	case ScopeProject:
		return ProjectWritePath()
	case ScopeUser, "":
		return configFilePath()
	}
	return "", fmt.Errorf("unknown scope %q", scope)
}

// Set validates and writes key=value to the config file for scope.
func Set(scope Scope, key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := validateValue(key, value); err != nil {
		return err
	}
	path, err := scopePath(scope)
	if err != nil {
		return err
	}
	c, err := loadFile(path)
	if err != nil {
		return err
	}
	c.put(key, value)
	return saveFile(path, c)
}

// ProjectConfigFile returns the project config path for the repository at dir.
func ProjectConfigFile(dir string) string { return filepath.Join(dir, ".hikma", "config.json") }

// SetFile validates and writes key=value to the config file at path.
func SetFile(path, key, value string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	if err := validateValue(key, value); err != nil {
		return err
	}
	c, err := loadFile(path)
	if err != nil {
		return err
	}
	c.put(key, value)
	return saveFile(path, c)
}

// Unset removes key from the config file for scope.
func Unset(scope Scope, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	path, err := scopePath(scope)
	if err != nil {
		return err
	}
	c, err := loadFile(path)
	if err != nil {
		return err
	}
	c.put(key, "")
	if c.isEmpty() {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("cannot remove empty config file: %w", err)
		}
		return nil
	}
	return saveFile(path, c)
}

// CheckSkillName reports whether name is acceptable under the given naming
// convention. Every convention requires a safe single path segment; kebab-case
// additionally requires lowercase letters, digits, and hyphens.
func CheckSkillName(naming, name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
		return fmt.Errorf("invalid skill name %q: must be a single path segment", name)
	}
	if naming == NamingKebabCase {
		for _, r := range name {
			if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
				return fmt.Errorf("skill name must be kebab-case (lowercase letters, digits, hyphens): got %q\nchange with: hikma config set naming loose", name)
			}
		}
	}
	return nil
}

// NamingConvention returns the effective naming convention.
func NamingConvention() (string, error) {
	e, err := Lookup(KeyNaming)
	if err != nil {
		return "", err
	}
	if e.Value != NamingLoose && e.Value != NamingKebabCase {
		return "", fmt.Errorf("unsupported naming %q in %s config - use %s or %s", e.Value, e.Source, NamingLoose, NamingKebabCase)
	}
	return e.Value, nil
}

func (c Config) isEmpty() bool {
	return c.Agent == "" && c.Registry == "" && c.Naming == "" && len(c.Agents) == 0
}
