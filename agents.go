package jig

import _ "embed"

// agentGuide is the agent-facing onboarding text printed by `jig agents`.
// It lives in its own file so it reads and diffs like documentation, not
// like a Go string literal.
//
//go:embed agents.txt
var agentGuide string

// AgentGuide returns the agent-facing onboarding text: what jig is, its
// trust model, non-interactive usage, and pointers to --help and the docs.
func AgentGuide() string {
	return agentGuide
}
