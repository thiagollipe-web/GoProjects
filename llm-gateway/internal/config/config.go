// Package config loads the gateway configuration from a JSON file.
// API keys are never stored in the file: each upstream names the
// environment variable that holds its key.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// Duration is a time.Duration that unmarshals from strings like "30s".
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return fmt.Errorf("duration must be a string like \"30s\": %w", err)
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	d.Duration = v
	return nil
}

// Upstream describes one OpenAI-compatible provider (Ollama, Groq, Maritaca...).
type Upstream struct {
	Name      string            `json:"name"`
	BaseURL   string            `json:"base_url"`
	APIKeyEnv string            `json:"api_key_env"` // optional (e.g. local Ollama)
	Models    map[string]string `json:"models"`      // public alias -> upstream model id
	Timeout   Duration          `json:"timeout"`
}

// Config is the full gateway configuration.
type Config struct {
	Listen        string     `json:"listen"`
	ClientKeysEnv string     `json:"client_keys_env"` // comma-separated keys accepted from clients; empty = open
	RatePerMinute int        `json:"rate_per_minute"` // per client, 0 = unlimited
	BreakerFails  int        `json:"breaker_failures"`
	BreakerCool   Duration   `json:"breaker_cooldown"`
	Upstreams     []Upstream `json:"upstreams"` // order = priority
}

// Load reads and validates a configuration file.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate applies defaults and checks required fields.
func (c *Config) Validate() error {
	if c.Listen == "" {
		c.Listen = ":8080"
	}
	if c.BreakerFails <= 0 {
		c.BreakerFails = 3
	}
	if c.BreakerCool.Duration <= 0 {
		c.BreakerCool.Duration = 30 * time.Second
	}
	if len(c.Upstreams) == 0 {
		return errors.New("config: at least one upstream is required")
	}
	seen := map[string]bool{}
	for i := range c.Upstreams {
		u := &c.Upstreams[i]
		if u.Name == "" || u.BaseURL == "" {
			return fmt.Errorf("config: upstream #%d needs name and base_url", i)
		}
		if seen[u.Name] {
			return fmt.Errorf("config: duplicate upstream %q", u.Name)
		}
		seen[u.Name] = true
		if len(u.Models) == 0 {
			return fmt.Errorf("config: upstream %q needs at least one model alias", u.Name)
		}
		if u.Timeout.Duration <= 0 {
			u.Timeout.Duration = 60 * time.Second
		}
	}
	return nil
}
