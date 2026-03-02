/**
 * Cloudflare Worker — repo request proxy + loading page
 *
 * POST /api/request  — creates GitHub issue, returns docs URL
 * GET  /generating/*  — serves skeleton loading page
 *
 * Environment secrets (set via wrangler secret put):
 *   GITHUB_TOKEN — fine-grained PAT with issues:write on supermodeltools.github.io
 */

const CORS_HEADERS = {
  'Access-Control-Allow-Origin': 'https://repos.supermodeltools.com',
  'Access-Control-Allow-Methods': 'POST, OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type',
};

const REPO_RE = /github\.com\/([a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+)/;

export default {
  async fetch(request, env) {
    const url = new URL(request.url);

    // CORS preflight
    if (request.method === 'OPTIONS') {
      return new Response(null, { status: 204, headers: CORS_HEADERS });
    }

    // POST /api/request — create issue
    if (url.pathname === '/api/request' && request.method === 'POST') {
      return handleRequest(request, env);
    }

    // GET /generating/{name} — serve loading page
    const genMatch = url.pathname.match(/^\/generating\/([a-zA-Z0-9._-]+)/);
    if (genMatch && request.method === 'GET') {
      return serveSkeleton(genMatch[1]);
    }

    return new Response('Not found', { status: 404 });
  },
};

async function handleRequest(request, env) {
  let body;
  try {
    body = await request.json();
  } catch {
    return jsonResponse({ error: 'Invalid JSON' }, 400);
  }

  const rawUrl = (body.url || '').trim().replace(/\/+$/, '').replace(/\.git$/, '');
  const match = rawUrl.match(REPO_RE);
  if (!match) {
    return jsonResponse({ error: 'Invalid GitHub repository URL' }, 400);
  }

  const upstream = match[1];
  const name = upstream.split('/')[1];

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
    console.error('GitHub API error:', ghResponse.status, await ghResponse.text());
    return jsonResponse({ error: 'Failed to submit. Please try again.' }, 502);
  }

  return jsonResponse({
    success: true,
    name,
    upstream,
    generating_url: `/generating/${name}`,
    docs_url: `https://repos.supermodeltools.com/${name}/`,
  });
}

function serveSkeleton(name) {
  const html = SKELETON_HTML.replaceAll('{{NAME}}', escapeHtml(name));
  return new Response(html, {
    status: 200,
    headers: { 'Content-Type': 'text/html; charset=utf-8', 'Cache-Control': 'no-store' },
  });
}

function jsonResponse(data, status = 200) {
  return new Response(JSON.stringify(data), {
    status,
    headers: { 'Content-Type': 'application/json', ...CORS_HEADERS },
  });
}

function escapeHtml(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

// ---------------------------------------------------------------------------
// Skeleton page — mirrors the real arch-docs layout with shimmer placeholders
// ---------------------------------------------------------------------------
const SKELETON_HTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{NAME}} — Architecture Documentation</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500&display=swap" rel="stylesheet">
  <style>
:root {
  --bg: #0f1117;
  --bg-card: #1a1d27;
  --bg-hover: #22263a;
  --border: #2a2e3e;
  --text: #e4e4e7;
  --text-muted: #9ca3af;
  --accent: #6366f1;
  --accent-light: #818cf8;
  --font: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif;
  --mono: 'JetBrains Mono', 'Fira Code', monospace;
  --max-w: 1200px;
  --radius: 8px;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
html { overflow-x: hidden; }
body {
  font-family: var(--font);
  background: var(--bg);
  color: var(--text);
  line-height: 1.6;
  -webkit-font-smoothing: antialiased;
  overflow-x: hidden;
}
a { color: var(--accent-light); text-decoration: none; }
a:hover { text-decoration: underline; }
.container { max-width: var(--max-w); margin: 0 auto; padding: 0 24px; }

/* Header — matches real arch-docs */
.site-header {
  border-bottom: 1px solid var(--border);
  padding: 16px 0;
  position: sticky;
  top: 0;
  background: var(--bg);
  z-index: 100;
}
.site-header .container {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
}
.site-brand {
  font-size: 18px; font-weight: 700; color: var(--text);
  display: flex; align-items: center; gap: 8px; white-space: nowrap;
}
.site-brand svg { width: 24px; height: 24px; }
.site-nav { display: flex; gap: 16px; align-items: center; }
.site-nav a, .site-nav span { color: var(--text-muted); font-size: 14px; font-weight: 500; white-space: nowrap; }
.nav-all-repos { color: var(--accent-light) !important; padding-right: 12px; margin-right: 4px; border-right: 1px solid var(--border); }

/* Hero — matches real layout */
.hero { padding: 48px 0 40px; text-align: center; }
.hero h1 { font-size: 28px; font-weight: 700; margin-bottom: 12px; }
.hero-sub {
  color: var(--text-muted); font-size: 15px;
  max-width: 560px; margin: 0 auto 24px;
}
.hero-actions { display: flex; gap: 8px; justify-content: center; margin-bottom: 16px; }
.hero-btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 7px 14px; font-size: 13px; font-weight: 500;
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: var(--radius); color: var(--text-muted);
}
.hero-stats {
  display: flex; justify-content: center; gap: 28px; flex-wrap: wrap;
}
.hero-stat { text-align: center; }
.hero-stat .num { font-size: 24px; font-weight: 700; color: var(--accent-light); }
.hero-stat .label { font-size: 12px; color: var(--text-muted); }

