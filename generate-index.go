package main

import (
	"fmt"
	"html"
	"net/url"
	"os"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Categories []Category `yaml:"categories"`
}

type Category struct {
	Name  string `yaml:"name"`
	Slug  string `yaml:"slug"`
	Repos []Repo `yaml:"repos"`
}

type Repo struct {
	Name      string `yaml:"name"`
	Upstream  string `yaml:"upstream"`
	Desc      string `yaml:"description"`
	Pill      string `yaml:"pill"`
	PillClass string `yaml:"pill_class"`
}

const baseURL = "https://repos.supermodeltools.com"

func main() {
	data, err := os.ReadFile("repos.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading repos.yaml: %v\n", err)
		os.Exit(1)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing repos.yaml: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll("site", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating site dir: %v\n", err)
		os.Exit(1)
	}

	if err := generateIndex(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating index: %v\n", err)
		os.Exit(1)
	}

	if err := generateSitemap(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating sitemap: %v\n", err)
		os.Exit(1)
	}

	// Copy CNAME and static root files to site directory
	if cname, err := os.ReadFile("CNAME"); err == nil {
		os.WriteFile("site/CNAME", cname, 0644)
	}
	if vf, err := os.ReadFile("google3f45b72e3ef79ea3.html"); err == nil {
		os.WriteFile("site/google3f45b72e3ef79ea3.html", vf, 0644)
	}

	totalRepos := 0
	for _, cat := range cfg.Categories {
		totalRepos += len(cat.Repos)
	}
	fmt.Printf("Generated index.html and sitemap.xml (%d repos)\n", totalRepos)
}

func generateSitemap(cfg Config) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, cat := range cfg.Categories {
		for _, repo := range cat.Repos {
			b.WriteString(fmt.Sprintf("  <sitemap>\n    <loc>%s/%s/sitemap.xml</loc>\n  </sitemap>\n", baseURL, url.PathEscape(repo.Name)))
		}
	}
	b.WriteString("</sitemapindex>\n")
	return os.WriteFile("site/sitemap.xml", []byte(b.String()), 0644)
}

