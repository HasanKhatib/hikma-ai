package config_test

import (
	"testing"

	"github.com/hasankhatib/hikma-ai/internal/config"
)

func TestParseRegistryAcceptsGitHubForms(t *testing.T) {
	tests := []string{
		"hasankhatib/ai",
		"https://github.com/hasankhatib/ai.git",
		"git@github.com:hasankhatib/ai.git",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			r, err := config.ParseRegistry(tt)
			if err != nil {
				t.Fatalf("ParseRegistry() error = %v", err)
			}
			if got := r.FullName(); got != "hasankhatib/ai" {
				t.Fatalf("FullName() = %q", got)
			}
		})
	}
}

func TestParseRegistryRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"", "hasankhatib", "https://example.com/hasankhatib/ai"} {
		t.Run(value, func(t *testing.T) {
			if _, err := config.ParseRegistry(value); err == nil {
				t.Fatalf("ParseRegistry(%q) returned nil error", value)
			}
		})
	}
}
