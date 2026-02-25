# vpnrd — VPN Router Daemon for macOS (sing-box + pf kill-switch)

vpnrd is a local VPN router / watchdog daemon for macOS.  
It supervises a `sing-box` TUN tunnel and enforces a **pf-based kill-switch** so LAN clients only have internet access **through the tunnel**.

This project is designed for hostile networks where short disconnects are common and policy enforcement must be strict.

---

## What vpnrd does

- **Runs and supervises sing-box**
  - TUN inbound (utun interface)
  - controlled start/stop
  - health checking

- **Enforces a pf kill-switch (router mode)**
  - LAN clients are NATed through the tunnel
  - if the tunnel is down → LAN clients lose internet (fail-closed)
  - optional minimal WAN allowlist (VPN server IPs, WAN DNS/NTP if desired)

- **Self-heals**
  - re-applies pf rules when needed
  - recovers from transient failures without manual pfctl surgery

---

## Why pf (and not “just a VPN app”)

A VPN app protects the local Mac.  
vpnrd protects the **entire LAN behind the Mac** by turning the Mac into a strict policy router:
- explicit allow/deny rules
- predictable fail-closed behavior
- auditable, reproducible configuration

---

## High-level architecture

- `vpnrd` (Go daemon)
  - lifecycle + watchdog
  - state machine (up/down/recover)
  - applies pf anchors and sysctl/network settings

- `sing-box` (tunnel engine)
  - produces `utunX` interface
  - handles routing inside the tunnel

- `pf` (policy enforcement)
  - NAT + filtering + allowlist
  - kill-switch rules anchored for clean enable/disable

---

## Safety model (kill-switch)

**Default stance:** deny.  
Only allow:
1) traffic required to establish/maintain the tunnel (optional allowlist)
2) traffic through the tunnel interface once it is healthy

If the tunnel goes down or becomes unhealthy:
- pf remains in a fail-closed state
- LAN clients lose WAN until recovery succeeds

---

## Repo structure (typical)

- `cmd/vpnrd/` — daemon entrypoint
- `internal/` — configuration, watchdog, pf management, health checks
- `dist/` — packaged configs/templates (if used)
- `.config/` — example configs (if included)

---

## Development notes

This repo is intentionally CLI-first and ops-friendly:
- explicit configs
- minimal surprises
- designed to be run headless on a dedicated Mac (router node)

---

## Roadmap

- stronger health model (multi-signal: DNS/TCP/route/interface checks)
- structured events/audit logs of policy state
- “WAN minimal allowlist” helper tooling
- future: multi-uplink / bonding layer integration

---

## Disclaimer

This project configures system networking and pf rules.
Use on a test machine first. Misconfiguration can cut off network access.

