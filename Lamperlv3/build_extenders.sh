#!/bin/bash
# Quick build script for Lamperlv3 extenders
# Usage: ./build_extenders.sh

set -e

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"

echo "=========================================="
echo "Building Lamperlv3 Extenders"
echo "=========================================="
echo ""

# Check Go is installed
if ! command -v go &> /dev/null; then
    echo "[ERROR] Go is not installed"
    echo "Install Go 1.18+: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
echo "[OK] Go $GO_VERSION found"

# Build Listener
echo ""
echo "[1] Building HTTP Listener..."
cd "$REPO_ROOT/Lamperlv3/lamperl_listener_http"

echo "  - Checking Go dependencies..."
go mod download

echo "  - Compiling listener plugin..."
make clean && make

LISTENER_SIZE=$(du -sh dist/lamperl_http.so | awk '{print $1}')
echo "[OK] Listener built: dist/lamperl_http.so ($LISTENER_SIZE)"

# Build Agent
echo ""
echo "[2] Building Agent Extender..."
cd "$REPO_ROOT/Lamperlv3/lamperl_agent"

echo "  - Checking Go dependencies..."
go mod download

echo "  - Validating Perl agent..."
if command -v perl &> /dev/null; then
    perl -c src_lamperl/lamperl.pl
    echo "[OK] Perl agent syntax OK"
else
    echo "[WARNING] Perl not found, skipping syntax check"
fi

echo "  - Compiling agent plugin..."
make clean && make

AGENT_SIZE=$(du -sh dist/lamperl_agent.so | awk '{print $1}')
echo "[OK] Agent built: dist/lamperl_agent.so ($AGENT_SIZE)"

# Summary
echo ""
echo "=========================================="
echo "[SUCCESS] Build Complete"
echo "=========================================="
echo ""
echo "Output files:"
echo "  - $REPO_ROOT/Lamperlv3/lamperl_listener_http/dist/"
echo "  - $REPO_ROOT/Lamperlv3/lamperl_agent/dist/"
echo ""
echo "Next: Deploy to Adaptix"
echo "  ./Lamperlv3/deploy_to_adaptix.sh /path/to/AdaptixC2"
echo ""
