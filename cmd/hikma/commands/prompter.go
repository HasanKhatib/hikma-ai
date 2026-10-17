package commands

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

// errCancelled is returned by a Prompter when the user backs out (Ctrl-C). The
// wizard treats it as a quiet exit, not a failure.
var errCancelled = errors.New("cancelled")

// Prompter is the one place the wizard asks the user anything, so tests use a
// fake and a change of prompt library touches this file only.
type Prompter interface {
	// Select shows options and returns the index of the chosen one.
	Select(message string, options []string) (int, error)
	// Input asks for a line of text and re-asks until validate accepts it.
	Input(message string, validate func(string) error) (string, error)
}

type surveyPrompter struct{}

func (surveyPrompter) Select(message string, options []string) (int, error) {
	var idx int
	err := survey.AskOne(&survey.Select{Message: message, Options: options, PageSize: len(options)}, &idx)
	return idx, cancelled(err)
}

func (surveyPrompter) Input(message string, validate func(string) error) (string, error) {
	var s string
	opts := []survey.AskOpt{survey.WithValidator(survey.Required)}
	if validate != nil {
		opts = append(opts, survey.WithValidator(func(ans any) error { return validate(ans.(string)) }))
	}
	err := survey.AskOne(&survey.Input{Message: message}, &s, opts...)
	return s, cancelled(err)
}

func cancelled(err error) error {
	if errors.Is(err, terminal.InterruptErr) {
		return errCancelled
	}
	return err
}
