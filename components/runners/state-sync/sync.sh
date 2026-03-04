#!/bin/bash
# sync.sh - Sidecar script to sync session state to S3 every N seconds

set -e

# Configuration from environment
S3_ENDPOINT="${S3_ENDPOINT:-http://minio.ambient-code.svc:9000}"
S3_BUCKET="${S3_BUCKET:-ambient-sessions}"
NAMESPACE="${NAMESPACE:-default}"
SESSION_NAME="${SESSION_NAME:-unknown}"
SYNC_INTERVAL="${SYNC_INTERVAL:-60}"
MAX_SYNC_SIZE="${MAX_SYNC_SIZE:-1073741824}"  # 1GB default

# Sanitize inputs to prevent path traversal
NAMESPACE="${NAMESPACE//[^a-zA-Z0-9-]/}"
SESSION_NAME="${SESSION_NAME//[^a-zA-Z0-9-]/}"

# Paths to sync (non-git content)
# Note: .claude uses /app/.claude (SubPath mount), others use /workspace
SYNC_PATHS=(
    "artifacts"
    "file-uploads"
)
CLAUDE_DATA_PATH="/app/.claude"

# Patterns to exclude from sync
EXCLUDE_PATTERNS=(
    "repos/**"           # Git handles this
    "node_modules/**"
    ".venv/**"
    "__pycache__/**"
    ".cache/**"
    "*.pyc"
    "target/**"
    "dist/**"
    "build/**"
    ".git/**"
    ".claude/debug/**"   # Debug logs with symlinks that break rclone
)

# Configure rclone for S3
setup_rclone() {
    # Use explicit /tmp path since HOME may not be set in container
    mkdir -p /tmp/.config/rclone
    cat > /tmp/.config/rclone/rclone.conf << EOF
[s3]
type = s3
provider = Other
access_key_id = ${AWS_ACCESS_KEY_ID}
secret_access_key = ${AWS_SECRET_ACCESS_KEY}
endpoint = ${S3_ENDPOINT}
acl = private
EOF
    # Protect config file with credentials
    chmod 600 /tmp/.config/rclone/rclone.conf
}

# Check total size before sync
check_size() {
    local total=0
    
    # Check .claude directory size (at /app/.claude via SubPath)
    if [ -d "${CLAUDE_DATA_PATH}" ]; then
        size=$(du -sb "${CLAUDE_DATA_PATH}" 2>/dev/null | cut -f1 || echo 0)
        total=$((total + size))
    fi
    
    # Check other paths in /workspace
    for path in "${SYNC_PATHS[@]}"; do
        if [ -d "/workspace/${path}" ]; then
            size=$(du -sb "/workspace/${path}" 2>/dev/null | cut -f1 || echo 0)
            total=$((total + size))
        fi
    done
    
    if [ $total -gt $MAX_SYNC_SIZE ]; then
        echo "WARNING: Sync size (${total} bytes) exceeds limit (${MAX_SYNC_SIZE} bytes)"
        echo "Some files may be skipped"
        return 1
    fi
    return 0
}

# Sync workspace state to S3
sync_to_s3() {
    local s3_path="s3:${S3_BUCKET}/${NAMESPACE}/${SESSION_NAME}"
    
    echo "[$(date -Iseconds)] Starting sync to S3..."
    
    local synced=0
    
    # Sync .claude data from /app/.claude (SubPath mount matches runner container)
    if [ -d "${CLAUDE_DATA_PATH}" ]; then
        echo "  Syncing .claude/..."
        if rclone --config /tmp/.config/rclone/rclone.conf sync "${CLAUDE_DATA_PATH}" "${s3_path}/.claude/" \
            --checksum \
            --copy-links \
            --transfers 4 \
            --fast-list \
            --stats-one-line \
            --max-size ${MAX_SYNC_SIZE} \
            $(printf -- '--exclude %s ' "${EXCLUDE_PATTERNS[@]}") \
            2>&1; then
            synced=$((synced + 1))
        else
            echo "  Warning: sync of .claude had errors"
        fi
    fi
    
    # Sync other paths from /workspace
    for path in "${SYNC_PATHS[@]}"; do
        if [ -d "/workspace/${path}" ]; then
            echo "  Syncing ${path}/..."
            if rclone --config /tmp/.config/rclone/rclone.conf sync "/workspace/${path}" "${s3_path}/${path}/" \
                --checksum \
                --copy-links \
                --transfers 4 \
                --fast-list \
                --stats-one-line \
                --max-size ${MAX_SYNC_SIZE} \
                $(printf -- '--exclude %s ' "${EXCLUDE_PATTERNS[@]}") \
                2>&1; then
                synced=$((synced + 1))
            else
                echo "  Warning: sync of ${path} had errors"
            fi
        fi
    done
    
    # Save metadata
    echo "{\"lastSync\": \"$(date -Iseconds)\", \"session\": \"${SESSION_NAME}\", \"namespace\": \"${NAMESPACE}\", \"pathsSynced\": ${synced}}" > /tmp/metadata.json
    rclone --config /tmp/.config/rclone/rclone.conf copy /tmp/metadata.json "${s3_path}/" 2>&1 || true
    
    echo "[$(date -Iseconds)] Sync complete (${synced} paths synced)"
}

