#!/bin/bash
# publish-community.sh
# Publishes a pruned snapshot of elko-agit to elko-agit-community (public).
# Sensitive files listed in .community-exclude are stripped.
# Community repo receives a clean squashed commit — no internal history leaks.

set -euo pipefail

PRIVATE_REPO="git@github.com:jsoprych/elko-agit.git"
PUBLIC_REPO="git@github.com:jsoprych/elko-agit-community.git"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
EXCLUDE_FILE="$ROOT_DIR/.community-exclude"
COMMUNITY_AGENTS="$SCRIPT_DIR/community-AGENTS.md"
WORK_DIR="$(mktemp -d)"

echo "==> Publishing elko-agit → elko-agit-community"
echo "    Work dir: $WORK_DIR"

cleanup() { rm -rf "$WORK_DIR"; }
trap cleanup EXIT

# 1. Clone private repo (shallow — we don't want to copy history)
echo "==> Cloning private repo..."
git clone --depth 1 "$PRIVATE_REPO" "$WORK_DIR/source"

# 2. Build rsync exclude args from .community-exclude
RSYNC_EXCLUDES=()
while IFS= read -r line || [[ -n "$line" ]]; do
  [[ -z "$line" || "$line" == \#* ]] && continue
  RSYNC_EXCLUDES+=(--exclude="$line")
done < "$EXCLUDE_FILE"

# 3. Copy to clean community dir (no .git)
echo "==> Pruning sensitive files..."
mkdir "$WORK_DIR/community"
rsync -a --exclude='.git' "${RSYNC_EXCLUDES[@]}" "$WORK_DIR/source/" "$WORK_DIR/community/"

# 4. Drop in the community AGENTS.md stub
cp "$COMMUNITY_AGENTS" "$WORK_DIR/community/AGENTS.md"

# 5. Init orphan branch — zero history
echo "==> Initialising clean git history..."
cd "$WORK_DIR/community"
git init
git checkout --orphan master
git add -A
COMMIT_MSG="chore: community release $(date +%Y-%m-%d)"
git -c user.name="Elko.AI" -c user.email="noreply@elko.ai" \
    commit -m "$COMMIT_MSG"

# 6. Force-push to public repo
echo "==> Pushing to elko-agit-community..."
git remote add origin "$PUBLIC_REPO"
git push -f origin master

echo ""
echo "✓ Published: https://github.com/jsoprych/elko-agit-community"
