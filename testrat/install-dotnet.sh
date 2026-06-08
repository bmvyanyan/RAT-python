#!/usr/bin/env bash
set -euo pipefail

DOTNET_VERSION="6.0.400"
INSTALL_DIR="${DOTNET_INSTALL_DIR:-${HOME}/.dotnet}"
SCRIPT_PATH="/tmp/dotnet-install.sh"

mkdir -p "$INSTALL_DIR"

echo "Downloading .NET installer..."
if command -v curl >/dev/null 2>&1; then
  curl -fsSL https://dot.net/v1/dotnet-install.sh -o "$SCRIPT_PATH"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$SCRIPT_PATH" https://dot.net/v1/dotnet-install.sh
else
  echo "Error: curl or wget is required to download the installer." >&2
  exit 1
fi

chmod +x "$SCRIPT_PATH"

bash "$SCRIPT_PATH" --version "$DOTNET_VERSION" --install-dir "$INSTALL_DIR"

echo "Installed .NET SDK $DOTNET_VERSION to $INSTALL_DIR"
echo "To use it in this shell, run:"
echo "  export DOTNET_ROOT=\"$INSTALL_DIR\""
echo "  export PATH=\"$INSTALL_DIR:$PATH\""
echo "Then run: dotnet --version"
