#!/bin/bash
# hydrate.sh - Init container script to download session state from S3

set -e

# Configuration from environment
S3_ENDPOINT="${S3_ENDPOINT:-http://minio.ambient-code.svc:9000}"
S3_BUCKET="${S3_BUCKET:-ambient-sessions}"
NAMESPACE="${NAMESPACE:-default}"
SESSION_NAME="${SESSION_NAME:-unknown}"

# Sanitize inputs to prevent path traversal
NAMESPACE="${NAMESPACE//[^a-zA-Z0-9-]/}"
SESSION_NAME="${SESSION_NAME//[^a-zA-Z0-9-]/}"

# Paths to sync (must match sync.sh)
# Note: .claude uses /app/.claude (SubPath mount), others use /workspace
SYNC_PATHS=(
    "artifacts"
    "file-uploads"
)
CLAUDE_DATA_PATH="/app/.claude"

# Error handler
error_exit() {
    echo "ERROR: $1" >&2
    exit 1
}

# Configure rclone for S3
setup_rclone() {
    # Use explicit /tmp path since HOME may not be set in container
    mkdir -p /tmp/.config/rclone || error_exit "Failed to create rclone config directory"
    cat > /tmp/.config/rclone/rclone.conf << EOF
[s3]
type = s3
provider = Other
access_key_id = ${AWS_ACCESS_KEY_ID}
secret_access_key = ${AWS_SECRET_ACCESS_KEY}
endpoint = ${S3_ENDPOINT}
acl = private
EOF
    if [ $? -ne 0 ]; then
        error_exit "Failed to write rclone configuration"
    fi
    # Protect config file with credentials
    chmod 600 /tmp/.config/rclone/rclone.conf || error_exit "Failed to secure rclone config"
}

echo "========================================="
echo "Ambient Code Session State Hydration"
echo "========================================="
echo "Session: ${NAMESPACE}/${SESSION_NAME}"
echo "S3 Endpoint: ${S3_ENDPOINT}"
echo "S3 Bucket: ${S3_BUCKET}"
echo "========================================="

# Create workspace structure
echo "Creating workspace structure..."
# .claude is mounted at /app/.claude via SubPath (same location as runner container)
mkdir -p "${CLAUDE_DATA_PATH}" || error_exit "Failed to create .claude directory"
mkdir -p "${CLAUDE_DATA_PATH}/debug" || error_exit "Failed to create .claude/debug directory"
mkdir -p /workspace/artifacts || error_exit "Failed to create artifacts directory"
mkdir -p /workspace/file-uploads || error_exit "Failed to create file-uploads directory"
mkdir -p /workspace/repos || error_exit "Failed to create repos directory"

# Set ownership to runner user (works on standard K8s, may fail on SELinux/SCC)
chown -R 1001:0 "${CLAUDE_DATA_PATH}" /workspace/artifacts /workspace/file-uploads /workspace/repos 2>/dev/null || true

# Set permissions for .claude (best-effort; may be restricted by SCC)
# If the SCC assigns an fsGroup, the directory should already be writable.
chmod -R 777 "${CLAUDE_DATA_PATH}" 2>/dev/null || echo "Warning: failed to chmod ${CLAUDE_DATA_PATH} (continuing)"

# Other directories - standard permissions since chown sets ownership to runner user
chmod 755 /workspace/artifacts 2>/dev/null || true
# SECURITY: 777 required for /workspace/file-uploads because:
# - Init container runs as root but content sidecar runs as user 1001
# - Content sidecar needs write access to store user-uploaded files
# - Directory contains user-uploaded files (no secrets), so world-writable is acceptable
chmod 777 /workspace/file-uploads 2>/dev/null || true
# SECURITY: 777 required for /workspace/repos because:
# - Init container runs as root but runner container runs as user 1001
# - Group-based permissions (775) don't work as containers may not share groups
# - EmptyDir doesn't support fsGroup propagation in all environments
# - Directory contains cloned git repos (no secrets), so world-writable is acceptable
chmod 777 /workspace/repos 2>/dev/null || true

