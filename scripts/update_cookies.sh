#!/usr/bin/env bash
# Extracts __Secure-1PSID and __Secure-1PSIDTS from Firefox and updates .env
# Opens a Firefox tab to refresh the session if cookies are missing or expired.
#
# Usage:
#   bash scripts/update_cookies.sh                     # uses default Firefox profile session
#   bash scripts/update_cookies.sh --container 1       # uses Firefox Container 1
#   bash scripts/update_cookies.sh --list              # list all available sessions

set -euo pipefail

ENV_FILE="$(dirname "$0")/../.env"
COMPOSE_FILE="$(dirname "$0")/../docker-compose.yml"
TMP_DB="/tmp/firefox_cookies_$$.sqlite"
CONTAINER_ID=""
LIST_ONLY=false
HEALTH_URL="http://localhost:4981/health"
HEALTH_WAIT=15  # seconds to wait for Docker to initialize

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    --container) CONTAINER_ID="$2"; shift 2 ;;
    --list)      LIST_ONLY=true; shift ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

GEMINI_URL="https://gemini.google.com/?authuser=1"

# --- Open a Firefox tab ---
open_firefox() {
  echo "Opening Firefox at $GEMINI_URL ..."
  if command -v xdg-open &>/dev/null; then
    xdg-open "$GEMINI_URL" &
  elif command -v firefox &>/dev/null; then
    firefox --new-tab "$GEMINI_URL" &
  elif snap list firefox &>/dev/null 2>&1; then
    snap run firefox --new-tab "$GEMINI_URL" &
  else
    echo "  Could not find a browser to open. Please visit: $GEMINI_URL"
  fi
}

# --- Copy cookies DB (Firefox may have it locked) ---
copy_db() {
  COOKIES_DB=$(find \
    "$HOME/snap/firefox/common/.mozilla/firefox" \
    "$HOME/.mozilla/firefox" \
    -name "cookies.sqlite" 2>/dev/null | head -1)

  if [ -z "$COOKIES_DB" ]; then
    echo "ERROR: Firefox cookies.sqlite not found." >&2
    exit 1
  fi
  cp "$COOKIES_DB" "$TMP_DB"
}

trap 'rm -f "$TMP_DB"' EXIT

copy_db

# --- List mode ---
if [ "$LIST_ONLY" = true ]; then
  echo "Available Gemini sessions in Firefox:"
  echo ""
  sqlite3 "$TMP_DB" \
    "SELECT
       CASE originAttributes WHEN '' THEN 'default' ELSE originAttributes END as session,
       name,
       substr(value,1,40)||'...',
       datetime(expiry, 'unixepoch') as expires
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

# --- Check cookie expiry before using them ---
NOW=$(date +%s)
EXPIRY=$(sqlite3 "$TMP_DB" \
  "SELECT MIN(expiry) FROM moz_cookies
   WHERE host LIKE '%.google.com'
   AND name IN ('__Secure-1PSID','__Secure-1PSIDTS')
   AND $ORIGIN_FILTER;" 2>/dev/null || echo "0")

if [ -z "$EXPIRY" ] || [ "$EXPIRY" = "" ] || [ "$EXPIRY" -le "$NOW" ]; then
  echo "Cookies are missing or expired. Opening Firefox to refresh session..."
  open_firefox
  echo ""
  read -rp "Press Enter once you have logged in to Gemini in Firefox..."
  echo ""
  # Re-copy the DB after user has refreshed the session
  copy_db
fi

# --- Extract matched cookie pair ---
PSID=$(sqlite3 "$TMP_DB" \
  "SELECT value FROM moz_cookies WHERE host LIKE '%.google.com' AND name='__Secure-1PSID' AND $ORIGIN_FILTER LIMIT 1;")
PSIDTS=$(sqlite3 "$TMP_DB" \
  "SELECT value FROM moz_cookies WHERE host LIKE '%.google.com' AND name='__Secure-1PSIDTS' AND $ORIGIN_FILTER LIMIT 1;")

if [ -z "$PSID" ] || [ -z "$PSIDTS" ]; then
  echo "ERROR: Still could not find Gemini cookies in Firefox." >&2
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
if [ -f "$COMPOSE_FILE" ]; then
  echo ""
  echo "Restarting Docker container..."
  docker compose -f "$COMPOSE_FILE" up -d
  echo "Waiting ${HEALTH_WAIT}s for initialization..."
  sleep "$HEALTH_WAIT"

  # --- Verify health ---
  HEALTH=$(curl -s --max-time 5 "$HEALTH_URL" 2>/dev/null || echo "")
  GEMINI_OK=$(echo "$HEALTH" | python3 -c \
    "import sys,json; d=json.load(sys.stdin); print(d['providers']['gemini']['healthy'])" 2>/dev/null || echo "False")

  if [ "$GEMINI_OK" = "True" ]; then
    echo "Gemini provider is healthy."
  else
    echo "Gemini provider is unhealthy — cookies may still be invalid."
    echo "Opening Firefox to refresh session..."
    open_firefox
    echo ""
    echo "After logging in, run this script again to update the cookies."
  fi
else
  echo "WARNING: docker-compose.yml not found, skipping restart." >&2
fi
