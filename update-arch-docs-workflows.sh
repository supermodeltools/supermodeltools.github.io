#!/usr/bin/env bash
# update-arch-docs-workflows.sh
#
# Updates arch-docs.yml in all repos listed in repos.yaml to push output
# to the central GraphTechnologyDevelopers/graphtechnologydevelopers.github.io
# site instead of deploying to individual GitHub Pages.
#
# Prerequisites:
#   - gh CLI authenticated with supermodeltools org access
#   - BOT_TOKEN secret must be set on each repo (PAT with write access to
#     GraphTechnologyDevelopers/graphtechnologydevelopers.github.io)
#
# Usage:
#   BOT_TOKEN=ghp_... ./update-arch-docs-workflows.sh

set -euo pipefail

ORG="supermodeltools"

WORKFLOW_CONTENT='name: Architecture Docs

on:
  push:
    branches: [main, master]
  workflow_dispatch:

permissions:
  contents: read

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: supermodeltools/arch-docs@main
        id: docs
        with:
          supermodel-api-key: ${{ secrets.SUPERMODEL_API_KEY }}
          base-url: https://repos.supermodeltools.com

      - name: Deploy to central site
        env:
          BOT_TOKEN: ${{ secrets.BOT_TOKEN }}
          REPO_NAME: ${{ github.event.repository.name }}
        run: |
          git config --global user.name "supermodel-bot"
          git config --global user.email "bot@supermodeltools.com"
          git clone https://x-access-token:${BOT_TOKEN}@github.com/GraphTechnologyDevelopers/graphtechnologydevelopers.github.io.git central-site
          rm -rf central-site/site/${REPO_NAME}
          mkdir -p central-site/site/${REPO_NAME}
          cp -r arch-docs-output/. central-site/site/${REPO_NAME}/
          cd central-site
          git add site/${REPO_NAME}/
          git diff --staged --quiet && echo "No changes" && exit 0
          git commit -m "Deploy arch-docs for ${REPO_NAME}"
          for i in 1 2 3 4 5; do
            git push && break
            echo "Push failed, retrying in ${i}0s..."
            sleep $((i * 10))
            git pull --rebase origin main
          done'

# Read repo names from repos.yaml
REPOS=$(grep "^      - name:" repos.yaml | awk '{print $3}')

for REPO_NAME in $REPOS; do
  FORK="${ORG}/${REPO_NAME}"
  echo "=== Updating ${FORK} ==="

  # Set BOT_TOKEN secret if provided
  if [ -n "${BOT_TOKEN:-}" ]; then
    echo "  Setting BOT_TOKEN secret..."
    gh secret set BOT_TOKEN --repo "${FORK}" --body "${BOT_TOKEN}"
  else
    echo "  Warning: BOT_TOKEN not set in environment, skipping secret update"
  fi

  ENCODED=$(echo -n "${WORKFLOW_CONTENT}" | base64)
  DEFAULT_BRANCH=$(gh api "repos/${FORK}" --jq '.default_branch' 2>/dev/null || echo "main")
  EXISTING_SHA=$(gh api "repos/${FORK}/contents/.github/workflows/arch-docs.yml" --jq '.sha' 2>/dev/null || echo "")

  if [ -n "${EXISTING_SHA}" ]; then
    gh api --method PUT "repos/${FORK}/contents/.github/workflows/arch-docs.yml" \
      -f message="Update arch-docs workflow to deploy to central site" \
      -f content="${ENCODED}" \
      -f branch="${DEFAULT_BRANCH}" \
      -f sha="${EXISTING_SHA}" \
      --silent
    echo "  Updated arch-docs.yml"
  else
    echo "  No arch-docs.yml found in ${FORK}, skipping"
  fi

  echo ""
done

echo "=== Done! Run workflows manually to trigger initial deploys: ==="
for REPO_NAME in $REPOS; do
  echo "  gh workflow run arch-docs.yml --repo ${ORG}/${REPO_NAME}"
done
