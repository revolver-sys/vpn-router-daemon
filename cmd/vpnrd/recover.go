package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/revolver-sys/vpn-router-daemon/internal/config"
	"github.com/revolver-sys/vpn-router-daemon/internal/control"
	"github.com/revolver-sys/vpn-router-daemon/internal/debugdump"
	"github.com/revolver-sys/vpn-router-daemon/internal/singboxctl"
)

func doRecovery(ctx context.Context, cfg *config.Config, effectiveWAN, effectiveLAN string) error {
	sb0, _ := singboxctl.Inspect(cfg)
	debugdump.Dump("singbox_before_recover", sb0)

	// Only restart if we own it. Never kill an external sing-box.
	if sb0 != nil && sb0.OwnedByUs {
		if err := singboxctl.StopIfOwned(cfg); err != nil {
			return fmt.Errorf("stop sing-box (owned): %w", err)
		}
	}

	sb, err := singboxctl.EnsureRunning(ctx, cfg, cfg.SingBoxStartTimeout)
	if err != nil {
		return fmt.Errorf("ensure sing-box: %w", err)
	}
	debugdump.Dump("singbox_after_ensure", sb)
	if sb == nil || !sb.Running || sb.NewUTUN == "" {
		return fmt.Errorf("sing-box not running or utun not detected")
	}

	args := []string{
		fmt.Sprintf("utun=%s", sb.NewUTUN),
		fmt.Sprintf("wan=%s", strings.TrimSpace(effectiveWAN)),
		fmt.Sprintf("lan=%s", strings.TrimSpace(effectiveLAN)),
		fmt.Sprintf("vpn_server_ips=%q", strings.Join(cfg.VPNServerIPs, ",")),
		fmt.Sprintf("wan_dns=%q", strings.Join(cfg.WANDNSIPs, ",")),
		fmt.Sprintf("allow_ntp=%t", cfg.AllowWANNTP),
	}
	_, err = control.RunScript(ctx, cfg.VPNRouterPFApplyPath, cfg.CommandTimeout, args...)

	if err != nil {
		return fmt.Errorf("pf_apply: %w", err)
	}
	return nil
}
