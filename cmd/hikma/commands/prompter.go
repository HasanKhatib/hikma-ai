package commands

import (
	"errors"

	survey "github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
)

// errCancelled is returned by a Prompter when the user backs out (Ctrl-C). The
// wizard treats it as a quiet exit, not a failure.
var errCancelled = errors.New("cancelled")

// Prompter is the one place hikma asks the user anything, so tests use a fake
// and a change of prompt library touches this file only. Callers check
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
}

// prompter is what commands ask through. Tests replace it with a fake.
var prompter = defaultPrompter()

func defaultPrompter() Prompter {
	if useHuh() {
		return huhPrompter{}
	}
	return surveyPrompter{}
}

type surveyPrompter struct{}

func (surveyPrompter) Select(message string, options []string, def int) (int, error) {
	var idx int
	err := survey.AskOne(&survey.Select{Message: message, Options: options, Default: options[def], PageSize: len(options)}, &idx)
	return idx, cancelled(err)
}

func (surveyPrompter) MultiSelect(message string, options []string, defaults []int, min int) ([]int, error) {
	var picked []int
	var def []int
	def = append(def, defaults...)
	err := survey.AskOne(&survey.MultiSelect{Message: message, Options: options, Default: def}, &picked, survey.WithValidator(survey.MinItems(min)))
	return picked, cancelled(err)
}

func (surveyPrompter) Confirm(message string, def bool) (bool, error) {
	var ok bool
	err := survey.AskOne(&survey.Confirm{Message: message, Default: def}, &ok)
	return ok, cancelled(err)
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
