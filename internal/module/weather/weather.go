package weather

import (
	"context"
	"fmt"

	"github.com/jlineaweaver/mordecai/internal/module"
)

// Module fetches weather forecasts.
// TODO: Implement using Open-Meteo API in Phase 3.
type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "weather"
}

func (m *Module) Fetch(ctx context.Context, cfg map[string]interface{}) (*module.Result, error) {
	return nil, fmt.Errorf("weather module not yet implemented")
}