/* Shimmer animation */
@keyframes shimmer {
  0% { background-position: -400px 0; }
  100% { background-position: 400px 0; }
}
.shim {
  background: linear-gradient(90deg, var(--bg-card) 25%, var(--bg-hover) 50%, var(--bg-card) 75%);
  background-size: 800px 100%;
  animation: shimmer 1.8s ease-in-out infinite;
  border-radius: 4px;
}
.shim-text { height: 14px; border-radius: 3px; }
.shim-num { width: 48px; height: 28px; margin: 0 auto 4px; border-radius: 4px; }
.shim-label { width: 64px; height: 12px; margin: 0 auto; border-radius: 3px; }

/* Panels — match real arch-docs chart panels */
.chart-panel {
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: var(--radius); padding: 24px; margin-bottom: 24px;
}
.chart-panel h3 { font-size: 16px; font-weight: 600; margin-bottom: 16px; }
.shim-chart { height: 280px; border-radius: var(--radius); }

/* Taxonomy grid skeleton */
.section { margin-bottom: 40px; }
.section-title { font-size: 20px; font-weight: 700; margin-bottom: 12px; }
.tax-grid {
  display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 8px;
}
.tax-entry-skel {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px;
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: 6px;
}
.shim-entry-name { width: 60%; height: 14px; }
.shim-entry-count { width: 28px; height: 14px; }

