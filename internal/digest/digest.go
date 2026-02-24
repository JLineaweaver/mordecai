package digest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jlineaweaver/mordecai/internal/config"
	"github.com/jlineaweaver/mordecai/internal/delivery"
	"github.com/jlineaweaver/mordecai/internal/module"
)

// Orchestrator runs modules in parallel and delivers the combined digest.
type Orchestrator struct {
	modules    map[string]module.Module
	deliveries []delivery.Delivery
	cfg        *config.Config
}

// New creates a new Orchestrator with the given modules and delivery channels.
func New(cfg *config.Config, modules map[string]module.Module, deliveries []delivery.Delivery) *Orchestrator {
	return &Orchestrator{
		modules:    modules,
		deliveries: deliveries,
		cfg:        cfg,
	}
}

type moduleResult struct {
	name   string
	result *module.Result
	err    error
}

// Run executes all enabled modules in parallel, assembles the digest, and
// sends it via all enabled delivery channels.
func (o *Orchestrator) Run(ctx context.Context) error {
	enabledModules := o.cfg.EnabledModules()
	if len(enabledModules) == 0 {
		return fmt.Errorf("no modules enabled in config")
	}

	// Run modules in parallel.
	results := make(chan moduleResult, len(enabledModules))
	var wg sync.WaitGroup

	for name, settings := range enabledModules {
		mod, ok := o.modules[name]
		if !ok {
			fmt.Printf("warning: module %q enabled in config but not registered\n", name)
			continue
		}

		wg.Add(1)
		go func(name string, mod module.Module, settings map[string]interface{}) {
			defer wg.Done()
			res, err := mod.Fetch(ctx, settings)
			results <- moduleResult{name: name, result: res, err: err}
		}(name, mod, settings)
	}

	// Close results channel once all modules finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results in order of completion.
	var sections []string
	var errors []string
	for r := range results {
		if r.err != nil {
			errors = append(errors, fmt.Sprintf("**%s**: %v", r.name, r.err))
			continue
		}
		if r.result != nil {
			sections = append(sections, formatSection(r.result))
		}
	}

	if len(sections) == 0 && len(errors) > 0 {
		return fmt.Errorf("all modules failed:\n%s", strings.Join(errors, "\n"))
	}

	digest := buildDigest(sections, errors)

	// Deliver to all channels.
	var deliveryErrors []string
	for _, d := range o.deliveries {
		if err := d.Send(ctx, digest); err != nil {
			deliveryErrors = append(deliveryErrors, fmt.Sprintf("%s: %v", d.Name(), err))
		}
	}

	if len(deliveryErrors) > 0 {
		return fmt.Errorf("delivery errors: %s", strings.Join(deliveryErrors, "; "))
	}

	return nil
}

func formatSection(r *module.Result) string {
	return fmt.Sprintf("## %s\n%s", r.Title, r.Content)
}

func buildDigest(sections []string, errors []string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("# Mordecai Daily Digest - %s\n\n", time.Now().Format("Monday, January 2, 2006")))

	b.WriteString(strings.Join(sections, "\n\n---\n\n"))

	if len(errors) > 0 {
		b.WriteString("\n\n---\n\n## Errors\n")
		b.WriteString(strings.Join(errors, "\n"))
	}

	return b.String()
}
