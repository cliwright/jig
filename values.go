package jig

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Prompter collects variable values interactively, mutating values in
// place (typically defaulting on empty input). A nil Prompter skips
// prompting entirely.
type Prompter func(m *Manifest, values map[string]string) error

// StdinPrompter returns a Prompter asking for each variable in sorted
// order on in/out; an empty answer keeps the current value.
func StdinPrompter(in io.Reader, out io.Writer) Prompter {
	return func(m *Manifest, values map[string]string) error {
		reader := bufio.NewReader(in)
		for _, name := range m.Names() {
			v := m.Variables[name]
			label := v.Prompt
			if label == "" {
				label = name
			}
			fmt.Fprintf(out, "%s [%s]: ", label, values[name])
			answer, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("reading answer for %s: %w", name, err)
			}
			if answer = strings.TrimSpace(answer); answer != "" {
				values[name] = answer
			}
		}
		return nil
	}
}

// ResolveValues builds the final variable values for a render over m:
// manifest defaults, overlaid by varsFile (when non-empty) and then by the
// key=value pairs in set, and finally passed to prompt (when non-nil).
// Later sources win. The result is not validated here — Render validates
// values against the manifest, patterns included.
func ResolveValues(m *Manifest, varsFile string, set []string, prompt Prompter) (map[string]string, error) {
	values := m.Defaults()
	if varsFile != "" {
		fromFile, err := ReadVarsFile(varsFile)
		if err != nil {
			return nil, err
		}
		for k, v := range fromFile {
			values[k] = v
		}
	}
	for _, kv := range set {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("variable override %q: want key=value", kv)
		}
		values[key] = value
	}
	if prompt != nil {
		if err := prompt(m, values); err != nil {
			return nil, err
		}
	}
	return values, nil
}

// ReadVarsFile reads a JSON object of string values, the non-interactive
// alternative to prompting.
func ReadVarsFile(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parsing %s: want a JSON object of string values: %w", path, err)
	}
	return values, nil
}
