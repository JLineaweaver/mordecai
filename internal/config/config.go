package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration structure.
type Config struct {
	ModuleOrder []string       `yaml:"module_order,omitempty"`
	Modules     ModulesConfig  `yaml:"modules"`
	Delivery    DeliveryConfig `yaml:"delivery"`
}

// ModulesConfig holds config for all available modules.
type ModulesConfig struct {
	Sports  *ModuleEntry `yaml:"sports,omitempty"`
	Stocks  *ModuleEntry `yaml:"stocks,omitempty"`
	News    *ModuleEntry `yaml:"news,omitempty"`
	Weather *ModuleEntry `yaml:"weather,omitempty"`
}

// ModuleEntry represents a single module's config — an enabled flag plus
// arbitrary settings that get passed to the module as map[string]interface{}.
type ModuleEntry struct {
	Enabled  bool                   `yaml:"enabled"`
	Settings map[string]interface{} `yaml:",inline"`
}

// DeliveryConfig holds config for all delivery channels.
type DeliveryConfig struct {
	Discord *DeliveryEntry `yaml:"discord,omitempty"`
}

// DeliveryEntry represents a single delivery channel's config.
type DeliveryEntry struct {
	Enabled    bool   `yaml:"enabled"`
	WebhookURL string `yaml:"webhook_url"`
}

// Load reads a YAML config file, performs env var substitution, and returns
// the parsed Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	// Substitute ${ENV_VAR} references with environment variable values.
	expanded := expandEnvVars(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	return &cfg, nil
}

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnvVars replaces ${VAR_NAME} with the corresponding env var value.
func expandEnvVars(s string) string {
	return envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match // leave unresolved vars as-is
	})
}

// EnabledModules returns a map of module name -> settings for all enabled modules.
func (c *Config) EnabledModules() map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	entries := map[string]*ModuleEntry{
		"sports":  c.Modules.Sports,
		"stocks":  c.Modules.Stocks,
		"news":    c.Modules.News,
		"weather": c.Modules.Weather,
	}

	for name, entry := range entries {
		if entry != nil && entry.Enabled {
			settings := entry.Settings
			if settings == nil {
				settings = make(map[string]interface{})
			}
			result[name] = settings
		}
	}

	return result
}
