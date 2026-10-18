package commands

import (
	"errors"
	"strings"
)

// Exit codes. They are part of the 1.0 contract: scripts and CI may depend on
// them, and new ones are added only in a major version.
const (
	ExitOK       = 0 // success
	ExitFailure  = 1 // the command failed
	ExitUsage    = 2 // wrong command, flag, or argument
	ExitMismatch = 3 // installed skills differ from the lockfile (skill verify, sync --frozen)
)

// ExitError carries the exit code a command wants.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

func usageError(err error) error {
	if err == nil {
		return nil
	}
	return &ExitError{Code: ExitUsage, Err: err}
}

func mismatchError(err error) error { return &ExitError{Code: ExitMismatch, Err: err} }

// ExitCode returns the process exit code for err: 0 for nil, the code an
// ExitError carries, 2 for the usage mistakes cobra reports itself, and 1 otherwise.
func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var ee *ExitError
	if errors.As(err, &ee) {
		return ee.Code
	}
	// Cobra words these itself, before any command runs.
	if strings.HasPrefix(err.Error(), "unknown command ") {
		return ExitUsage
	}
	return ExitFailure
}
