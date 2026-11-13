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
			if len(v.Choices) > 0 {
				label = fmt.Sprintf("%s (%s)", label, strings.Join(v.Choices, " | "))
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
// defaults resolved in dependency order, with varsFile (when non-empty) and
// then the key=value pairs in set as explicit overrides, and finally passed
// to prompt (when non-nil). An override wins for its variable; defaults
// derived from it re-evaluate against the overridden value. Later sources
// win. The result is not validated here — Render validates values against
// the manifest, patterns included.
func ResolveValues(m *Manifest, varsFile string, set []string, prompt Prompter) (map[string]string, error) {
	overrides := map[string]string{}
	if varsFile != "" {
		fromFile, err := ReadVarsFile(varsFile)
		if err != nil {
			return nil, err
		}
		for k, v := range fromFile {
			overrides[k] = v
		}
	}
	for _, kv := range set {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			return nil, fmt.Errorf("variable override %q: want key=value", kv)
		}
		overrides[key] = value
	}
	values, err := m.resolveDefaults(overrides)
	if err != nil {
		return nil, err
	}
	for k, v := range overrides {
		if _, ok := m.Variables[k]; !ok {
			values[k] = v // undeclared: Validate reports it
		}
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
