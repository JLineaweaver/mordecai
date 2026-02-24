package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/jlineaweaver/mordecai/internal/module"
)

// Module fetches weather forecasts from the Open-Meteo API.
type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "weather"
}

func (m *Module) Fetch(ctx context.Context, cfg map[string]interface{}) (*module.Result, error) {
	lat, lon, tz, err := parseConfig(cfg)
	if err != nil {
		return nil, err
	}

	data, err := fetchForecast(ctx, lat, lon, tz)
	if err != nil {
		return nil, fmt.Errorf("fetching weather: %w", err)
	}

	content := formatWeather(data, tz)

	return &module.Result{
		Title:   "Weather",
		Content: content,
	}, nil
}

func parseConfig(cfg map[string]interface{}) (lat, lon float64, tz string, err error) {
	latRaw, ok := cfg["latitude"]
	if !ok {
		return 0, 0, "", fmt.Errorf("weather module requires 'latitude' config")
	}
	lat, ok = toFloat64(latRaw)
	if !ok {
		return 0, 0, "", fmt.Errorf("weather 'latitude' must be a number")
	}

	lonRaw, ok := cfg["longitude"]
	if !ok {
		return 0, 0, "", fmt.Errorf("weather module requires 'longitude' config")
	}
	lon, ok = toFloat64(lonRaw)
	if !ok {
		return 0, 0, "", fmt.Errorf("weather 'longitude' must be a number")
	}

	tz, _ = cfg["timezone"].(string)
	if tz == "" {
		tz = "America/Denver"
	}

	return lat, lon, tz, nil
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func fetchForecast(ctx context.Context, lat, lon float64, tz string) (*forecastResponse, error) {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f"+
			"&current=temperature_2m,weather_code,wind_speed_10m"+
			"&daily=temperature_2m_max,temperature_2m_min,precipitation_probability_max,weather_code"+
			"&temperature_unit=fahrenheit&wind_speed_unit=mph"+
			"&timezone=%s&forecast_days=3",
		lat, lon, tz,
	)

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

	var data forecastResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parsing Open-Meteo response: %w", err)
	}

	return &data, nil
}

func formatWeather(data *forecastResponse, tz string) string {
	var lines []string

	// Current conditions.
	currentDesc := wmoDescription(data.Current.WeatherCode)
	lines = append(lines, fmt.Sprintf("**Currently**: %.0f°F, %s", data.Current.Temperature, currentDesc))

	// Today's details (first day in the daily arrays).
	if len(data.Daily.TempMax) > 0 {
		lines = append(lines, fmt.Sprintf("**Today**: High %.0f°F / Low %.0f°F",
			data.Daily.TempMax[0], data.Daily.TempMin[0]))

		if len(data.Daily.PrecipProb) > 0 {
			lines = append(lines, fmt.Sprintf("**Precipitation**: %d%% chance", data.Daily.PrecipProb[0]))
		}
	}

	lines = append(lines, fmt.Sprintf("**Wind**: %.0f mph", data.Current.WindSpeed))

	// 2-day lookahead.
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.FixedZone("MST", -7*60*60)
	}

	for i := 1; i < len(data.Daily.Time) && i <= 2; i++ {
		t, err := time.ParseInLocation("2006-01-02", data.Daily.Time[i], loc)
		if err != nil {
			continue
		}
		dayName := t.Format("Monday")
		desc := wmoDescription(data.Daily.WeatherCode[i])
		precip := 0
		if i < len(data.Daily.PrecipProb) {
			precip = data.Daily.PrecipProb[i]
		}
		lines = append(lines, fmt.Sprintf("**%s**: High %.0f°F / Low %.0f°F, %s (%d%% precip)",
			dayName,
			math.Round(data.Daily.TempMax[i]), math.Round(data.Daily.TempMin[i]),
			desc, precip))
	}

	return strings.Join(lines, "\n")
}

// WMO Weather interpretation codes -> human-readable descriptions.
func wmoDescription(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1:
		return "Mainly clear"
	case 2:
		return "Partly cloudy"
	case 3:
		return "Overcast"
	case 45, 48:
		return "Foggy"
	case 51:
		return "Light drizzle"
	case 53:
		return "Moderate drizzle"
	case 55:
		return "Dense drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61:
		return "Light rain"
	case 63:
		return "Moderate rain"
	case 65:
		return "Heavy rain"
	case 66, 67:
		return "Freezing rain"
	case 71:
		return "Light snow"
	case 73:
		return "Moderate snow"
	case 75:
		return "Heavy snow"
	case 77:
		return "Snow grains"
	case 80:
		return "Light rain showers"
	case 81:
		return "Moderate rain showers"
	case 82:
		return "Violent rain showers"
	case 85:
		return "Light snow showers"
	case 86:
		return "Heavy snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	default:
		return "Unknown"
	}
}

// Open-Meteo API response types.

type forecastResponse struct {
	Current currentData `json:"current"`
	Daily   dailyData   `json:"daily"`
}

type currentData struct {
	Temperature float64 `json:"temperature_2m"`
	WeatherCode int     `json:"weather_code"`
	WindSpeed   float64 `json:"wind_speed_10m"`
}

type dailyData struct {
	Time        []string  `json:"time"`
	TempMax     []float64 `json:"temperature_2m_max"`
	TempMin     []float64 `json:"temperature_2m_min"`
	PrecipProb  []int     `json:"precipitation_probability_max"`
	WeatherCode []int     `json:"weather_code"`
}
