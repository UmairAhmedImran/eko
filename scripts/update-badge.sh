#!/usr/bin/env bash
# =============================================================================
# update-badge.sh — Auto-update the "Good First Issues" badge in README.md
#
# Usage:
#   ./scripts/update-badge.sh              # auto-detect repo from git remote
#   ./scripts/update-badge.sh kavix/eko   # pass repo explicitly
#
# Requirements:
#   - gh CLI (authenticated)   OR   curl (uses GitHub REST API)
#   - sed, git (standard on macOS/Linux)
# =============================================================================

set -euo pipefail

# ── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
RESET='\033[0m'

# ── Helpers ───────────────────────────────────────────────────────────────────
info()    { echo -e "${CYAN}ℹ${RESET}  $*"; }
success() { echo -e "${GREEN}✔${RESET}  $*"; }
warn()    { echo -e "${YELLOW}⚠${RESET}  $*"; }
error()   { echo -e "${RED}✖${RESET}  $*" >&2; exit 1; }

# ── Resolve repo ─────────────────────────────────────────────────────────────
REPO="${1:-}"

if [[ -z "$REPO" ]]; then
  # Auto-detect from git remote
  REMOTE_URL=$(git remote get-url origin 2>/dev/null) || error "Not a git repo and no REPO argument given."
  # Handle both SSH and HTTPS remotes
  REPO=$(echo "$REMOTE_URL" \
    | sed -E 's|git@github.com:||; s|https://github.com/||; s|\.git$||')
  info "Auto-detected repo: ${BOLD}$REPO${RESET}"
fi

README="${2:-README.md}"
[[ -f "$README" ]] || error "README file not found: $README"

# ── Fetch open 'good first issue' count ──────────────────────────────────────
info "Fetching open 'good first issue' count from GitHub..."

if command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
  # Use gh CLI (authenticated — no rate limit concerns)
  COUNT=$(gh issue list \
    --repo "$REPO" \
    --label "good first issue" \
    --state open \
    --limit 500 \
    --json number \
    --jq 'length')
  info "Fetched via ${BOLD}gh CLI${RESET}"
else
  # Fallback: GitHub REST API via curl (unauthenticated — 60 req/hr limit)
  warn "gh CLI not available or not authenticated. Falling back to curl + GitHub API."
  API_URL="https://api.github.com/search/issues?q=repo:${REPO}+label:\"good+first+issue\"+state:open+is:issue&per_page=1"
  RESPONSE=$(curl -sf \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    "$API_URL") || error "Failed to reach GitHub API. Check your internet connection."
  COUNT=$(echo "$RESPONSE" | grep -o '"total_count":[0-9]*' | grep -o '[0-9]*')
fi

[[ "$COUNT" =~ ^[0-9]+$ ]] || error "Unexpected count value: '$COUNT'"
info "Open good first issues: ${BOLD}$COUNT${RESET}"

# ── Read current badge count from README ─────────────────────────────────────
CURRENT=$(grep -oE 'Good%20First%20Issues-[0-9]+%20Open' "$README" \
  | grep -oE '[0-9]+' | head -1 || echo "unknown")

if [[ "$CURRENT" == "$COUNT" ]]; then
  success "Badge already up-to-date (${BOLD}$COUNT Open${RESET}). No changes needed."
  exit 0
fi

info "Badge count: ${YELLOW}$CURRENT${RESET} → ${GREEN}$COUNT${RESET}"

# ── Update the badge in README ────────────────────────────────────────────────
# Matches: Good%20First%20Issues-<N>%20Open-brightgreen
sed -i.bak \
  "s|Good%20First%20Issues-[0-9]*%20Open|Good%20First%20Issues-${COUNT}%20Open|g" \
  "$README"

rm -f "${README}.bak"
success "Updated ${BOLD}$README${RESET} badge → ${GREEN}${COUNT} Open${RESET}"

# ── Optional: commit and push ─────────────────────────────────────────────────
if [[ "${AUTO_COMMIT:-0}" == "1" ]]; then
  info "AUTO_COMMIT=1 — committing and pushing..."
  git add "$README"
  git commit -m "chore: update Good First Issues badge to ${COUNT} open [skip ci]"
  git push
  success "Pushed badge update to remote."
else
  echo ""
  echo -e "  ${YELLOW}Tip:${RESET} To auto-commit + push, run:"
  echo -e "  ${BOLD}AUTO_COMMIT=1 ./scripts/update-badge.sh${RESET}"
fi

echo ""
echo -e "${GREEN}${BOLD}Done!${RESET} Good First Issues badge is now ${BOLD}${COUNT} Open${RESET}."
