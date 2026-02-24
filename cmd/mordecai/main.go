package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/jlineaweaver/mordecai/internal/config"
	"github.com/jlineaweaver/mordecai/internal/delivery/discord"
	"github.com/jlineaweaver/mordecai/internal/digest"
	"github.com/jlineaweaver/mordecai/internal/delivery"
	"github.com/jlineaweaver/mordecai/internal/module"
	"github.com/jlineaweaver/mordecai/internal/module/news"
	"github.com/jlineaweaver/mordecai/internal/module/sports"
	"github.com/jlineaweaver/mordecai/internal/module/stocks"
	"github.com/jlineaweaver/mordecai/internal/module/weather"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Register available modules.
	modules := map[string]module.Module{
		"news":   news.New(),
		"sports": sports.New(),
		"stocks":  stocks.New(),
		"weather": weather.New(),
	}

	// Register available delivery channels.
	var deliveries []delivery.Delivery
	if cfg.Delivery.Discord != nil && cfg.Delivery.Discord.Enabled {
		if cfg.Delivery.Discord.WebhookURL == "" {
			fmt.Fprintf(os.Stderr, "error: discord enabled but webhook_url is empty (set DISCORD_WEBHOOK_URL env var)\n")
			os.Exit(1)
		}
		deliveries = append(deliveries, discord.New(cfg.Delivery.Discord.WebhookURL))
	}

	if len(deliveries) == 0 {
		fmt.Fprintf(os.Stderr, "error: no delivery channels enabled\n")
		os.Exit(1)
	}

	// Run the digest with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	orch := digest.New(cfg, modules, deliveries)
	if err := orch.Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Digest sent successfully!")
}
