#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$REPO_ROOT"

echo "Removing git metadata..."
rm -rf .git
rm -rf testrat/.git

echo "Initializing a fresh git repository..."
git init

git add .
git commit -m "Reinitialize repository after removing upstream git metadata"

git branch -M main

git remote add origin https://github.com/bmvyanyan/RAT-python.git

echo "Pushing to GitHub..."
git push -u origin main --force

echo "Done. Repository metadata removed and new commit pushed."
