#!/bin/bash
#
# vpn_router_pf_apply.sh
# FAST apply of pf "vpn" rules (safe for frequent recovery calls).
# It ONLY rewrites the vpnrd_vpn anchor file and reloads pf.
#
# Args:
#   $1 = VPN_IF (utunX)            [required]
#   $2 = WAN_IF (e.g. en5)         [optional; default en5]
#   $3 = LAN_IF (e.g. en8)         [optional; default en8]
#   $4 = VPN_SERVER_IPS CSV        [required; e.g. "1.2.3.4,5.6.7.8"]
#   $5 = WAN_DNS_IPS CSV           [optional; e.g. "1.1.1.1,8.8.8.8"]
#   $6 = ALLOW_WAN_NTP             [optional; "true" or "false"]

set -e

echo "=== PF APPLY (fast) ==="

#-VPN_IF="$1"
#-WAN_IF="${2:-en5}"
#-LAN_IF="${3:-en8}"
#-VPN_SERVER_IPS_CSV="$4"
#-WAN_DNS_IPS_CSV="${5:-}"
#-ALLOW_WAN_NTP="${6:-false}"

 # vpnrd may pass either positional values:
 #   utun66 en5 en8 "89.40.206.121" "1.1.1.1,8.8.8.8" true
 # or key=value:
 #   utun=utun66 wan=en5 lan=en8 vpn_server_ips=... wan_dns=... allow_ntp=true
 strip_kv() {
   case "${1:-}" in
     *=*) echo "${1#*=}" ;;
     *)   echo "${1:-}" ;;
   esac
 }

 VPN_IF="$(strip_kv "${1:-}")"
 WAN_IF="$(strip_kv "${2:-en5}")"
 LAN_IF="$(strip_kv "${3:-en8}")"
 VPN_SERVER_IPS_CSV="$(strip_kv "${4:-}")"
 WAN_DNS_IPS_CSV="$(strip_kv "${5:-}")"
 ALLOW_WAN_NTP="$(strip_kv "${6:-false}")"

LAN_CIDR="192.168.50.0/24"
LAN_IP="192.168.50.1"

PF_ANCHOR_VPN="/etc/pf.anchors/vpnrd_vpn"

if [ -z "$VPN_IF" ]; then
  echo "ERROR: VPN interface not provided (expected utunX)"
  exit 1
fi

# Wait briefly for utun to appear (sing-box can create it asynchronously).
deadline=$((SECONDS+5))
while ! ifconfig "$VPN_IF" >/dev/null 2>&1; do
  if (( SECONDS >= deadline )); then
    echo "ERROR: Provided VPN interface utun=$VPN_IF does not exist" >&2
    exit 1
  fi
  sleep 0.1
done

if [ -z "$VPN_SERVER_IPS_CSV" ]; then
  echo "ERROR: VPN server IP list is empty (pass CSV in arg #4)"
  exit 1
fi

echo "VPN: $VPN_IF  WAN: $WAN_IF  LAN: $LAN_IF"
echo "VPN servers (WAN allowlist): $VPN_SERVER_IPS_CSV"

# Convert CSV -> pf table list: "1.2.3.4,5.6.7.8" -> "1.2.3.4 5.6.7.8"
# pf tables want space-separated entries inside { ... } (commas break parsing)
VPN_SERVER_IPS_PF="$(printf "%s" "$VPN_SERVER_IPS_CSV" | tr ', ' ' ' | xargs)"

WAN_DNS_IPS_PF=""
if [ -n "$WAN_DNS_IPS_CSV" ]; then
  echo "WAN DNS allowed: $WAN_DNS_IPS_CSV"
  # "1.1.1.1,8.8.8.8" -> "1.1.1.1 8.8.8.8"
  WAN_DNS_IPS_PF="$(printf "%s" "$WAN_DNS_IPS_CSV" | tr ', ' ' ' | xargs)"
fi

echo "Writing dynamic anchor: $PF_ANCHOR_VPN ..."

sudo tee "$PF_ANCHOR_VPN" >/dev/null <<EOF
# vpnrd_vpn (DYNAMIC)
# Rewritten by vpn_router_pf_apply.sh
# VPN: $VPN_IF
# WAN: $WAN_IF
# LAN: $LAN_IF

# Tables (WAN allowlist)
table <vpnrd_vpn_servers> persist { $VPN_SERVER_IPS_PF }
EOF

# Optional WAN DNS table
if [ -n "$WAN_DNS_IPS_PF" ]; then
  sudo tee -a "$PF_ANCHOR_VPN" >/dev/null <<EOF
table <vpnrd_wan_dns> persist { $WAN_DNS_IPS_PF }
EOF
fi

# Rules block (append)
sudo tee -a "$PF_ANCHOR_VPN" >/dev/null <<EOF

# --- NAT ---
# NAT LAN -> VPN tunnel
nat on $VPN_IF from $LAN_CIDR to any -> ($VPN_IF)

# --- LAN -> VPN allowed ---
pass in  quick on $LAN_IF inet from $LAN_CIDR to any keep state
pass out quick on $VPN_IF inet from $LAN_CIDR to any keep state

# --- This Mac: allow WAN ONLY to VPN servers (to maintain the tunnel) ---
pass out quick on $WAN_IF inet from ($WAN_IF) to <vpnrd_vpn_servers> keep state

EOF

# Optional: allow WAN DNS (helpful before tunnel comes up)
if [ -n "$WAN_DNS_IPS_PF" ]; then
  sudo tee -a "$PF_ANCHOR_VPN" >/dev/null <<EOF
pass out quick on $WAN_IF inet proto { udp, tcp } from ($WAN_IF) to <vpnrd_wan_dns> port 53 keep state
EOF
fi

# Optional: allow WAN NTP
if [ "$ALLOW_WAN_NTP" = "true" ]; then
  echo "WAN NTP allowed: true"
  sudo tee -a "$PF_ANCHOR_VPN" >/dev/null <<EOF
pass out quick on $WAN_IF inet proto udp from ($WAN_IF) to any port 123 keep state
EOF
else
  echo "WAN NTP allowed: false"
fi

echo "Reloading pf..."
sudo pfctl -f /etc/pf.conf

echo
echo "=== PF APPLY complete ==="
echo "Useful checks:"
echo "  sudo pfctl -s nat"
echo "  sudo pfctl -s rules | head -n 60"

