#!/usr/bin/env bash
# setup-community-repos.sh
#
# Fork community repos into supermodeltools org, set up arch-docs workflow,
# enable GitHub Pages, and trigger the first build.
#
# Community repos deploy arch-docs to GitHub Pages at supermodeltools.github.io/{repo}/
# The central site proxies those paths via Cloudflare Pages _redirects.
#
# Prerequisites:
#   - gh CLI authenticated with supermodeltools org admin access
#   - SUPERMODEL_API_KEY is available as an org-level secret (no local env needed)
#
# Usage:
#   ./setup-community-repos.sh

set -euo pipefail

ORG="supermodeltools"

# Community repos from repos.yaml (upstream → fork name)
REPOS=(
  "vuejs/vue"
  "oven-sh/bun"
  "tiangolo/fastapi"
  "gin-gonic/gin"
  "spring-projects/spring-boot"
  "pytorch/pytorch"
  "supabase/supabase"
  "tailwindlabs/tailwindcss"
)

# GitHub Pages deployment workflow (no BOT_TOKEN needed)
WORKFLOW_CONTENT='name: Architecture Docs

on:
  push:
    branches: [main, master]
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: true

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deploy.outputs.page_url }}
    steps:
      - uses: actions/checkout@v4

      - uses: supermodeltools/arch-docs@main
        id: docs
        with:
          supermodel-api-key: ${{ secrets.SUPERMODEL_API_KEY }}
          base-url: https://repos.supermodeltools.com

      - uses: actions/configure-pages@v5

      - uses: actions/upload-pages-artifact@v3
        with:
          path: ./arch-docs-output

      - uses: actions/deploy-pages@v4
        id: deploy'

for UPSTREAM in "${REPOS[@]}"; do
  REPO_NAME="${UPSTREAM##*/}"
  FORK="${ORG}/${REPO_NAME}"

  echo "=== Setting up ${UPSTREAM} as ${FORK} ==="

  # 1. Fork the repo (skip if already exists)
  if gh repo view "${FORK}" &>/dev/null; then
    echo "  Fork ${FORK} already exists, skipping fork step"
  else
    echo "  Forking ${UPSTREAM} into ${ORG}..."
    gh repo fork "${UPSTREAM}" --org "${ORG}" --clone=false
    sleep 3
  fi

  # 2. Detect default branch
  DEFAULT_BRANCH=$(gh api "repos/${FORK}" --jq '.default_branch')
  echo "  Default branch: ${DEFAULT_BRANCH}"

  # 3. Push the arch-docs workflow file
  echo "  Creating arch-docs workflow..."
  ENCODED=$(printf '%s' "${WORKFLOW_CONTENT}" | base64 | tr -d '\n')

  # Check if workflow already exists
  EXISTING_SHA=$(gh api "repos/${FORK}/contents/.github/workflows/arch-docs.yml" --jq '.sha' 2>/dev/null || echo "")

  if [ -n "${EXISTING_SHA}" ]; then
    gh api --method PUT "repos/${FORK}/contents/.github/workflows/arch-docs.yml" \
      -f message="Add arch-docs workflow" \
      -f content="${ENCODED}" \
      -f branch="${DEFAULT_BRANCH}" \
      -f sha="${EXISTING_SHA}" \
      --silent
  else
    gh api --method PUT "repos/${FORK}/contents/.github/workflows/arch-docs.yml" \
      -f message="Add arch-docs workflow" \
      -f content="${ENCODED}" \
      -f branch="${DEFAULT_BRANCH}" \
      --silent
  fi

  # 4. Enable GitHub Pages with Actions as source
  echo "  Enabling GitHub Pages..."
  gh api --method POST "repos/${FORK}/pages" \
    -f build_type="workflow" \
    --silent 2>/dev/null || \
  gh api --method PUT "repos/${FORK}/pages" \
    -f build_type="workflow" \
    --silent 2>/dev/null || \
  echo "  (Pages may already be configured)"

  # 5. Trigger the workflow
  echo "  Triggering arch-docs workflow..."
  sleep 2  # brief pause for workflow file to propagate
  gh workflow run arch-docs.yml --repo "${FORK}" --ref "${DEFAULT_BRANCH}" 2>/dev/null || \
    echo "  (Workflow trigger may need a moment — trigger manually if needed)"

  echo "  Done: ${FORK}"
  echo ""
done

echo "=== All community repos set up! ==="
echo "Arch-docs will deploy to supermodeltools.github.io/{repo}/ as workflows complete."
echo "The central site at repos.supermodeltools.com/{repo}/ will proxy there via _redirects."
