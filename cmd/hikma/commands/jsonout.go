package commands

import (
	"encoding/json"
	"fmt"
	"io"
)

// jsonVersion is the schema version of every --json document. Fields may be
// added within a major version; existing fields keep their names and meaning.
const jsonVersion = 1

// printJSON marshals v to indented JSON on w (the command's stdout).
func printJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}
	_, err = fmt.Fprintln(w, string(data))
	return err
}

// strs returns s, or an empty slice so JSON shows [] and never null.
func strs(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
