#!/bin/bash
# Helper script to verify GitHub App setup

set -e

echo "🔍 Checking GitHub App Setup..."
echo ""

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) is not installed"
    echo "   Install from: https://cli.github.com/"
    exit 1
fi
echo "✅ GitHub CLI is installed"

# Check if authenticated
if ! gh auth status &> /dev/null; then
    echo "❌ Not authenticated with GitHub CLI"
    echo "   Run: gh auth login"
    exit 1
fi
echo "✅ Authenticated with GitHub CLI"

# Get repository info
REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner)
echo "📦 Repository: $REPO"
echo ""

# Check for APP_ID variable
echo "Checking for APP_ID variable..."
if gh variable list | grep -q "APP_ID"; then
    APP_ID=$(gh variable get APP_ID)
    echo "✅ APP_ID is set: $APP_ID"
else
    echo "❌ APP_ID variable is not set"
    echo "   Set it with: gh variable set APP_ID --body <your-app-id>"
    exit 1
fi

# Check for APP_PRIVATE_KEY secret
echo "Checking for APP_PRIVATE_KEY secret..."
if gh secret list | grep -q "APP_PRIVATE_KEY"; then
    echo "✅ APP_PRIVATE_KEY secret is set"
else
    echo "❌ APP_PRIVATE_KEY secret is not set"
    echo "   Set it with: gh secret set APP_PRIVATE_KEY < path/to/private-key.pem"
    exit 1
fi

echo ""
echo "🎉 All checks passed! Your GitHub App setup looks good."
echo ""
echo "Next steps:"
echo "1. Make sure your GitHub App is installed on this repository"
echo "2. Trigger the workflow: gh workflow run run-release.yml"
echo "3. Check the workflow status: gh run list --workflow=run-release.yml"