func generateIndex(cfg Config) error {
	tmpl, err := template.New("index").Funcs(template.FuncMap{
		"escape":     html.EscapeString,
		"pathEscape": url.PathEscape,
		"pillClass": func(s string) string {
			if s == "" {
				return "pill"
			}
			return "pill " + s
		},
		"shieldsURL": func(upstream string) string {
			if upstream == "" {
				return ""
			}
			return fmt.Sprintf("https://img.shields.io/github/stars/%s?style=flat&logo=github&color=818cf8&labelColor=1a1d27", upstream)
		},
		"totalRepos": func() int {
			total := 0
			for _, cat := range cfg.Categories {
				total += len(cat.Repos)
			}
			return total
		},
	}).Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	f, err := os.Create("site/index.html")
	if err != nil {
		return fmt.Errorf("creating index.html: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, cfg)
}

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Supermodel Tools — Architecture Docs</title>
  <meta name="description" content="Architecture documentation for popular open source repositories. Browse code graphs, dependency diagrams, and codebase structure.">
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
  --green: #22c55e;
  --orange: #f59e0b;
  --red: #ef4444;
  --blue: #3b82f6;
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
a:focus-visible { outline: 2px solid var(--accent-light); outline-offset: 2px; border-radius: 2px; }
.container { max-width: var(--max-w); margin: 0 auto; padding: 0 24px; }
.site-header {
  border-bottom: 1px solid var(--border);
  padding: 16px 0;
  position: sticky;
  top: 0;
  background: var(--bg);
  z-index: 100;
}
.site-header .container {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.site-brand {
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
  flex-shrink: 0;
}
.site-brand:hover { text-decoration: none; color: var(--accent-light); }
.site-brand svg { width: 24px; height: 24px; }
.site-nav { display: flex; gap: 16px; align-items: center; }
.site-nav a { color: var(--text-muted); font-size: 14px; font-weight: 500; white-space: nowrap; }
.site-nav a:hover { color: var(--text); text-decoration: none; }
.hero {
  padding: 64px 0 48px;
  text-align: center;
}
.hero h1 {
  font-size: 36px;
  font-weight: 700;
  margin-bottom: 12px;
}
.hero p {
  color: var(--text-muted);
  font-size: 18px;
  max-width: 600px;
  margin: 0 auto;
}
.hero-stats {
  display: flex;
  justify-content: center;
  gap: 32px;
  margin-top: 32px;
}
.hero-stat { text-align: center; }
.hero-stat .num {
  font-size: 28px;
  font-weight: 700;
  color: var(--accent-light);
}
.hero-stat .label {
  font-size: 13px;
  color: var(--text-muted);
}
.search-box {
  max-width: 480px;
  margin: 24px auto 0;
  position: relative;
}
.search-input {
  width: 100%;
  padding: 10px 16px 10px 40px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 14px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s;
}
.search-input:focus { border-color: var(--accent); }
.search-input::placeholder { color: var(--text-muted); }
.search-icon {
  position: absolute;
  left: 12px;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  color: var(--text-muted);
  pointer-events: none;
}
.section-title {
  font-size: 22px;
  font-weight: 700;
  margin-bottom: 16px;
}
.section { margin-bottom: 48px; }
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 16px;
}
.card {
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 24px;
  transition: border-color 0.2s;
  display: flex;
  flex-direction: column;
}
.card:hover {
  border-color: var(--accent);
  text-decoration: none;
}
.card.hidden { display: none; }
.card-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text);
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.card-title svg { width: 18px; height: 18px; flex-shrink: 0; color: var(--accent-light); }
.card-desc {
  font-size: 14px;
  color: var(--text-muted);
  flex: 1;
  margin-bottom: 12px;
}
.card-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  align-items: center;
}
.pill {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: 20px;
  font-size: 12px;
  color: var(--text-muted);
  font-weight: 500;
}
.pill-accent { border-color: var(--accent); color: var(--accent-light); }
.pill-green { border-color: var(--green); color: var(--green); }
.pill-blue { border-color: var(--blue); color: var(--blue); }
.pill-orange { border-color: var(--orange); color: var(--orange); }
.star-badge { height: 20px; vertical-align: middle; }
.site-footer {
  border-top: 1px solid var(--border);
  padding: 32px 0;
  margin-top: 64px;
  color: var(--text-muted);
  font-size: 13px;
  text-align: center;
}
.no-results {
  text-align: center;
  color: var(--text-muted);
  padding: 48px 0;
  font-size: 15px;
  display: none;
}
.no-results a { cursor: pointer; }
.submit-box {
  max-width: 480px;
  margin: 16px auto 0;
}
.submit-label {
  font-size: 13px;
  color: var(--text-muted);
  margin-bottom: 6px;
}
.submit-row {
  display: flex;
  gap: 8px;
}
.submit-input {
  flex: 1;
  padding: 10px 14px;
  background: var(--bg-card);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  color: var(--text);
  font-size: 14px;
  font-family: var(--mono);
  outline: none;
  transition: border-color 0.2s;
}
.submit-input:focus { border-color: var(--accent); }
.submit-input::placeholder { color: var(--text-muted); font-family: var(--font); }
.submit-btn {
  padding: 10px 20px;
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: var(--radius);
  font-size: 14px;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.2s;
  opacity: 0.4;
  pointer-events: none;
}
.submit-btn.active {
  opacity: 1;
  pointer-events: auto;
}
.submit-btn.active:hover { background: var(--accent-light); }
.submit-btn.loading {
  opacity: 0.6;
  pointer-events: none;
}
.submit-feedback {
  margin-top: 10px;
  font-size: 13px;
  font-family: var(--mono);
  display: none;
}
.submit-feedback.visible { display: block; }
.submit-feedback.preview { color: var(--text-muted); }
.submit-feedback.success { color: var(--green); }
.submit-feedback.success a { color: var(--green); text-decoration: underline; }
.submit-feedback.error { color: var(--red); }
@media (max-width: 768px) {
  .container { padding: 0 16px; }
  .hero { padding: 40px 0 32px; }
  .hero h1 { font-size: 24px; }
  .hero p { font-size: 15px; }
  .hero-stats { flex-wrap: wrap; gap: 12px; }
  .card-grid { grid-template-columns: 1fr; }
  .card { padding: 18px; }
  .section-title { font-size: 18px; }
  .site-footer { margin-top: 40px; padding: 24px 0; }
  .submit-row { flex-direction: column; }
  .submit-btn { width: 100%; }
  .site-nav { gap: 10px; flex-wrap: wrap; justify-content: flex-end; }
}
  </style>
