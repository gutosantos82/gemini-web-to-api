#!/usr/bin/env bash
# Extracts __Secure-1PSID and __Secure-1PSIDTS from Firefox and updates .env
#
# Usage:
#   bash scripts/update_cookies.sh                     # uses default Firefox profile session
#   bash scripts/update_cookies.sh --container 1       # uses Firefox Container 1
#   bash scripts/update_cookies.sh --list              # list all available sessions

set -euo pipefail

ENV_FILE="$(dirname "$0")/../.env"
TMP_DB="/tmp/firefox_cookies_$$.sqlite"
CONTAINER_ID=""
LIST_ONLY=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --container) CONTAINER_ID="$2"; shift 2 ;;
    --list)      LIST_ONLY=true; shift ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# --- Find Firefox cookies.sqlite (prefer Snap, fall back to native) ---
COOKIES_DB=$(find \
  "$HOME/snap/firefox/common/.mozilla/firefox" \
  "$HOME/.mozilla/firefox" \
  -name "cookies.sqlite" 2>/dev/null | head -1)

if [ -z "$COOKIES_DB" ]; then
  echo "ERROR: Firefox cookies.sqlite not found." >&2
  exit 1
fi

# Copy to temp (Firefox may have it locked)
cp "$COOKIES_DB" "$TMP_DB"
trap 'rm -f "$TMP_DB"' EXIT

# --- List mode: show all available sessions ---
if [ "$LIST_ONLY" = true ]; then
  echo "Available Gemini sessions in Firefox:"
  echo ""
  sqlite3 "$TMP_DB" \
    "SELECT
       CASE originAttributes WHEN '' THEN 'default' ELSE originAttributes END as session,
       name,
       substr(value,1,40)||'...'
     FROM moz_cookies
     WHERE host LIKE '%.google.com' AND name IN ('__Secure-1PSID','__Secure-1PSIDTS')
     ORDER BY originAttributes, name;"
  exit 0
fi

# --- Build originAttributes filter ---
if [ -z "$CONTAINER_ID" ]; then
  ORIGIN_FILTER="originAttributes = ''"
else
  ORIGIN_FILTER="originAttributes = '^userContextId=$CONTAINER_ID'"
fi

# --- Extract matched cookie pair ---
PSID=$(sqlite3 "$TMP_DB" \
  "SELECT value FROM moz_cookies WHERE host LIKE '%.google.com' AND name='__Secure-1PSID' AND $ORIGIN_FILTER LIMIT 1;")
PSIDTS=$(sqlite3 "$TMP_DB" \
  "SELECT value FROM moz_cookies WHERE host LIKE '%.google.com' AND name='__Secure-1PSIDTS' AND $ORIGIN_FILTER LIMIT 1;")

if [ -z "$PSID" ] || [ -z "$PSIDTS" ]; then
  echo "ERROR: Could not find Gemini cookies in Firefox." >&2
  echo "Make sure you are logged in to gemini.google.com." >&2
  echo "Run with --list to see available sessions." >&2
  exit 1
fi

# --- Update .env ---
if [ ! -f "$ENV_FILE" ]; then
  echo "ERROR: .env file not found at $ENV_FILE" >&2
  exit 1
fi

sed -i "s|^GEMINI_1PSID=.*|GEMINI_1PSID=$PSID|" "$ENV_FILE"
sed -i "s|^GEMINI_1PSIDTS=.*|GEMINI_1PSIDTS=$PSIDTS|" "$ENV_FILE"

echo "Cookies updated in $ENV_FILE"
echo "  GEMINI_1PSID:   ${PSID:0:20}..."
echo "  GEMINI_1PSIDTS: ${PSIDTS:0:20}..."

# --- Restart Docker container ---
COMPOSE_FILE="$(dirname "$0")/../docker-compose.yml"
if [ -f "$COMPOSE_FILE" ]; then
  echo ""
  echo "Restarting Docker container..."
  docker compose -f "$COMPOSE_FILE" up -d
  echo "Done."
else
  echo "WARNING: docker-compose.yml not found, skipping restart." >&2
fi