/* Generation status overlay */
.gen-status {
  position: fixed; bottom: 24px; left: 50%; transform: translateX(-50%);
  background: var(--bg-card); border: 1px solid var(--border);
  border-radius: 12px; padding: 14px 24px;
  display: flex; align-items: center; gap: 14px;
  font-size: 14px; color: var(--text);
  box-shadow: 0 8px 32px rgba(0,0,0,0.4);
  z-index: 200;
  max-width: 90vw;
}
.gen-spinner {
  width: 18px; height: 18px; flex-shrink: 0;
  border: 2px solid var(--border);
  border-top-color: var(--accent-light);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.gen-step { color: var(--text-muted); }
.gen-step strong { color: var(--text); }

/* Footer */
.site-footer {
  border-top: 1px solid var(--border);
  padding: 32px 0; margin-top: 48px;
  color: var(--text-muted); font-size: 13px; text-align: center;
}

@media (max-width: 768px) {
  .container { padding: 0 16px; }
  .hero { padding: 32px 0 24px; }
  .hero h1 { font-size: 22px; }
  .hero-stats { gap: 16px; }
  .tax-grid { grid-template-columns: 1fr; }
  .gen-status { bottom: 12px; padding: 10px 16px; font-size: 13px; }
}
  </style>
</head>
<body>

  <!-- Header — real links, real nav -->
  <header class="site-header">
    <div class="container">
      <a href="/" class="site-brand">
        <svg viewBox="0 0 90 78" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M90 61.1124C75.9375 73.4694 59.8419 78 44.7554 78C29.669 78 11.8614 72.6122 0 61.1011V16.9458C11.6168 6 29.891 0 44.9887 0C62.77 0 78.8723 6.97959 89.9887 16.9458V61.1124H90ZM88.1881 38.9553C77.7923 22.8824 59.8983 15.7959 44.7554 15.7959C29.6126 15.7959 13.4515 21.9008 1.556 38.9444C12.5382 54.69 26.9 62.5085 44.7554 62.0944C67.6297 61.5639 77.6495 51.9184 88.1881 38.9553ZM44.7554 16.3475C32.4756 16.3475 22.3888 26.6879 22.2554 38.9388C34.3765 38.9162 44.7554 29.1429 44.7554 16.3475C44.7554 29.1429 55.1344 38.9162 67.2554 38.9388C67.1202 26.5216 57.1141 16.3475 44.7554 16.3475ZM44.7554 61.5639C44.7554 48.4898 34.3765 38.9613 22.2554 38.9388C22.3888 51.1897 32.4756 61.5639 44.7554 61.5639C57.0352 61.5639 67.122 51.1897 67.2554 38.9388C55.1344 38.9613 44.7554 48.4898 44.7554 61.5639Z" fill="currentColor"/>
        </svg>
        {{NAME}}
      </a>
      <nav class="site-nav">
        <a href="https://repos.supermodeltools.com/" class="nav-all-repos">&larr; All Repos</a>
        <span>By Type</span>
        <span>Domains</span>
        <span>Languages</span>
        <span>Tags</span>
      </nav>
    </div>
  </header>

  <main>
    <div class="container">

      <!-- Hero skeleton -->
      <div class="hero">
        <h1>{{NAME}}</h1>
        <div class="hero-actions">
          <span class="hero-btn">View on GitHub</span>
          <span class="hero-btn">Star</span>
          <span class="hero-btn">Fork</span>
        </div>
        <p class="hero-sub">Architecture documentation generated from code analysis. Explore every file, function, class, and domain.</p>
        <div class="hero-stats">
          <div class="hero-stat"><div class="shim shim-num"></div><div class="label">Total Entities</div></div>
          <div class="hero-stat"><div class="shim shim-num"></div><div class="label">Node Types</div></div>
          <div class="hero-stat"><div class="shim shim-num"></div><div class="label">Languages</div></div>
          <div class="hero-stat"><div class="shim shim-num"></div><div class="label">Domains</div></div>
          <div class="hero-stat"><div class="shim shim-num"></div><div class="label">Subdomains</div></div>
          <div class="hero-stat"><div class="shim shim-num"></div><div class="label">Top Directories</div></div>
        </div>
      </div>

      <!-- Architecture overview skeleton -->
      <div class="chart-panel">
        <h3>Architecture Overview</h3>
        <div class="shim shim-chart"></div>
      </div>

      <!-- Codebase composition skeleton -->
      <div class="chart-panel">
        <h3>Codebase Composition</h3>
        <div class="shim shim-chart" style="height: 200px;"></div>
      </div>

      <!-- Taxonomy grids skeleton -->
      <div class="section">
        <h2 class="section-title">Node Types</h2>
        <div class="tax-grid">
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
        </div>
      </div>

      <div class="section">
        <h2 class="section-title">Domains</h2>
        <div class="tax-grid">
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
        </div>
      </div>

      <div class="section">
        <h2 class="section-title">Languages</h2>
        <div class="tax-grid">
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
          <div class="tax-entry-skel"><div class="shim shim-entry-name"></div><div class="shim shim-entry-count"></div></div>
        </div>
      </div>
    </div>
  </main>

  <footer class="site-footer">
    <div class="container">
      <p>Generated with <a href="https://github.com/supermodeltools/arch-docs">arch-docs</a> by <a href="https://supermodeltools.com">supermodeltools</a></p>
    </div>
  </footer>

  <!-- Floating status bar -->
  <div class="gen-status" id="gen-status">
    <div class="gen-spinner"></div>
    <div class="gen-step" id="gen-step"><strong>Generating docs</strong> &mdash; analyzing codebase&hellip;</div>
  </div>

  <script>
  (function() {
    var name = '{{NAME}}';
    var docsUrl = 'https://repos.supermodeltools.com/' + name + '/';
    var statusEl = document.getElementById('gen-step');

    // Cycle through status messages
    var messages = [
      { text: '<strong>Generating docs</strong> &mdash; forking repository&hellip;', at: 0 },
      { text: '<strong>Generating docs</strong> &mdash; analyzing codebase&hellip;', at: 8000 },
      { text: '<strong>Generating docs</strong> &mdash; building code graphs&hellip;', at: 35000 },
      { text: '<strong>Generating docs</strong> &mdash; mapping architecture&hellip;', at: 60000 },
      { text: '<strong>Generating docs</strong> &mdash; deploying site&hellip;', at: 90000 },
      { text: '<strong>Almost there</strong> &mdash; finalizing&hellip;', at: 120000 },
    ];

    messages.forEach(function(m) {
      setTimeout(function() { statusEl.innerHTML = m.text; }, m.at);
    });

    // Poll the real docs URL
    var pollInterval = 5000;
    var maxPolls = 120;
    var pollCount = 0;

    function poll() {
      pollCount++;
      if (pollCount > maxPolls) {
        statusEl.innerHTML = '<strong>Still working</strong> &mdash; this repo may be large. <a href="' + docsUrl + '">Check manually &rarr;</a>';
        return;
      }

      fetch(docsUrl, { cache: 'no-store', redirect: 'follow' })
        .then(function(resp) {
          if (resp.ok) {
            return resp.text().then(function(html) {
              // Verify it's real arch-docs content, not another loading/404 page
              if (html.indexOf('arch-docs') !== -1 && html.indexOf('gen-status') === -1) {
                statusEl.innerHTML = '<strong>Ready!</strong> Loading docs&hellip;';
                setTimeout(function() { window.location.href = docsUrl; }, 600);
                return;
              }
              setTimeout(poll, pollInterval);
            });
          }
          setTimeout(poll, pollInterval);
        })
        .catch(function() {
          setTimeout(poll, pollInterval);
        });
    }

    setTimeout(poll, pollInterval);
  })();
  </script>
</body>
</html>`;