</head>
<body>
  <header class="site-header">
    <div class="container">
      <a href="/" class="site-brand">
        <svg viewBox="0 0 90 78" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M90 61.1124C75.9375 73.4694 59.8419 78 44.7554 78C29.669 78 11.8614 72.6122 0 61.1011V16.9458C11.6168 6 29.891 0 44.9887 0C62.77 0 78.8723 6.97959 89.9887 16.9458V61.1124H90ZM88.1881 38.9553C77.7923 22.8824 59.8983 15.7959 44.7554 15.7959C29.6126 15.7959 13.4515 21.9008 1.556 38.9444C12.5382 54.69 26.9 62.5085 44.7554 62.0944C67.6297 61.5639 77.6495 51.9184 88.1881 38.9553ZM44.7554 16.3475C32.4756 16.3475 22.3888 26.6879 22.2554 38.9388C34.3765 38.9162 44.7554 29.1429 44.7554 16.3475C44.7554 29.1429 55.1344 38.9162 67.2554 38.9388C67.1202 26.5216 57.1141 16.3475 44.7554 16.3475ZM44.7554 61.5639C44.7554 48.4898 34.3765 38.9613 22.2554 38.9388C22.3888 51.1897 32.4756 61.5639 44.7554 61.5639C57.0352 61.5639 67.122 51.1897 67.2554 38.9388C55.1344 38.9613 44.7554 48.4898 44.7554 61.5639Z" fill="currentColor"/>
        </svg>
        Supermodel Tools
      </a>
      <nav class="site-nav">
        <a href="https://supermodeltools.com">Website</a>
        <a href="https://github.com/supermodeltools">GitHub</a>
        <a href="https://x.com/supermodeltools">X</a>
      </nav>
    </div>
  </header>

  <main>
    <div class="container">
      <div class="hero">
        <h1>Architecture Docs</h1>
        <p>Browse architecture documentation, dependency graphs, and code structure for popular open source repositories.</p>
        <div class="hero-stats">
          <div class="hero-stat">
            <div class="num">{{totalRepos}}</div>
            <div class="label">Repositories</div>
          </div>
          {{range .Categories}}
          <div class="hero-stat">
            <div class="num">{{len .Repos}}</div>
            <div class="label">{{.Name}}</div>
          </div>
          {{end}}
        </div>
        <div class="search-box">
          <svg class="search-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
          <input type="text" class="search-input" id="search" placeholder="Search repositories..." autocomplete="off">
        </div>
        <div class="submit-box">
          <div class="submit-label">Don't see your repo? Paste a URL to generate arch docs:</div>
          <div class="submit-row">
            <input type="text" class="submit-input" id="submit-url" placeholder="https://github.com/owner/repo" autocomplete="off" spellcheck="false">
            <button class="submit-btn" id="submit-btn" type="button">Request</button>
          </div>
          <div class="submit-feedback" id="submit-feedback"></div>
        </div>
      </div>

      <div id="no-results" class="no-results">
        No repositories match your search.
        <br><a id="no-results-request">Request docs for this repo &rarr;</a>
      </div>

      {{range .Categories}}
      <div class="section" data-section="{{.Slug}}">
        <h2 class="section-title">{{.Name}}</h2>
        <div class="card-grid">
          {{range .Repos}}
          <a href="/{{pathEscape .Name}}/" class="card" data-name="{{escape .Name}}" data-desc="{{escape .Desc}}">
            <div class="card-title">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2L2 7l10 5 10-5-10-5z"/><path d="M2 17l10 5 10-5"/><path d="M2 12l10 5 10-5"/></svg>
              {{.Name}}
            </div>
            <div class="card-desc">{{.Desc}}</div>
            <div class="card-meta">
              <span class="{{pillClass .PillClass}}">{{.Pill}}</span>
              {{if .Upstream}}<img class="star-badge" src="{{shieldsURL .Upstream}}" alt="GitHub Stars" loading="lazy">{{end}}
            </div>
          </a>
          {{end}}
        </div>
      </div>
      {{end}}
    </div>
  </main>

  <footer class="site-footer">
    <div class="container">
      <p>Generated with <a href="https://github.com/supermodeltools/arch-docs">arch-docs</a> by <a href="https://supermodeltools.com">supermodeltools</a></p>
    </div>
  </footer>

  <script>
  (function() {
    var searchInput = document.getElementById('search');
    var cards = document.querySelectorAll('.card');
    var sections = document.querySelectorAll('.section');
    var noResults = document.getElementById('no-results');
    var submitInput = document.getElementById('submit-url');
    var submitBtn = document.getElementById('submit-btn');
    var feedback = document.getElementById('submit-feedback');
    var noResultsRequest = document.getElementById('no-results-request');

    var API_URL = '/api/request';

    // --- Search ---
    searchInput.addEventListener('input', function() {
      var q = this.value.toLowerCase().trim();
      var anyVisible = false;

      cards.forEach(function(card) {
        var name = (card.getAttribute('data-name') || '').toLowerCase();
        var desc = (card.getAttribute('data-desc') || '').toLowerCase();
        var match = !q || name.indexOf(q) !== -1 || desc.indexOf(q) !== -1;
        card.classList.toggle('hidden', !match);
        if (match) anyVisible = true;
      });

      sections.forEach(function(section) {
        var visibleCards = section.querySelectorAll('.card:not(.hidden)');
        section.style.display = visibleCards.length ? '' : 'none';
      });

      noResults.style.display = anyVisible ? 'none' : 'block';
    });

    // --- Submit form ---
    function parseRepo(val) {
      val = val.trim().replace(/\/+$/, '').replace(/\.git$/, '');
      var m = val.match(/github\.com\/([a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+)/);
      if (m) return m[1];
      m = val.match(/^([a-zA-Z0-9._-]+\/[a-zA-Z0-9._-]+)$/);
      if (m) return m[1];
      return null;
    }

    function showFeedback(msg, type) {
      feedback.className = 'submit-feedback visible ' + type;
      feedback.innerHTML = msg;
    }

    submitInput.addEventListener('input', function() {
      var parsed = parseRepo(this.value);
      if (parsed) {
        var name = parsed.split('/')[1];
        showFeedback('\u2192 repos.supermodeltools.com/' + name + '/', 'preview');
        submitBtn.classList.add('active');
      } else {
        feedback.className = 'submit-feedback';
        submitBtn.classList.remove('active');
      }
    });

    async function submitRequest() {
      var parsed = parseRepo(submitInput.value);
      if (!parsed) return;

      var repoUrl = 'https://github.com/' + parsed;
      var name = parsed.split('/')[1];

      // Loading state
      submitBtn.classList.add('loading');
      submitBtn.textContent = 'Submitting...';
      showFeedback('Setting up ' + name + '...', 'preview');

      try {
        var resp = await fetch(API_URL, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url: repoUrl }),
        });
        var data = await resp.json();

        if (!resp.ok || !data.success) {
          showFeedback(data.error || 'Something went wrong. Please try again.', 'error');
          submitBtn.classList.remove('loading');
          submitBtn.textContent = 'Request';
          return;
        }

        // Success — show the link, clear the input
        showFeedback(
          '\u2713 Submitted! Docs will be generated at <a href="' + data.docs_url + '">' +
          'repos.supermodeltools.com/' + name + '/</a>', 'success'
        );
        submitInput.value = '';
        submitBtn.classList.remove('active', 'loading');
        submitBtn.textContent = 'Request';
      } catch (e) {
        showFeedback('Network error. Please try again.', 'error');
        submitBtn.classList.remove('loading');
        submitBtn.textContent = 'Request';
      }
    }

    submitBtn.addEventListener('click', submitRequest);
    submitInput.addEventListener('keydown', function(e) {
      if (e.key === 'Enter' && submitBtn.classList.contains('active')) submitRequest();
    });

    // "No results" link: scroll up and focus the submit input
    noResultsRequest.addEventListener('click', function() {
      var q = searchInput.value.trim();
      submitInput.value = q;
      submitInput.dispatchEvent(new Event('input'));
      submitInput.focus();
      window.scrollTo({ top: 0, behavior: 'smooth' });
    });
  })();
  </script>
</body>
</html>
`
