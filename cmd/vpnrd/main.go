package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexgoodkarma/vpn-router-daemon/internal/config"
	"github.com/alexgoodkarma/vpn-router-daemon/internal/control"
)

const version = "0.2.0"

func usage() {
	fmt.Fprintf(os.Stderr, `vpnrd - Sing-box pf NAT VPN router daemon

Usage:
  vpnrd up        - start VPN router (sing-box + pf NAT)
  vpnrd down      - stop VPN router and restore normal state
  vpnrd run       - run watchdog daemon (keeps tunnel healthy)
  vpnrd status    - show current status
  vpnrd -h        - show help

`)
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	flag.Usage = usage
	showVersion := flag.Bool("version", false, "show version and exit")

	// Global flag: config path
	defaultCfg, _ := config.DefaultPath()
	cfgPath := flag.String("config", defaultCfg, "path to config file")

	flag.Parse()

	if *showVersion {
		fmt.Printf("vpnrd version %s\n", version)
		return
	}

	if flag.NArg() < 1 {
		usage()
		os.Exit(1)
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Printf("config load failed: %v", err)
		os.Exit(1)
	}

	cmd := flag.Arg(0)

	switch cmd {
	case "up":
		if err := cmdUp(cfg); err != nil {
			log.Fatalf("up failed: %v", err)
		}
	case "down":
		if err := cmdDown(cfg); err != nil {
			log.Fatalf("down failed: %v", err)
		}
	case "run":
		if err := cmdRun(cfg); err != nil {
			log.Fatalf("run failed: %v", err)
		}
	case "status":
		if err := cmdStatus(cfg); err != nil {
			log.Fatalf("status failed: %v", err)
		}
	default:
		log.Printf("unknown command: %q\n", cmd)
		usage()
		os.Exit(1)
	}
}

func cmdUp(cfg *config.Config) error {
	res, err := control.RunScript(context.Background(), cfg.VPNRouterUpPath, cfg.CommandTimeout)
	if err != nil {
		return formatScriptFailure("up", res, err)
	}
	printScriptSuccess("up", res)
	return nil
}

func cmdDown(cfg *config.Config) error {
	res, err := control.RunScript(context.Background(), cfg.VPNRouterDownPath, cfg.CommandTimeout)
	if err != nil {
		return formatScriptFailure("down", res, err)
	}
	printScriptSuccess("down", res)
	return nil
}

func cmdRun(cfg *config.Config) error {
	// Placeholder; next step we implement the watchdog loop.
	log.Printf("daemon mode not implemented yet; check_interval=%s health_url=%s",
		cfg.CheckInterval, cfg.HealthCheckURL)

	t := time.NewTicker(cfg.CheckInterval)
	defer t.Stop()

	for range t.C {
		log.Printf("(dummy) would health-check: %s", cfg.HealthCheckURL)
	}
	return nil
}

func cmdStatus(cfg *config.Config) error {
	// Placeholder; next step we’ll check sing-box process + pf + current IP.
	log.Printf("status not implemented yet")
	return nil
}

// helper functions

func printScriptSuccess(tag string, res *control.Result) {
	// Minimal user-friendly output.
	// Logs already contain full details.
	if res == nil {
		fmt.Printf("[vpnrd] %s: ok\n", tag)
		return
	}

	if res.Stdout != "" && res.Stderr != "" {
		fmt.Printf("[vpnrd] %s: ok\nstdout:\n%s\nstderr:\n%s\n", tag, res.Stdout, res.Stderr)
		return
	}
	if res.Stdout != "" {
		fmt.Printf("[vpnrd] %s: ok\n%s\n", tag, res.Stdout)
		return
	}
	if res.Stderr != "" {
		fmt.Printf("[vpnrd] %s: ok\n%s\n", tag, res.Stderr)
		return
	}

	fmt.Printf("[vpnrd] %s: ok\n", tag)
}

func formatScriptFailure(tag string, res *control.Result, err error) error {
	// Build a rich error message that includes captured outputs.
	if res == nil {
		return fmt.Errorf("%s: %w", tag, err)
	}

	msg := fmt.Sprintf("%s failed: %v (exit=%d)", tag, err, res.ExitCode)
	if res.Stdout != "" {
		msg += "\nstdout:\n" + res.Stdout
	}
	if res.Stderr != "" {
		msg += "\nstderr:\n" + res.Stderr
	}
	return fmt.Errorf("%s", msg)
}