# Check if S3 is configured
if [ -z "${S3_ENDPOINT}" ] || [ -z "${S3_BUCKET}" ] || [ -z "${AWS_ACCESS_KEY_ID}" ] || [ -z "${AWS_SECRET_ACCESS_KEY}" ]; then
    echo "S3 not configured - using ephemeral storage only (no state persistence)"
    echo "========================================="
    exit 0
fi

# Setup rclone
echo "Setting up rclone..."
setup_rclone

S3_PATH="s3:${S3_BUCKET}/${NAMESPACE}/${SESSION_NAME}"

# Test S3 connection
echo "Testing S3 connection..."
if ! rclone --config /tmp/.config/rclone/rclone.conf lsd "s3:${S3_BUCKET}/" --max-depth 1 2>&1; then
    error_exit "Failed to connect to S3 at ${S3_ENDPOINT}. Check endpoint and credentials."
fi
echo "S3 connection successful"

# Check if session state exists in S3
echo "Checking for existing session state in S3..."
if rclone --config /tmp/.config/rclone/rclone.conf lsf "${S3_PATH}/" 2>/dev/null | grep -q .; then
    echo "Found existing session state, downloading from S3..."
    
    # Download .claude data to /app/.claude (SubPath mount matches runner container)
    if rclone --config /tmp/.config/rclone/rclone.conf lsf "${S3_PATH}/.claude/" 2>/dev/null | grep -q .; then
        echo "  Downloading .claude/..."
        rclone --config /tmp/.config/rclone/rclone.conf copy "${S3_PATH}/.claude/" "${CLAUDE_DATA_PATH}/" \
            --copy-links \
            --transfers 8 \
            --fast-list \
            --progress 2>&1 || echo "  Warning: failed to download .claude"
    else
        echo "  No data for .claude/"
    fi
    
    # Download other sync paths to /workspace
    for path in "${SYNC_PATHS[@]}"; do
        if rclone --config /tmp/.config/rclone/rclone.conf lsf "${S3_PATH}/${path}/" 2>/dev/null | grep -q .; then
            echo "  Downloading ${path}/..."
            rclone --config /tmp/.config/rclone/rclone.conf copy "${S3_PATH}/${path}/" "/workspace/${path}/" \
                --copy-links \
                --transfers 8 \
                --fast-list \
                --progress 2>&1 || echo "  Warning: failed to download ${path}"
        else
            echo "  No data for ${path}/"
        fi
    done
    
    echo "State hydration complete!"
else
    echo "No existing state found, starting fresh session"
fi

# Set ownership and permissions on subdirectories after S3 download
echo "Setting ownership and permissions on subdirectories..."
# Try chown first (works on standard K8s), fall back to 777 if blocked by SELinux/SCC
chown -R 1001:0 "${CLAUDE_DATA_PATH}" /workspace/artifacts /workspace/file-uploads /workspace/repos 2>/dev/null || true
# .claude needs 777 for SDK internals
chmod -R 777 "${CLAUDE_DATA_PATH}" 2>/dev/null || true
# repos also needs write access for runtime repo additions (clone_repo_at_runtime)
# See security rationale above for why 777 is used
chmod -R 755 /workspace/artifacts 2>/dev/null || true
chmod -R 777 /workspace/file-uploads 2>/dev/null || true
chmod -R 777 /workspace/repos 2>/dev/null || true

# ========================================
# Clone repositories and workflows
# ========================================
echo "========================================="
echo "Setting up repositories and workflows..."
echo "========================================="

# Disable errexit for git clones (failures are non-fatal for private repos without auth)
set +e

# Set HOME for git config (alpine doesn't set it by default)
export HOME=/tmp

# Git identity - now auto-derived from GitHub/GitLab credentials via API
# Set defaults here, backend git operations will override with user's actual identity
git config --global user.name "Ambient Code Bot" || echo "Warning: failed to set git user.name"
git config --global user.email "bot@ambient-code.local" || echo "Warning: failed to set git user.email"

# Mark workspace as safe (in case runner needs it)
git config --global --add safe.directory /workspace 2>/dev/null || true

