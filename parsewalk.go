package jig

import (
	"text/template"
	"text/template/parse"
)

// collectUsedAll records variable references across every template
// associated with t — the root tree and every {{define}} — so a broken
// named template is caught even when no executed branch invokes it.
func collectUsedAll(t *template.Template, used map[string]bool) {
	for _, tt := range t.Templates() {
		if tt.Tree != nil {
			collectUsed(tt.Tree.Root, used)
		}
	}
}

// collectUsed walks a parsed template tree recording the names of every
// variable reference ({{.Name}}) in used, driving the unused-variable
// warning. Function arguments, pipelines, and branches are all traversed.
func collectUsed(node parse.Node, used map[string]bool) {
	switch n := node.(type) {
	case nil:
	case *parse.ListNode:
		if n == nil {
			return // a branch without an else has a typed-nil ElseList
		}
		for _, child := range n.Nodes {
			collectUsed(child, used)
		}
	case *parse.ActionNode:
		collectUsed(n.Pipe, used)
	case *parse.PipeNode:
		if n == nil {
			return
		}
		for _, cmd := range n.Cmds {
			collectUsed(cmd, used)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			collectUsed(arg, used)
		}
	case *parse.FieldNode:
		if len(n.Ident) > 0 {
			used[n.Ident[0]] = true
		}
	case *parse.BranchNode:
		collectUsed(n.Pipe, used)
		collectUsed(n.List, used)
		collectUsed(n.ElseList, used)
	case *parse.IfNode:
		collectUsed(&n.BranchNode, used)
	case *parse.RangeNode:
		collectUsed(&n.BranchNode, used)
	case *parse.WithNode:
		collectUsed(&n.BranchNode, used)
	case *parse.TemplateNode:
		collectUsed(n.Pipe, used)
	}
}
