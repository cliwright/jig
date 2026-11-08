package jig

import "text/template/parse"

// collectUsed walks a parsed template tree recording the names of every
// variable reference ({{.Name}}) in used, driving the unused-variable
// warning. Function arguments, pipelines, and branches are all traversed.
func collectUsed(node parse.Node, used map[string]bool) {
	switch n := node.(type) {
	case nil:
	case *parse.ListNode:
		for _, child := range n.Nodes {
			collectUsed(child, used)
		}
	case *parse.ActionNode:
		collectUsed(n.Pipe, used)
	case *parse.PipeNode:
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
