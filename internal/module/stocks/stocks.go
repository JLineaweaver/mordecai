package stocks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jlineaweaver/mordecai/internal/module"
)

// Module fetches stock prices from Yahoo Finance's public chart API.
type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "stocks"
}

func (m *Module) Fetch(ctx context.Context, cfg map[string]interface{}) (*module.Result, error) {
	tickers, err := parseTickers(cfg)
	if err != nil {
		return nil, err
	}

	var lines []string
	for _, ticker := range tickers {
		line, err := fetchTicker(ctx, ticker)
		if err != nil {
			lines = append(lines, fmt.Sprintf("- **%s**: *failed to fetch*", ticker))
			continue
		}
		lines = append(lines, line)
	}

	return &module.Result{
		Title:   "Stocks",
		Content: strings.Join(lines, "\n"),
	}, nil
}

func parseTickers(cfg map[string]interface{}) ([]string, error) {
	raw, ok := cfg["tickers"]
	if !ok {
		return nil, fmt.Errorf("stocks module requires 'tickers' config")
	}

	list, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("stocks 'tickers' must be a list")
	}

	var tickers []string
	for _, item := range list {
		if s, ok := item.(string); ok && s != "" {
			tickers = append(tickers, s)
		}
	}

	if len(tickers) == 0 {
		return nil, fmt.Errorf("no valid tickers configured")
	}

	return tickers, nil
}

func fetchTicker(ctx context.Context, ticker string) (string, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=1d", ticker)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var data yfResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}

	if len(data.Chart.Result) == 0 {
		return "", fmt.Errorf("no data returned")
	}

	meta := data.Chart.Result[0].Meta
	price := meta.RegularMarketPrice
	prevClose := meta.ChartPreviousClose

	change := price - prevClose
	pctChange := 0.0
	if prevClose != 0 {
		pctChange = (change / prevClose) * 100
	}

	arrow := "▲"
	if change < 0 {
		arrow = "▼"
	}

	return fmt.Sprintf("- **%s** $%.2f %s %.2f (%.2f%%)",
		ticker, price, arrow, change, pctChange), nil
}

// Yahoo Finance v8 chart API response types

type yfResponse struct {
	Chart struct {
		Result []struct {
			Meta yfMeta `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
}

type yfMeta struct {
	Symbol             string  `json:"symbol"`
	Currency           string  `json:"currency"`
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	ChartPreviousClose float64 `json:"chartPreviousClose"`
}
