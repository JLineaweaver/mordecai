package news

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jlineaweaver/mordecai/internal/module"
)

// Module fetches news headlines from RSS feeds.
type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "news"
}

type rssSource struct {
	Name string
	URL  string
}

func (m *Module) Fetch(ctx context.Context, cfg map[string]interface{}) (*module.Result, error) {
	sources, err := parseSources(cfg)
	if err != nil {
		return nil, err
	}

	maxItems := 10
	if v, ok := cfg["max_items"]; ok {
		switch n := v.(type) {
		case int:
			maxItems = n
		case float64:
			maxItems = int(n)
		}
	}

	var sections []string
	for _, src := range sources {
		items, err := fetchFeed(ctx, src.URL)
		if err != nil {
			sections = append(sections, fmt.Sprintf("**%s** - failed to fetch: %v", src.Name, err))
			continue
		}

		limit := maxItems
		if limit > len(items) {
			limit = len(items)
		}

		var lines []string
		for _, item := range items[:limit] {
			lines = append(lines, fmt.Sprintf("- [%s](%s)", item.Title, item.Link))
		}

		sections = append(sections, fmt.Sprintf("### %s\n%s", src.Name, strings.Join(lines, "\n")))
	}

	return &module.Result{
		Title:   "News",
		Content: strings.Join(sections, "\n\n"),
	}, nil
}

func parseSources(cfg map[string]interface{}) ([]rssSource, error) {
	rawSources, ok := cfg["sources"]
	if !ok {
		return nil, fmt.Errorf("news module requires 'sources' config")
	}

	sourceList, ok := rawSources.([]interface{})
	if !ok {
		return nil, fmt.Errorf("news 'sources' must be a list")
	}

	var sources []rssSource
	for _, raw := range sourceList {
		entry, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		url, _ := entry["url"].(string)
		if name != "" && url != "" {
			sources = append(sources, rssSource{Name: name, URL: url})
		}
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no valid news sources configured")
	}

	return sources, nil
}

// RSS XML structures
type rssFeed struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []rssItem `xml:"item"`
}

type rssItem struct {
	Title string `xml:"title"`
	Link  string `xml:"link"`
}

func fetchFeed(ctx context.Context, url string) ([]rssItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parsing RSS: %w", err)
	}

	return feed.Channel.Items, nil
}
