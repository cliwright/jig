package jig

import (
	"strings"
	"testing"
)

func TestAgentGuide(t *testing.T) {
	guide := AgentGuide()
	for _, want := range []string{"jig.cliwright.dev", "github.com/cliwright/jig", "--var", "jig --help"} {
		if !strings.Contains(guide, want) {
			t.Errorf("agent guide should mention %q", want)
		}
	}
}
