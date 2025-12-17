#!/bin/bash
set -euo pipefail

# Configure ClickHouse TTL for system log tables
# This prevents system log tables from consuming excessive disk space
# Usage: ./configure-clickhouse-ttl.sh [--namespace langfuse] [--password clickhouse123]

NAMESPACE="langfuse"
PASSWORD=""
RETENTION_DAYS=7

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --namespace|-n)
      NAMESPACE="$2"
      shift 2
      ;;
    --password|-p)
      PASSWORD="$2"
      shift 2
      ;;
    --retention-days|-r)
      RETENTION_DAYS="$2"
      shift 2
      ;;
    --help|-h)
      echo "Usage: $0 [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --namespace, -n       Kubernetes namespace (default: langfuse)"
      echo "  --password, -p        ClickHouse password (auto-detected from secret if not provided)"
      echo "  --retention-days, -r  Number of days to retain logs (default: 7)"
      echo "  --help, -h            Show this help message"
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Auto-detect password if not provided
if [ -z "$PASSWORD" ]; then
  echo "Auto-detecting ClickHouse password from secret..."
  PASSWORD=$(kubectl get secret -n "$NAMESPACE" langfuse-clickhouse -o jsonpath='{.data.admin-password}' 2>/dev/null | base64 -d || echo "")
  if [ -z "$PASSWORD" ]; then
    echo "❌ Could not auto-detect password. Please provide --password"
    exit 1
  fi
  echo "   ✓ Password retrieved from secret"
fi

# Find ClickHouse pod
POD_NAME=$(kubectl get pod -n "$NAMESPACE" -l app.kubernetes.io/component=clickhouse,app.kubernetes.io/instance=langfuse -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -z "$POD_NAME" ]; then
  echo "❌ Could not find ClickHouse pod in namespace $NAMESPACE"
  exit 1
fi

echo "Configuring ClickHouse TTL on pod: $POD_NAME"
echo "   Namespace: $NAMESPACE"
echo "   Retention: $RETENTION_DAYS days"
echo ""

# Wait for ClickHouse to be ready
echo "⏳ Waiting for ClickHouse to be ready..."
kubectl wait --namespace "$NAMESPACE" \
  --for=condition=ready \
  --timeout=120s \
  pod/"$POD_NAME" &>/dev/null || true
echo "   ✓ ClickHouse is ready"
echo ""

# Configure TTL on system log tables
echo "Configuring TTL for system log tables..."

# Tables with event_date column
TABLES_WITH_EVENT_DATE=(
  "trace_log"
  "text_log"
  "metric_log"
  "asynchronous_metric_log"
  "query_log"
  "error_log"
  "part_log"
  "query_metric_log"
  "latency_log"
  "processors_profile_log"
  "asynchronous_insert_log"
  "query_thread_log"
  "query_views_log"
  "crash_log"
  "session_log"
  "zookeeper_log"
)

# Build multi-query for event_date tables
SQL_COMMANDS=""
for table in "${TABLES_WITH_EVENT_DATE[@]}"; do
  SQL_COMMANDS+="ALTER TABLE system.$table MODIFY TTL event_date + INTERVAL $RETENTION_DAYS DAY;"$'\n'
done

# Special case: opentelemetry_span_log uses finish_date
SQL_COMMANDS+="ALTER TABLE system.opentelemetry_span_log MODIFY TTL finish_date + INTERVAL $RETENTION_DAYS DAY;"

# Execute all TTL commands
kubectl exec -n "$NAMESPACE" "$POD_NAME" -- clickhouse-client \
  --password="$PASSWORD" \
  --multiquery \
  --query "$SQL_COMMANDS" 2>&1 | grep -v "Table.*doesn't exist" || true

echo "   ✓ TTL configured for all system log tables"
echo ""

# Verify configuration
echo "Verifying TTL settings..."
VERIFICATION=$(kubectl exec -n "$NAMESPACE" "$POD_NAME" -- clickhouse-client \
  --password="$PASSWORD" \
  --query "
SELECT table,
       formatReadableSize(sum(bytes)) AS current_size,
       multiIf(
         table = 'opentelemetry_span_log', 'finish_date + $RETENTION_DAYS days',
         table LIKE '%_log', 'event_date + $RETENTION_DAYS days',
         'none'
       ) AS ttl_setting
FROM system.parts
WHERE database = 'system'
  AND table LIKE '%_log'
  AND active = 1
GROUP BY table
ORDER BY sum(bytes) DESC
LIMIT 5
FORMAT PrettyCompact
")

echo "$VERIFICATION"
echo ""
echo "✅ ClickHouse TTL configuration complete!"
echo ""
echo "System log tables will now automatically delete data older than $RETENTION_DAYS days."
echo "This prevents disk space issues while retaining recent logs for debugging."