# Clone repos from REPOS_JSON
if [ -n "$REPOS_JSON" ] && [ "$REPOS_JSON" != "null" ] && [ "$REPOS_JSON" != "" ]; then
    echo "Cloning repositories from spec..."
    # Parse JSON array and clone each repo
    REPO_COUNT=$(echo "$REPOS_JSON" | jq -e 'if type == "array" then length else 0 end' 2>/dev/null || echo "0")
    echo "Found $REPO_COUNT repositories to clone"
    if [ "$REPO_COUNT" -gt 0 ]; then
        i=0
        while [ $i -lt $REPO_COUNT ]; do
            REPO_URL=$(echo "$REPOS_JSON" | jq -r ".[$i].url // empty" 2>/dev/null || echo "")
            REPO_BRANCH=$(echo "$REPOS_JSON" | jq -r ".[$i].branch // empty" 2>/dev/null || echo "")
            
            # Derive repo name from URL
            REPO_NAME=$(basename "$REPO_URL" .git 2>/dev/null || echo "")
            
            if [ -n "$REPO_NAME" ] && [ -n "$REPO_URL" ] && [ "$REPO_URL" != "null" ]; then
                REPO_DIR="/workspace/repos/$REPO_NAME"
                
                # Mark repo directory as safe
                git config --global --add safe.directory "$REPO_DIR" 2>/dev/null || true
                
                # Clone repository. When branch is specified, clone that branch.
                # When empty, clone the repo's default branch (main/master/etc).
                if [ -n "$REPO_BRANCH" ]; then
                    echo "  Cloning $REPO_NAME (branch: $REPO_BRANCH)..."
                    if git clone --branch "$REPO_BRANCH" --single-branch "$REPO_URL" "$REPO_DIR" 2>&1; then
                        echo "  ✓ Cloned $REPO_NAME (branch: $REPO_BRANCH)"
                        # Install pre-commit hooks if the repo has a config
                        if [ -f "$REPO_DIR/.pre-commit-config.yaml" ] && command -v pre-commit &>/dev/null; then
                            (cd "$REPO_DIR" && pre-commit install 2>/dev/null) || true
                        fi
                    else
                        echo "  ⚠ Failed to clone $REPO_NAME (may require authentication)"
                    fi
                else
                    echo "  Cloning $REPO_NAME (default branch)..."
                    if git clone --single-branch "$REPO_URL" "$REPO_DIR" 2>&1; then
                        echo "  ✓ Cloned $REPO_NAME (default branch)"
                        # Install pre-commit hooks if the repo has a config
                        if [ -f "$REPO_DIR/.pre-commit-config.yaml" ] && command -v pre-commit &>/dev/null; then
                            (cd "$REPO_DIR" && pre-commit install 2>/dev/null) || true
                        fi
                    else
                        echo "  ⚠ Failed to clone $REPO_NAME (may require authentication)"
                    fi
                fi
            fi
            i=$((i + 1))
        done
    fi
else
    echo "No repositories configured in spec"
fi

