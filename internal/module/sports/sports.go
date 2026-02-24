package sports

import (
	"context"
	"fmt"

	"github.com/jlineaweaver/mordecai/internal/module"
)

// Module fetches sports scores and upcoming games.
// TODO: Implement using ESPN API in Phase 2.
type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "sports"
}

func (m *Module) Fetch(ctx context.Context, cfg map[string]interface{}) (*module.Result, error) {
	return nil, fmt.Errorf("sports module not yet implemented")
}
