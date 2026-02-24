package sports

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jlineaweaver/mordecai/internal/module"
)

// Module fetches sports scores and upcoming games from ESPN's public API.
type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "sports"
}

// ESPN league slug mapping: league name -> (sport, league) for URL construction.
var leagueSlugs = map[string][2]string{
	"NFL": {"football", "nfl"},
	"NBA": {"basketball", "nba"},
	"MLB": {"baseball", "mlb"},
	"NHL": {"hockey", "nhl"},
	"EPL": {"soccer", "eng.1"},
	"MLS": {"soccer", "usa.1"},
}

func (m *Module) Fetch(ctx context.Context, cfg map[string]interface{}) (*module.Result, error) {
	leagues, err := parseLeagues(cfg)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	today := now

	var sections []string
	for _, league := range leagues {
		section, err := fetchLeague(ctx, league, yesterday, today)
		if err != nil {
			sections = append(sections, fmt.Sprintf("### %s\n*Failed to fetch: %v*", league.Name, err))
			continue
		}
		if section != "" {
			sections = append(sections, section)
		}
	}

	if len(sections) == 0 {
		return &module.Result{
			Title:   "Sports",
			Content: "*No games found for your teams.*",
		}, nil
	}

	return &module.Result{
		Title:   "Sports",
		Content: strings.Join(sections, "\n\n"),
	}, nil
}

type leagueConfig struct {
	Name  string
	Teams []string
}

func parseLeagues(cfg map[string]interface{}) ([]leagueConfig, error) {
	raw, ok := cfg["leagues"]
	if !ok {
		return nil, fmt.Errorf("sports module requires 'leagues' config")
	}

	list, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("sports 'leagues' must be a list")
	}

	var leagues []leagueConfig
	for _, item := range list {
		entry, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		rawTeams, _ := entry["teams"].([]interface{})

		var teams []string
		for _, t := range rawTeams {
			if s, ok := t.(string); ok {
				teams = append(teams, s)
			}
		}

		if name != "" && len(teams) > 0 {
			leagues = append(leagues, leagueConfig{Name: name, Teams: teams})
		}
	}

	if len(leagues) == 0 {
		return nil, fmt.Errorf("no valid leagues configured")
	}

	return leagues, nil
}

func fetchLeague(ctx context.Context, league leagueConfig, yesterday, today time.Time) (string, error) {
	slugs, ok := leagueSlugs[strings.ToUpper(league.Name)]
	if !ok {
		return "", fmt.Errorf("unknown league %q", league.Name)
	}

	// Fetch yesterday and today's games in a single request using date range.
	dateRange := fmt.Sprintf("%s-%s", yesterday.Format("20060102"), today.Format("20060102"))
	url := fmt.Sprintf("https://site.api.espn.com/apis/site/v2/sports/%s/%s/scoreboard?dates=%s",
		slugs[0], slugs[1], dateRange)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

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

	var data espnResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("parsing ESPN response: %w", err)
	}

	// Build a set of team names we care about (lowercase for matching).
	teamSet := make(map[string]bool)
	for _, t := range league.Teams {
		teamSet[strings.ToLower(t)] = true
	}

	// The ESPN date range gives us yesterday+today's games.
	// We want: completed games (results) + today's scheduled/in-progress games.
	// Filter out scheduled games that aren't today.
	todayUTC := today.UTC().Format("2006-01-02")

	var lines []string
	for _, event := range data.Events {
		if len(event.Competitions) == 0 {
			continue
		}
		comp := event.Competitions[0]

		// Check if any of our teams are in this game.
		relevant := false
		for _, c := range comp.Competitors {
			if teamSet[strings.ToLower(c.Team.DisplayName)] {
				relevant = true
				break
			}
		}
		if !relevant {
			continue
		}

		eventTime, err := time.Parse(time.RFC3339, event.Date)
		if err != nil {
			// Try alternate format ESPN sometimes uses.
			eventTime, err = time.Parse("2006-01-02T15:04Z", event.Date)
			if err != nil {
				continue
			}
		}

		state := comp.Status.Type.State

		// Always show completed and in-progress games.
		// For scheduled games, only show if they're today.
		if state == "pre" {
			eventDateUTC := eventTime.UTC().Format("2006-01-02")
			if eventDateUTC != todayUTC {
				continue
			}
		}

		lines = append(lines, formatGame(comp, eventTime))
	}

	if len(lines) == 0 {
		return "", nil
	}

	return fmt.Sprintf("### %s\n%s", league.Name, strings.Join(lines, "\n")), nil
}

func formatGame(comp espnCompetition, eventTime time.Time) string {
	var home, away espnCompetitor
	for _, c := range comp.Competitors {
		if c.HomeAway == "home" {
			home = c
		} else {
			away = c
		}
	}

	state := comp.Status.Type.State

	switch state {
	case "post":
		return fmt.Sprintf("- **%s** %s - %s **%s** (%s)",
			away.Team.Abbreviation, away.Score,
			home.Score, home.Team.Abbreviation,
			comp.Status.Type.Description)
	case "in":
		return fmt.Sprintf("- **%s** %s - %s **%s** (%s %s)",
			away.Team.Abbreviation, away.Score,
			home.Score, home.Team.Abbreviation,
			comp.Status.Type.Description, comp.Status.DisplayClock)
	default:
		// Scheduled — show game time in local timezone.
		gameTime := eventTime.Local().Format("3:04 PM")
		return fmt.Sprintf("- %s @ %s — %s",
			away.Team.DisplayName, home.Team.DisplayName, gameTime)
	}
}

// ESPN API response types

type espnResponse struct {
	Events []espnEvent `json:"events"`
}

type espnEvent struct {
	Date         string            `json:"date"`
	Competitions []espnCompetition `json:"competitions"`
}

type espnCompetition struct {
	Competitors []espnCompetitor `json:"competitors"`
	Status      espnStatus       `json:"status"`
}

type espnCompetitor struct {
	HomeAway string   `json:"homeAway"`
	Score    string   `json:"score"`
	Team     espnTeam `json:"team"`
}

type espnTeam struct {
	DisplayName  string `json:"displayName"`
	Abbreviation string `json:"abbreviation"`
}

type espnStatus struct {
	DisplayClock string         `json:"displayClock"`
	Type         espnStatusType `json:"type"`
}

type espnStatusType struct {
	State       string `json:"state"`
	Description string `json:"description"`
}
