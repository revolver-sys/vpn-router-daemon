package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VPNRouterUpPath   string        `yaml:"vpn_router_up_path"`
	VPNRouterDownPath string        `yaml:"vpn_router_down_path"`
	HealthCheckURL    string        `yaml:"health_check_url"`
	CheckInterval     time.Duration `yaml:"check_interval"`
	CommandTimeout    time.Duration `yaml:"command_timeout"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, "vpn", ".config", "vpnrd", "config.yaml"), nil
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse yaml %q: %w", path, err)
	}

	applyDefaults(&c)

	if err := validate(&c); err != nil {
		return nil, err
	}

	return &c, nil
}

func applyDefaults(c *Config) {
	if c.HealthCheckURL == "" {
		c.HealthCheckURL = "https://api.ipify.org?format=text"
	}
	if c.CheckInterval == 0 {
		c.CheckInterval = 10 * time.Second
	}
	if c.CommandTimeout == 0 {
		c.CommandTimeout = 20 * time.Second
	}
}

func validate(c *Config) error {
	var problems []string

	if c.VPNRouterUpPath == "" {
		problems = append(problems, "vpn_router_up_path is required")
	}
	if c.VPNRouterDownPath == "" {
		problems = append(problems, "vpn_router_down_path is required")
	}
	if c.CheckInterval < 1*time.Second {
		problems = append(problems, "check_interval must be >= 1s")
	}
	if c.CommandTimeout < 1*time.Second {
		problems = append(problems, "command_timeout must be >= 1s")
	}

	if len(problems) > 0 {
		return errors.New("config invalid: " + joinProblems(problems))
	}
	return nil
}

func joinProblems(p []string) string {
	if len(p) == 1 {
		return p[0]
	}
	out := ""
	for i, s := range p {
		if i > 0 {
			out += "; "
		}
		out += s
	}
	return out
}
