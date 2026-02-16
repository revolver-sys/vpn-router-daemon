package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Interfaces (optional; scripts can still have defaults)
	WANIF string `yaml:"wan_if"`
	LANIF string `yaml:"lan_if"`

	// VPNRouterUpPath   string `yaml:"vpn_router_up_path"`
	VPNRouterDownPath string `yaml:"vpn_router_down_path"`
	// Router scripts (new split)
	VPNRouterSetupPath   string `yaml:"vpn_router_setup_path"`
	VPNRouterPFApplyPath string `yaml:"vpn_router_pf_apply_path"`

	HealthCheckURL string        `yaml:"health_check_url"`
	CheckInterval  time.Duration `yaml:"check_interval"`
	CommandTimeout time.Duration `yaml:"command_timeout"`

	// sing-box control
	SingBoxAdoptExternal *bool         `yaml:"singbox_adopt_external"`
	SingBoxPath          string        `yaml:"singbox_path"`
	SingBoxConfigPath    string        `yaml:"singbox_config_path"`
	SingBoxAutoStart     bool          `yaml:"singbox_auto_start"`
	SingBoxAutoStop      bool          `yaml:"singbox_auto_stop"`
	SingBoxStartTimeout  time.Duration `yaml:"singbox_start_timeout"`
	SingBoxStopTimeout   time.Duration `yaml:"singbox_stop_timeout"`
	SingBoxPidFile       string        `yaml:"singbox_pid_file"`
	SingBoxLogFile       string        `yaml:"singbox_log_file"`

	// Watchdog
	FailureThreshold int           `yaml:"failure_threshold"`
	RecoverCooldown  time.Duration `yaml:"recover_cooldown"`
	MaxRecoveries    int           `yaml:"max_recoveries"`
	HealthTimeout    time.Duration `yaml:"health_timeout"`

	// Kill-switch allowlists (planned)
	VPNServerIPs []string `yaml:"vpn_server_ips"` // e.g. ["89.40.206.121"]
	WANDNSIPs    []string `yaml:"wan_dns_ips"`    // optional
	AllowWANNTP  bool     `yaml:"allow_wan_ntp"`  // optional
}

// defoult config.yaml path: /Users/alexgoodkarma/vpn/config/vpnrd/config.yaml
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, "vpn", "config", "vpnrd", "config.yaml"), nil
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

	// sing-box defaults
	if c.SingBoxPath == "" {
		c.SingBoxPath = "/usr/local/bin/sing-box"
	}
	if c.SingBoxStartTimeout == 0 {
		c.SingBoxStartTimeout = 8 * time.Second
	}
	if c.SingBoxStopTimeout == 0 {
		c.SingBoxStopTimeout = 8 * time.Second
	}
	// /Users/alexgoodkarma/vpn/config/vpnrd/singbox.pid
	// /Users/alexgoodkarma/vpn/config/vpnrd/singbox.log
	if c.SingBoxPidFile == "" || c.SingBoxLogFile == "" {
		home, _ := os.UserHomeDir()
		if c.SingBoxPidFile == "" {
			c.SingBoxPidFile = filepath.Join(home, "config", "vpnrd", "singbox.pid")
		}
		if c.SingBoxLogFile == "" {
			c.SingBoxLogFile = filepath.Join(home, "config", "vpnrd", "singbox.log")
		}
	}
	if c.SingBoxAdoptExternal == nil {
		v := true
		c.SingBoxAdoptExternal = &v
	}

	// Watchdog
	if c.FailureThreshold == 0 {
		c.FailureThreshold = 3
	}
	if c.RecoverCooldown == 0 {
		c.RecoverCooldown = 5 * time.Second
	}
	if c.MaxRecoveries == 0 {
		c.MaxRecoveries = 5
	}
	if c.HealthTimeout == 0 {
		c.HealthTimeout = 5 * time.Second
	}

}

func (c *Config) AdoptExternal() bool {
	if c.SingBoxAdoptExternal == nil {
		return true
	}
	return *c.SingBoxAdoptExternal
}

func validate(c *Config) error {
	var problems []string

	if c.SingBoxAutoStart {
		if c.SingBoxPath == "" {
			problems = append(problems, "singbox_path is required when singbox_auto_start=true")
		}
		if c.SingBoxConfigPath == "" {
			problems = append(problems, "singbox_config_path is required when singbox_auto_start=true")
		}
		// Policy B: adoption may be used even when auto-start is disabled.
		if c.AdoptExternal() && strings.TrimSpace(c.SingBoxConfigPath) == "" {
			problems = append(problems, "singbox_config_path is required when singbox_adopt_external=true (needed to adopt external process)")
		}
		if c.SingBoxStartTimeout < 1*time.Second {
			problems = append(problems, "singbox_start_timeout must be >= 1s")
		}
	}

	// if c.VPNRouterUpPath == "" {
	//	problems = append(problems, "vpn_router_up_path is required")
	// }
	// if c.VPNRouterDownPath == "" {
	//	problems = append(problems, "vpn_router_down_path is required")
	// }
	// Scripts: required + must exist + must be executable
	if strings.TrimSpace(c.VPNRouterSetupPath) == "" {
		problems = append(problems, "vpn_router_setup_path is required")
	} else if err := mustBeExecutableFile(c.VPNRouterSetupPath); err != nil {
		problems = append(problems, fmt.Sprintf("vpn_router_setup_path invalid: %v", err))
	}
	if strings.TrimSpace(c.VPNRouterPFApplyPath) == "" {
		problems = append(problems, "vpn_router_pf_apply_path is required")
	} else if err := mustBeExecutableFile(c.VPNRouterPFApplyPath); err != nil {
		problems = append(problems, fmt.Sprintf("vpn_router_pf_apply_path invalid: %v", err))
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

func mustBeExecutableFile(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%q not accessible: %w", path, err)
	}
	if fi.IsDir() {
		return fmt.Errorf("%q is a directory (expected a file)", path)
	}
	// Require at least one execute bit (owner/group/other)
	if fi.Mode()&0o111 == 0 {
		return fmt.Errorf("%q is not executable (run: chmod +x %s)", path, path)
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
