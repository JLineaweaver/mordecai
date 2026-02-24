package stocks

import (
	"context"
	"fmt"

	"github.com/jlineaweaver/mordecai/internal/module"
)

// Module fetches stock prices for watched tickers.
// TODO: Implement using Yahoo Finance in Phase 2.
type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "stocks"
}

func (m *Module) Fetch(ctx context.Context, cfg map[string]interface{}) (*module.Result, error) {
	return nil, fmt.Errorf("stocks module not yet implemented")
}
