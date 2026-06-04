#!/bin/bash
set -euo pipefail

CLAUDE_BIN="${CLAUDE_CLI_PATH:-/usr/local/bin/claude}"

if [[ "${OPENSHELL_ENABLED:-}" == "true" ]]; then
  # Clean up stale sandbox network namespaces left by a previous session.
  # OpenShell uses hardcoded 10.200.0.0/24 for its veth pair; leftover
  # namespaces from crashed/killed supervisors poison the routing table
  # and prevent the new sandbox's proxy from receiving connections.
  for ns in /var/run/netns/sandbox-*; do
    [[ -e "$ns" ]] || continue
    ns_name="$(basename "$ns")"
    echo "Cleaning stale sandbox netns: $ns_name" >&2
    ip link del "veth-h-${ns_name#sandbox-}" 2>/dev/null || true
    ip netns del "$ns_name" 2>/dev/null || true
  done

  # Disable reverse-path filtering so ARP resolves on veth interfaces created
  # by the supervisor. New interfaces inherit net.ipv4.conf.default.rp_filter.
  # Requires SYS_ADMIN capability; /proc/sys is mounted read-only by the
  # container runtime so we remount it read-write first.
  mount -o remount,rw /proc/sys 2>/dev/null || true
  for rp in default all; do
    echo 0 > "/proc/sys/net/ipv4/conf/${rp}/rp_filter" 2>/dev/null || true
  done

  exec /openshell-sandbox \
    --policy-rules "${OPENSHELL_POLICY_RULES:-/etc/openshell/policy.rego}" \
    --policy-data "${OPENSHELL_POLICY_DATA:-/etc/openshell/policy.yaml}" \
    --log-level "${OPENSHELL_LOG_LEVEL:-warn}" \
    -- "$CLAUDE_BIN" "$@"
else
  exec "$CLAUDE_BIN" "$@"
fi
