#!/usr/bin/env bash
# setup-community-repos.sh
#
# Fork community repos into supermodeltools org, set up arch-docs workflow,
# enable GitHub Pages, and trigger the first build.
#
# Prerequisites:
#   - gh CLI authenticated with supermodeltools org admin access
#   - SUPERMODEL_API_KEY environment variable set
#
# Usage:
#   SUPERMODEL_API_KEY=sk-... ./setup-community-repos.sh

set -euo pipefail

if [ -z "${SUPERMODEL_API_KEY:-}" ]; then
  echo "Error: SUPERMODEL_API_KEY environment variable is required"
  exit 1
fi

ORG="supermodeltools"

REPOS=(
  "facebook/react"
  "vercel/next.js"
  "vuejs/vue"
  "sveltejs/svelte"
  "expressjs/express"
  "fastify/fastify"
  "pallets/flask"
  "django/django"
  "golang/go"
  "rust-lang/rust"
  "denoland/deno"
  "oven-sh/bun"
  "langchain-ai/langchain"
  "huggingface/transformers"
  "anthropics/anthropic-sdk-python"
  "supabase/supabase"
  "drizzle-team/drizzle-orm"
  "tailwindlabs/tailwindcss"
  "shadcn-ui/ui"
  "astro-build/astro"
)

WORKFLOW_CONTENT='name: Architecture Docs

on:
  push:
    branches: [main, master]
  workflow_dispatch:

permissions:
  contents: write
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
    sleep 2
  fi

  # 2. Set the SUPERMODEL_API_KEY secret
  echo "  Setting SUPERMODEL_API_KEY secret..."
  gh secret set SUPERMODEL_API_KEY --repo "${FORK}" --body "${SUPERMODEL_API_KEY}"

  # 3. Detect default branch
  DEFAULT_BRANCH=$(gh api "repos/${FORK}" --jq '.default_branch')
  echo "  Default branch: ${DEFAULT_BRANCH}"

  # 4. Push the arch-docs workflow file
  echo "  Creating arch-docs workflow..."
  ENCODED=$(echo -n "${WORKFLOW_CONTENT}" | base64)

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

  # 5. Enable GitHub Pages with Actions as source
  echo "  Enabling GitHub Pages..."
  gh api --method POST "repos/${FORK}/pages" \
    -f build_type="workflow" \
    --silent 2>/dev/null || \
  gh api --method PUT "repos/${FORK}/pages" \
    -f build_type="workflow" \
    --silent 2>/dev/null || \
  echo "  (Pages may already be configured)"

  # 6. Trigger the workflow
  echo "  Triggering arch-docs workflow..."
  gh workflow run arch-docs.yml --repo "${FORK}" --ref "${DEFAULT_BRANCH}" 2>/dev/null || \
    echo "  (Workflow trigger may need a moment, try manually if needed)"

  echo "  Done: ${FORK}"
  echo ""
done

echo "=== All community repos set up! ==="
echo "Docs will deploy at repos.supermodeltools.com/{repo}/ as workflows complete."
