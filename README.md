# vpnrd

**vpnrd** is a macOS-based VPN router daemon that supervises a `sing-box` TUN tunnel and enforces a strict `pf`-based kill-switch for LAN clients.

It is designed for environments where:
- network interruptions are common,
- policy enforcement must be fail-closed,
- routing control must be explicit and auditable.

This project turns a macOS machine into a deterministic VPN gateway.

---

## Design Goals

- Fail-closed network posture
- Explicit packet policy via `pf`
- Deterministic state transitions
- Recoverable tunnel lifecycle
- Minimal hidden automation
- Operator visibility over magic

vpnrd is not a GUI VPN client.  
It is infrastructure software.

---

## Core Responsibilities

### 1. Tunnel Supervision

- Runs and supervises `sing-box`
- Verifies TUN interface state
- Monitors health signals
- Restarts or recovers as required

### 2. Policy Enforcement (Kill-Switch)

`pf` rules are applied through controlled anchors to ensure:

- LAN traffic is NATed only through the tunnel
- WAN access is denied unless explicitly allowed
- If the tunnel fails → traffic is blocked (fail-closed)
- No silent fallback to direct WAN

### 3. Router Mode

When configured as a gateway:

- LAN clients route through the macOS host
- NAT is performed via `pf`
- Traffic segmentation is explicit
- Optional minimal WAN allowlist may be applied (VPN endpoint, DNS, NTP)

---

## Architectural Overview

LAN Clients
│
▼
macOS (vpnrd)
│
├── sing-box (TUN interface utunX)
│
└── pf (NAT + filtering + kill-switch)
│
▼
WAN


vpnrd manages state.  
`sing-box` manages encrypted transport.  
`pf` enforces packet policy.

Each component has a clearly separated responsibility.

---

## Failure Model

vpnrd assumes hostile or unstable networks.

If any of the following occurs:
- TUN interface disappears
- Health checks fail
- sing-box exits unexpectedly

Then:

- pf remains in a restrictive state
- LAN traffic is denied
- Recovery logic is triggered
- No implicit WAN fallback occurs

The system prioritizes integrity over availability.

---

## Why pf?

Application-level VPN clients protect the local machine.

vpnrd uses `pf` because:

- Packet filtering must be kernel-enforced
- Policy must be explicit and inspectable
- Kill-switch must not depend on application state alone
- Router-mode requires deterministic NAT + filtering

---

## Operational Philosophy

vpnrd favors:

- explicit configuration
- predictable behavior
- auditability
- minimal hidden side effects

It avoids:

- automatic policy drift
- silent route changes
- background heuristics without operator visibility

---

## Intended Use Cases

- Dedicated macOS VPN router node
- Segmented LAN environments
- Fail-closed network setups
- Research / lab environments requiring strict tunnel enforcement

---

## Status

Early public release.  
Core tunnel supervision and pf kill-switch logic implemented.

Future work:
- multi-uplink supervision
- richer health signal evaluation
- structured event logging
- bonding layer integration

---

## Warning

vpnrd modifies system routing and `pf` rules.

Incorrect configuration can disrupt connectivity.

Use on a dedicated test node before production deployment.
