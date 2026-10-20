package commands

import (
	"context"
	"errors"
)

// errCancelled is returned by a Prompter when the user backs out (Ctrl-C). The
// wizard treats it as a quiet exit, not a failure.
var errCancelled = errors.New("cancelled")

// Prompter is the one place hikma asks the user anything, so tests use a fake
// and a change of prompt library touches prompter_huh.go only. Callers check
// isInteractive first; a Prompter never has to cope with a missing terminal.
type Prompter interface {
	// Select shows options and returns the index of the chosen one. def is the
	// index highlighted first.
	Select(message string, options []string, def int) (int, error)
	// MultiSelect returns the indexes of the chosen options, at least min of
	// them. defaults are the indexes ticked first.
	MultiSelect(message string, options []string, defaults []int, min int) ([]int, error)
	// Confirm asks a yes or no question. def is the answer Enter gives.
	Confirm(message string, def bool) (bool, error)
	// Input asks for a line of text and re-asks until validate accepts it.
	Input(message string, validate func(string) error) (string, error)
	// Spinner shows message while fn runs, so a slow fetch does not look like a
	// hang. Ctrl-C stops waiting and returns errCancelled; ctx is cancelled then.
	// Without a terminal or in accessible mode it prints message once and runs fn.
	Spinner(message string, fn func(ctx context.Context) error) error
}

// prompter is what commands ask through. Tests replace it with a fake.
var prompter Prompter = huhPrompter{}
