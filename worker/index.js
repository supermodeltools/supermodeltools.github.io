/**
 * Cloudflare Worker — repo request proxy
 *
 * Receives a repo URL from the homepage, creates a GitHub issue
 * (which triggers the auto-add-repo workflow), and returns immediately.
 *
 * Environment secrets (set via wrangler secret put):
 *   GITHUB_TOKEN — fine-grained PAT with issues:write on supermodeltools.github.io
 *
 * Deploy:
 *   cd worker && npx wrangler deploy
 *
 * Route (add in Cloudflare dashboard or wrangler.toml):
 *   repos.supermodeltools.com/api/*  →  this worker
 */

const CORS_HEADERS = {
  'Access-Control-Allow-Origin': 'https://repos.supermodeltools.com',
  'Access-Control-Allow-Methods': 'POST, OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type',
};

const REPO_RE = /github\.com\/([a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+)/;

export default {
  async fetch(request, env) {
    if (request.method === 'OPTIONS') {
      return new Response(null, { status: 204, headers: CORS_HEADERS });
    }

    if (request.method !== 'POST') {
      return jsonResponse({ error: 'Method not allowed' }, 405);
    }

    let body;
    try {
      body = await request.json();
    } catch {
      return jsonResponse({ error: 'Invalid JSON' }, 400);
    }

    const url = (body.url || '').trim().replace(/\/+$/, '').replace(/\.git$/, '');
    const match = url.match(REPO_RE);
    if (!match) {
      return jsonResponse({ error: 'Invalid GitHub repository URL' }, 400);
    }

    const upstream = match[1];
    const name = upstream.split('/')[1];

    // Create the GitHub issue — this triggers auto-add-repo.yml
    const ghResponse = await fetch(
      'https://api.github.com/repos/supermodeltools/supermodeltools.github.io/issues',
      {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${env.GITHUB_TOKEN}`,
          Accept: 'application/vnd.github+json',
          'User-Agent': 'supermodel-request-bot',
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          title: `[Repo Request] ${name}`,
          body: `### Repository URL\n\nhttps://github.com/${upstream}`,
          labels: ['repo-request'],
        }),
      }
    );

    if (!ghResponse.ok) {
      const err = await ghResponse.text();
      console.error('GitHub API error:', ghResponse.status, err);
      return jsonResponse({ error: 'Failed to submit request. Please try again.' }, 502);
    }

    const issue = await ghResponse.json();

    return jsonResponse({
      success: true,
      name,
      upstream,
      docs_url: `https://repos.supermodeltools.com/${name}/`,
      issue_url: issue.html_url,
    });
  },
};

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json', ...CORS_HEADERS },
  });
}
