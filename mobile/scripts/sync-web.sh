#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# Builds the Capacitor `www/` payload from the existing web app.
#
# This is the ONLY link between the mobile wrapper and the rest of the repo, and
# it is READ-ONLY toward frontend/. It copies frontend/ into mobile/www/, swaps
# in the baked mobile config, and appends the native bridge script — none of
# which touches the original files under frontend/.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

MOBILE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_DIR="$(cd "$MOBILE_DIR/.." && pwd)"
FRONTEND_DIR="$REPO_DIR/frontend"
WWW_DIR="$MOBILE_DIR/www"

if [ ! -d "$FRONTEND_DIR" ]; then
  echo "✗ frontend/ not found at $FRONTEND_DIR" >&2
  exit 1
fi

echo "==> Rebuilding www/ from frontend/"
rm -rf "$WWW_DIR"
mkdir -p "$WWW_DIR"
cp -R "$FRONTEND_DIR/." "$WWW_DIR/"

# 1. Baked-in config (replaces the S3-served config.js).
if [ ! -f "$MOBILE_DIR/config.mobile.js" ]; then
  echo "✗ config.mobile.js missing — copy the template and fill in prod values." >&2
  exit 1
fi
cp "$MOBILE_DIR/config.mobile.js" "$WWW_DIR/config.js"

# 2. Native bridge (deep-link auth, push notifications). Web build never has it.
cp "$MOBILE_DIR/src/native-bridge.js" "$WWW_DIR/native-bridge.js"

# 3. Inject the bridge <script> right before </body> in the COPIED index.html.
if grep -q "native-bridge.js" "$WWW_DIR/index.html"; then
  echo "    (bridge already referenced)"
else
  sed -i.bak 's#</body>#  <script src="native-bridge.js"></script>\n</body>#' "$WWW_DIR/index.html"
  rm -f "$WWW_DIR/index.html.bak"
fi

# 4. Guard: warn if placeholder config values are still present.
if grep -q "REPLACE-WITH-YOUR" "$WWW_DIR/config.js"; then
  echo "⚠  config.js still has REPLACE-WITH-YOUR placeholders — edit config.mobile.js"
fi

echo "✅ www/ ready ($(find "$WWW_DIR" -type f | wc -l | tr -d ' ') files)"