# Backup git repo state (bundles, patches, metadata) to a temp directory for S3 upload.
# Only called during final_sync() to avoid overhead during periodic syncs.
backup_git_repos() {
    local repo_state_dir="/tmp/repo-state"
    local s3_path="s3:${S3_BUCKET}/${NAMESPACE}/${SESSION_NAME}"

    # Clean up any previous state
    rm -rf "${repo_state_dir}"
    mkdir -p "${repo_state_dir}"

    if [ ! -d "/workspace/repos" ]; then
        echo "  No /workspace/repos directory, skipping git backup"
        return 0
    fi

    local backed_up=0

    for repo_dir in /workspace/repos/*/; do
        # Skip if not a directory
        [ -d "${repo_dir}" ] || continue

        # Strip trailing slash for consistent git safe.directory matching
        repo_dir="${repo_dir%/}"

        # Skip if not a git repo (.git can be a directory or a file for worktrees/submodules)
        if [ ! -d "${repo_dir}/.git" ] && [ ! -f "${repo_dir}/.git" ]; then
            echo "  Skipping ${repo_dir}: not a git repo"
            continue
        fi

        local repo_name
        repo_name=$(basename "${repo_dir}")
        local dest="${repo_state_dir}/${repo_name}"
        mkdir -p "${dest}"

        echo "  Backing up git state for ${repo_name}..."

        # Mark directory as safe for git operations
        if ! git config --global --add safe.directory "${repo_dir}" 2>&1; then
            echo "  WARNING: Failed to mark ${repo_dir} as safe directory"
        fi

        # Create git bundle with all refs
        local bundle_err
        if bundle_err=$(git -C "${repo_dir}" bundle create "${dest}/repo.bundle" --all 2>&1); then
            echo "  Bundle created for ${repo_name} ($(stat -c%s "${dest}/repo.bundle" 2>/dev/null || echo "?") bytes)"
        else
            echo "  WARNING: Failed to create bundle for ${repo_name}: ${bundle_err}"
            # Still save metadata for empty repos
        fi

        # Capture uncommitted changes (tracked files)
        git -C "${repo_dir}" diff HEAD > "${dest}/uncommitted.patch" 2>/dev/null || true

        # Capture staged changes
        git -C "${repo_dir}" diff --cached > "${dest}/staged.patch" 2>/dev/null || true

        # Write metadata
        local remote_url branch head_sha local_branches
        remote_url=$(git -C "${repo_dir}" remote get-url origin 2>&1 || echo "")
        # Strip embedded credentials (e.g., x-access-token:TOKEN@) from URL
        remote_url=$(echo "${remote_url}" | sed 's|://[^@]*@|://|')
        branch=$(git -C "${repo_dir}" rev-parse --abbrev-ref HEAD 2>&1 || echo "unknown")
        head_sha=$(git -C "${repo_dir}" rev-parse HEAD 2>&1 || echo "")
        local_branches=$(git -C "${repo_dir}" branch --format='%(refname:short)' 2>&1 | jq -R -s 'split("\n") | map(select(length > 0))' || echo "[]")

        jq -n \
            --arg url "${remote_url}" \
            --arg branch "${branch}" \
            --arg sha "${head_sha}" \
            --argjson branches "${local_branches}" \
            --arg ts "$(date -Iseconds)" \
            '{remoteUrl: $url, currentBranch: $branch, headSha: $sha, localBranches: $branches, backedUpAt: $ts}' \
            > "${dest}/metadata.json"

        backed_up=$((backed_up + 1))
        echo "  Backed up ${repo_name} (branch: ${branch}, HEAD: ${head_sha:0:8})"
    done

    if [ "${backed_up}" -gt 0 ]; then
        echo "  Syncing ${backed_up} repo state(s) to S3..."
        if rclone --config /tmp/.config/rclone/rclone.conf sync "${repo_state_dir}/" "${s3_path}/repo-state/" \
            --transfers 4 \
            --fast-list \
            --stats-one-line \
            2>&1; then
            echo "  Git repo state backup complete (${backed_up} repos)"
        else
            echo "  WARNING: Failed to sync git repo state to S3"
        fi
    else
        echo "  No git repos to back up"
    fi

    # Clean up temp directory
    rm -rf "${repo_state_dir}"
}

# Final sync on shutdown
final_sync() {
    echo ""
    echo "========================================="
    echo "[$(date -Iseconds)] SIGTERM received, performing final sync..."
    echo "========================================="
    backup_git_repos
    sync_to_s3
    echo "========================================="
    echo "[$(date -Iseconds)] Final sync complete, exiting"
    echo "========================================="
    exit 0
}

# Set HOME for git config (alpine doesn't set it by default)
export HOME=/tmp

# Main
echo "========================================="
echo "Ambient Code State Sync Sidecar"
echo "========================================="
echo "Session: ${NAMESPACE}/${SESSION_NAME}"
echo "S3 Endpoint: ${S3_ENDPOINT}"
echo "S3 Bucket: ${S3_BUCKET}"
echo "Sync interval: ${SYNC_INTERVAL}s"
echo "Max sync size: ${MAX_SYNC_SIZE} bytes"
echo "========================================="

# Check if S3 is configured
if [ -z "${S3_ENDPOINT}" ] || [ -z "${S3_BUCKET}" ] || [ -z "${AWS_ACCESS_KEY_ID}" ] || [ -z "${AWS_SECRET_ACCESS_KEY}" ]; then
    echo "S3 not configured - state sync disabled (ephemeral storage only)"
    echo "Session will not persist across pod restarts"
    echo "========================================="
    # Sleep forever - keep sidecar alive but do nothing
    while true; do
        sleep 3600
    done
fi

setup_rclone
trap 'final_sync' SIGTERM SIGINT

# Initial delay to let workspace populate
echo "Waiting 30s for workspace to populate..."
sleep 30

# Main sync loop
while true; do
    check_size || echo "Size check warning (continuing anyway)"
    sync_to_s3 || echo "Sync failed, will retry in ${SYNC_INTERVAL}s..."
    sleep ${SYNC_INTERVAL}
done