# Clone workflow repository
if [ -n "$ACTIVE_WORKFLOW_GIT_URL" ] && [ "$ACTIVE_WORKFLOW_GIT_URL" != "null" ]; then
    WORKFLOW_BRANCH="${ACTIVE_WORKFLOW_BRANCH:-main}"
    WORKFLOW_PATH="${ACTIVE_WORKFLOW_PATH:-}"
    
    echo "Cloning workflow repository..."
    echo "  URL: $ACTIVE_WORKFLOW_GIT_URL"
    echo "  Branch: $WORKFLOW_BRANCH"
    if [ -n "$WORKFLOW_PATH" ]; then
        echo "  Subpath: $WORKFLOW_PATH"
    fi
    
    # Derive workflow name from URL
    WORKFLOW_NAME=$(basename "$ACTIVE_WORKFLOW_GIT_URL" .git)
    WORKFLOW_FINAL="/workspace/workflows/${WORKFLOW_NAME}"
    WORKFLOW_TEMP="/tmp/workflow-clone-$$"
    
    git config --global --add safe.directory "$WORKFLOW_FINAL" 2>/dev/null || true
    
    # Clone to temp location
    if git clone --branch "$WORKFLOW_BRANCH" --single-branch "$ACTIVE_WORKFLOW_GIT_URL" "$WORKFLOW_TEMP" 2>&1; then
        echo "  Clone successful, processing..."
        
        # Extract subpath if specified
        if [ -n "$WORKFLOW_PATH" ]; then
            SUBPATH_FULL="$WORKFLOW_TEMP/$WORKFLOW_PATH"
            echo "  Checking for subpath: $SUBPATH_FULL"
            ls -la "$SUBPATH_FULL" 2>&1 || echo "  Subpath does not exist"
            
            if [ -d "$SUBPATH_FULL" ]; then
                echo "  Extracting subpath: $WORKFLOW_PATH"
                mkdir -p "$(dirname "$WORKFLOW_FINAL")"
                cp -r "$SUBPATH_FULL" "$WORKFLOW_FINAL"
                rm -rf "$WORKFLOW_TEMP"
                echo "  ✓ Workflow extracted from subpath to /workspace/workflows/${WORKFLOW_NAME}"
            else
                echo "  ⚠ Subpath '$WORKFLOW_PATH' not found in cloned repo"
                echo "  Available paths in repo:"
                find "$WORKFLOW_TEMP" -maxdepth 3 -type d | head -10
                echo "  Using entire repo instead"
                mv "$WORKFLOW_TEMP" "$WORKFLOW_FINAL"
                echo "  ✓ Workflow ready at /workspace/workflows/${WORKFLOW_NAME}"
            fi
        else
            # No subpath - use entire repo
            mv "$WORKFLOW_TEMP" "$WORKFLOW_FINAL"
            echo "  ✓ Workflow ready at /workspace/workflows/${WORKFLOW_NAME}"
        fi
    else
        echo "  ⚠ Failed to clone workflow"
        rm -rf "$WORKFLOW_TEMP" 2>/dev/null || true
    fi
fi

# ========================================
# Restore git repo state from S3 backup
# ========================================
echo "========================================="
echo "Checking for git repo state backup..."
echo "========================================="

S3_REPO_STATE="${S3_PATH}/repo-state/"

