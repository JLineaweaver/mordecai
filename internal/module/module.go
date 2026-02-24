package module

import "context"

// Module defines the interface that all digest modules must implement.
type Module interface {
	Name() string
	Fetch(ctx context.Context, cfg map[string]interface{}) (*Result, error)
}

// Result holds the output of a module's Fetch call.
type Result struct {
	Title   string
	Content string // Markdown-formatted content
}
