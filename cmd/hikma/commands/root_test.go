package commands

import "testing"

func TestRootCommandUsesHikmaName(t *testing.T) {
	cmd := Root()
	if cmd.Use != "hikma" {
		t.Fatalf("Root().Use = %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Fatalf("Root().Short is empty")
	}
}