if rclone --config /tmp/.config/rclone/rclone.conf lsf "${S3_REPO_STATE}" 2>/dev/null | grep -q .; then
    echo "Found git repo state backup, restoring..."

    REPO_STATE_DIR="/tmp/repo-state"
    rm -rf "${REPO_STATE_DIR}"
    mkdir -p "${REPO_STATE_DIR}"

    # Download repo state from S3
    rclone --config /tmp/.config/rclone/rclone.conf copy "${S3_REPO_STATE}" "${REPO_STATE_DIR}/" \
        --transfers 8 \
        --fast-list \
        --progress 2>&1 || echo "  Warning: failed to download repo state"

    for repo_state_dir in "${REPO_STATE_DIR}"/*/; do
        [ -d "${repo_state_dir}" ] || continue

        REPO_NAME=$(basename "${repo_state_dir}")
        REPO_DIR="/workspace/repos/${REPO_NAME}"
        METADATA_FILE="${repo_state_dir}/metadata.json"

        echo "  Restoring git state for ${REPO_NAME}..."

        if [ ! -f "${METADATA_FILE}" ]; then
            echo "  WARNING: No metadata.json for ${REPO_NAME}, skipping"
            continue
        fi

        # Read metadata
        REMOTE_URL=$(jq -r '.remoteUrl // empty' "${METADATA_FILE}" 2>/dev/null || echo "")
        SAVED_BRANCH=$(jq -r '.currentBranch // "main"' "${METADATA_FILE}" 2>/dev/null || echo "main")
        SAVED_HEAD=$(jq -r '.headSha // empty' "${METADATA_FILE}" 2>/dev/null || echo "")

        # If repo was not already cloned (e.g., runtime-added repo), clone it
        if [ ! -d "${REPO_DIR}" ] && [ -n "${REMOTE_URL}" ]; then
            # Redact credentials from URL for logging
            SAFE_URL=$(echo "${REMOTE_URL}" | sed 's|://[^@]*@|://|')
            echo "  Cloning missing repo ${REPO_NAME} from ${SAFE_URL}..."
            git config --global --add safe.directory "${REPO_DIR}" 2>/dev/null || true
            git clone "${REMOTE_URL}" "${REPO_DIR}" 2>&1 || {
                echo "  WARNING: Failed to clone ${REPO_NAME}, skipping restore"
                continue
            }
        fi

        if [ ! -d "${REPO_DIR}/.git" ] && [ ! -f "${REPO_DIR}/.git" ]; then
            echo "  WARNING: ${REPO_DIR} is not a git repo, skipping restore"
            continue
        fi

        git config --global --add safe.directory "${REPO_DIR}" 2>/dev/null || true

        # Import local branches from bundle using fetch (creates local branch refs)
        if [ -f "${repo_state_dir}/repo.bundle" ]; then
            echo "  Fetching refs from bundle for ${REPO_NAME}..."
            # Detach HEAD so fetch can update all branches (including the checked-out one)
            git -C "${REPO_DIR}" checkout --detach 2>/dev/null || true
            git -C "${REPO_DIR}" fetch "${repo_state_dir}/repo.bundle" "+refs/heads/*:refs/heads/*" 2>&1 || {
                echo "  WARNING: Failed to fetch refs from bundle for ${REPO_NAME}"
            }
        fi

        # Fetch all remotes to ensure refs are up to date
        git -C "${REPO_DIR}" fetch --all 2>/dev/null || true

        # Checkout the saved branch
        if [ "${SAVED_BRANCH}" != "unknown" ] && [ -n "${SAVED_BRANCH}" ]; then
            echo "  Checking out branch: ${SAVED_BRANCH}"
            git -C "${REPO_DIR}" checkout "${SAVED_BRANCH}" 2>&1 || {
                echo "  WARNING: Failed to checkout ${SAVED_BRANCH}, staying on current branch"
            }
        fi

        # Apply uncommitted changes (best-effort)
        if [ -f "${repo_state_dir}/uncommitted.patch" ] && [ -s "${repo_state_dir}/uncommitted.patch" ]; then
            echo "  Applying uncommitted changes..."
            git -C "${REPO_DIR}" apply --allow-empty "${repo_state_dir}/uncommitted.patch" 2>&1 || {
                echo "  WARNING: Failed to apply uncommitted changes for ${REPO_NAME} (conflicts likely)"
            }
        fi

        # Apply staged changes (best-effort)
        if [ -f "${repo_state_dir}/staged.patch" ] && [ -s "${repo_state_dir}/staged.patch" ]; then
            echo "  Applying staged changes..."
            git -C "${REPO_DIR}" apply --cached --allow-empty "${repo_state_dir}/staged.patch" 2>&1 || {
                echo "  WARNING: Failed to apply staged changes for ${REPO_NAME} (conflicts likely)"
            }
        fi

        # Verify HEAD SHA matches expectation
        CURRENT_HEAD=$(git -C "${REPO_DIR}" rev-parse HEAD 2>/dev/null || echo "")
        if [ -n "${SAVED_HEAD}" ] && [ -n "${CURRENT_HEAD}" ] && [ "${SAVED_HEAD}" != "${CURRENT_HEAD}" ]; then
            echo "  WARNING: HEAD diverged for ${REPO_NAME}: expected ${SAVED_HEAD:0:8}, got ${CURRENT_HEAD:0:8}"
        fi

        echo "  Restored ${REPO_NAME} (branch: ${SAVED_BRANCH})"
    done

    # Clean up
    rm -rf "${REPO_STATE_DIR}"
    echo "Git repo state restore complete"
else
    echo "No git repo state backup found"
fi

# Set permissions on repos after restore (repos may have been cloned or restored)
chown -R 1001:0 /workspace/repos 2>/dev/null || true
chmod -R 777 /workspace/repos 2>/dev/null || true

echo "========================================="
echo "Workspace initialized successfully"
echo "========================================="
exit 0

