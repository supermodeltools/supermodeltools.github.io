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

	// generateRedirects must run before generateRepoPages so placeholder
	// index.html files don't exist yet when we check for local arch-docs.
	if err := generateRedirects(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating _redirects: %v\n", err)
		os.Exit(1)
	}

	if err := generateRepoPages(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating repo pages: %v\n", err)
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
	fmt.Printf("Generated index.html, sitemap.xml, and %d repo pages\n", totalRepos)
}

func generateSitemap(cfg Config) error {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	b.WriteString(fmt.Sprintf("  <url>\n    <loc>%s/</loc>\n    <priority>1.0</priority>\n  </url>\n", baseURL))
	for _, cat := range cfg.Categories {
		for _, repo := range cat.Repos {
			b.WriteString(fmt.Sprintf("  <url>\n    <loc>%s/%s/</loc>\n    <priority>0.8</priority>\n  </url>\n", baseURL, url.PathEscape(repo.Name)))
		}
	}
	b.WriteString("</urlset>\n")
	return os.WriteFile("site/sitemap.xml", []byte(b.String()), 0644)
}

func generateRepoPages(cfg Config) error {
	tmpl, err := template.New("repo").Funcs(template.FuncMap{
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
			return fmt.Sprintf("https://img.shields.io/github/stars/%s?style=flat&logo=github&color=8CC6C9&labelColor=161616", upstream)
		},
		"archDocsURL": func(name string) string {
			return fmt.Sprintf("https://graphtechnologydevelopers.github.io/%s/", url.PathEscape(name))
		},
	}).Parse(repoTemplate)
	if err != nil {
		return fmt.Errorf("parsing repo template: %w", err)
	}

	for _, cat := range cfg.Categories {
		for _, repo := range cat.Repos {
			dir := fmt.Sprintf("site/%s", url.PathEscape(repo.Name))
			// Skip community repos that have proxy rules in _redirects (upstream set)
			if repo.Upstream != "" {
				continue
			}
			// Skip if arch-docs have already been deployed to this directory
			if _, err := os.Stat(fmt.Sprintf("%s/index.html", dir)); err == nil {
				continue
			}
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("creating dir %s: %w", dir, err)
			}
			f, err := os.Create(fmt.Sprintf("%s/index.html", dir))
			if err != nil {
				return fmt.Errorf("creating %s/index.html: %w", dir, err)
			}
			err = tmpl.Execute(f, repo)
			f.Close()
			if err != nil {
				return fmt.Errorf("executing repo template for %s: %w", repo.Name, err)
			}
		}
	}
	return nil
}

func generateRedirects(cfg Config) error {
	var b strings.Builder
	for _, cat := range cfg.Categories {
		for _, repo := range cat.Repos {
			if repo.Upstream == "" {
				continue // supermodel repos use centralized site/ subdirectories
			}
			// Check if arch-docs are already deployed locally (centralized approach)
			if _, err := os.Stat(fmt.Sprintf("site/%s/index.html", url.PathEscape(repo.Name))); err == nil {
				continue // has local arch-docs, no proxy needed
			}
			b.WriteString(fmt.Sprintf("/%s/* https://graphtechnologydevelopers.github.io/%s/:splat 302\n",
				url.PathEscape(repo.Name), url.PathEscape(repo.Name)))
		}
	}
	if b.Len() == 0 {
		return nil
	}
	return os.WriteFile("site/_redirects", []byte(b.String()), 0644)
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
			return fmt.Sprintf("https://img.shields.io/github/stars/%s?style=flat&logo=github&color=8CC6C9&labelColor=161616", upstream)
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

const repoTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{escape .Name}} — Supermodel Tools</title>
  <meta name="description" content="Architecture documentation for {{escape .Name}}. {{escape .Desc}}">
  <link rel="canonical" href="https://repos.supermodeltools.com/{{pathEscape .Name}}/">
  <meta property="og:type" content="article">
  <meta property="og:title" content="{{escape .Name}} — Architecture Docs">
  <meta property="og:description" content="{{escape .Desc}}">
  <meta property="og:url" content="https://repos.supermodeltools.com/{{pathEscape .Name}}/">
  <meta property="og:site_name" content="Supermodel Tools">
  <meta name="twitter:card" content="summary">
  <meta name="twitter:title" content="{{escape .Name}} — Architecture Docs">
  <meta name="twitter:description" content="{{escape .Desc}}">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Public+Sans:wght@200;300;400;500;600;700&family=Martian+Mono:wght@300;400;500&family=Lexend+Peta:wght@400&display=swap" rel="stylesheet">
  <style>
:root {
  --bg: #000000;
  --bg-card: #161616;
  --border: #202020;
  --text: #FFFFFF;
  --text-muted: #808080;
  --accent: #71B9BC;
  --accent-light: #8CC6C9;
  --green: #7CCE86;
  --orange: #D0A27D;
  --blue: #8E8CE9;
  --font: 'Public Sans', -apple-system, BlinkMacSystemFont, sans-serif;
  --max-w: 1200px;
  --radius: 0px;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: var(--font); background: var(--bg); color: var(--text); line-height: 1.5; font-weight: 300; -webkit-font-smoothing: antialiased; }
a { color: var(--accent-light); text-decoration: none; }
a:hover { text-decoration: underline; }
.container { max-width: var(--max-w); margin: 0 auto; padding: 0 24px; }
.site-header { border-bottom: 1px solid var(--border); padding: 16px 0; position: sticky; top: 0; background: var(--bg); z-index: 100; }
.site-header .container { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.site-brand { display: flex; align-items: center; gap: 10px; }
.site-brand:hover { text-decoration: none; opacity: 0.8; }
.logo-icon { height: 20px; display: flex; align-items: center; }
.logo-icon svg { height: 100%; width: auto; }
.logo-wordmark { height: 16px; display: flex; align-items: center; }
.logo-wordmark svg { height: 100%; width: auto; }
.site-nav { display: flex; gap: 16px; align-items: center; }
.site-nav a { color: var(--text-muted); font-size: 0.68rem; font-weight: 300; font-family: 'Martian Mono', monospace; text-transform: uppercase; letter-spacing: .08em; }
.site-nav a:hover { color: var(--text); text-decoration: none; }
.breadcrumb { padding: 24px 0 0; font-size: 14px; color: var(--text-muted); display: flex; align-items: center; gap: 8px; }
.breadcrumb span { color: var(--border); }
.entity-header { padding: 32px 0 48px; }
.entity-header h1 { font-size: 32px; font-weight: 300; letter-spacing: -0.02em; margin-bottom: 12px; font-family: var(--font); }
.entity-header p { color: var(--text-muted); font-size: 16px; max-width: 600px; margin-bottom: 20px; }
.card-meta { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; margin-bottom: 24px; }
.pill { display: inline-flex; align-items: center; padding: 4px 10px; background: var(--bg-card); border: 1px solid var(--border); border-radius: 20px; font-size: 12px; color: var(--text-muted); font-weight: 500; }
.pill-accent { border-color: var(--accent); color: var(--accent-light); }
.pill-green { border-color: var(--green); color: var(--green); }
.pill-blue { border-color: var(--blue); color: var(--blue); }
.pill-orange { border-color: var(--orange); color: var(--orange); }
.star-badge { height: 20px; vertical-align: middle; }
.btn { display: inline-flex; align-items: center; gap: 8px; padding: 10px 20px; border-radius: var(--radius); font-size: 14px; font-weight: 600; transition: opacity 0.2s; }
.btn:hover { opacity: 0.85; text-decoration: none; }
.btn-primary { background: var(--accent); color: #000; }
.site-footer { border-top: 1px solid var(--border); padding: 32px 0; margin-top: 64px; color: var(--text-muted); font-size: 13px; text-align: center; }
  </style>
</head>
<body>
  <header class="site-header">
    <div class="container">
      <a href="https://supermodeltools.com" class="site-brand">
        <span class="logo-icon"><svg viewBox="0 0 90 78" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M90 61.1124C75.9375 73.4694 59.8419 78 44.7554 78C29.669 78 11.8614 72.6122 0 61.1011V16.9458C11.6168 6 29.891 0 44.9887 0C62.77 0 78.8723 6.97959 89.9887 16.9458V61.1124H90ZM88.1881 38.9553C77.7923 22.8824 59.8983 15.7959 44.7554 15.7959C29.6126 15.7959 13.4515 21.9008 1.556 38.9444C12.5382 54.69 26.9 62.5085 44.7554 62.0944C67.6297 61.5639 77.6495 51.9184 88.1881 38.9553ZM44.7554 16.3475C32.4756 16.3475 22.3888 26.6879 22.2554 38.9388C34.3765 38.9162 44.7554 29.1429 44.7554 16.3475C44.7554 29.1429 55.1344 38.9162 67.2554 38.9388C67.1202 26.5216 57.1141 16.3475 44.7554 16.3475ZM44.7554 61.5639C44.7554 48.4898 34.3765 38.9613 22.2554 38.9388C22.3888 51.1897 32.4756 61.5639 44.7554 61.5639C57.0352 61.5639 67.122 51.1897 67.2554 38.9388C55.1344 38.9613 44.7554 48.4898 44.7554 61.5639Z" fill="#61B8BC"/></svg></span>
        <span class="logo-wordmark"><svg viewBox="0 0 599 66" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M23.0873 0C30.6289 0 36.7264 2.17281 41.8071 6.16637L42.0132 6.32909V19.4152L40.986 17.713C36.1927 9.7765 29.3085 6.24042 21.73 6.24033C16.2897 6.24033 10.9687 9.40925 10.9686 15.2141C10.9686 17.3774 11.548 19.1892 12.8179 20.7592C14.0976 22.3411 16.1111 23.7184 19.0499 24.9434L31.2658 30.0109C35.4509 31.7279 38.6869 34.0259 40.876 36.9242C43.0688 39.8276 44.1861 43.3005 44.1862 47.317C44.1862 52.893 41.9482 57.5803 38.0664 60.8659C34.1913 64.1458 28.7144 66 22.2727 66C13.2811 66 6.33602 63.2904 1.32495 58.0934L1.19332 57.9571L1.17605 57.769L0 44.6079L1.02716 44.2878C7.38706 54.937 15.1949 59.6708 23.7207 59.6709C27.3779 59.6709 30.3373 58.6763 32.3739 56.9269C34.4021 55.1845 35.5686 52.6484 35.5686 49.4514C35.5685 47.2029 34.9232 45.1173 33.5175 43.2724C32.1079 41.4221 29.9103 39.785 26.7633 38.4785L15.0934 33.7671L15.088 33.765C10.7294 31.9618 7.58148 29.7323 5.5253 27.008C3.46381 24.2765 2.53229 21.09 2.53229 17.4372C2.53236 12.2578 4.65951 7.88309 8.3133 4.81074C11.9608 1.74391 17.097 9.1588e-06 23.0873 0Z" fill="white"/><path fill-rule="evenodd" clip-rule="evenodd" d="M395.406 0C415.003 0 428.633 14.1288 428.633 33C428.633 51.8712 415.003 66 395.406 66C375.806 65.9998 362.27 51.8691 362.27 33C362.27 14.1309 375.806 0.00023386 395.406 0ZM395.406 6.15052C388.851 6.15064 383.007 8.54118 378.798 13.066C374.587 17.5936 371.973 24.3021 371.973 33C371.973 41.6979 374.587 48.4064 378.798 52.934C383.007 57.4588 388.851 59.8494 395.406 59.8495C401.961 59.8495 407.805 57.459 412.014 52.934C416.225 48.4064 418.839 41.6979 418.839 33C418.839 24.3021 416.225 17.5936 412.014 13.066C407.805 8.54103 401.961 6.15052 395.406 6.15052Z" fill="white"/><path d="M67.9824 2.18824C66.2453 4.56874 65.4252 5.69742 65.0218 6.52139C64.6517 7.27742 64.6463 7.74277 64.6463 8.89981V40.914C64.6463 46.7288 66.314 51.3205 69.2631 54.4534C72.2074 57.5809 76.485 59.3148 81.834 59.3148C87.0424 59.3148 91.2052 57.5841 94.0714 54.4587C96.944 51.3262 98.5684 46.7325 98.5685 40.914V8.72125C98.5685 7.56246 98.5621 7.1043 98.1865 6.37558C97.7746 5.57652 96.9356 4.49442 95.1514 2.20197L94.476 1.33344L108.803 1.33344L108.28 2.15971C106.772 4.53886 106.058 5.66822 105.707 6.49497C105.38 7.26259 105.376 7.7413 105.376 8.89981V40.3804C105.376 48.5221 103.132 54.942 98.8652 59.3286C94.595 63.719 88.3701 65.9989 80.5662 65.9989C72.7212 65.9989 66.4518 63.7651 62.1464 59.4089C57.8436 55.0551 55.5767 48.6559 55.5767 40.4692L55.5767 8.72125C55.5767 7.56095 55.5714 7.08627 55.2476 6.34388C54.8977 5.54174 54.1864 4.45823 52.6775 2.1671L52.1283 1.33344L68.6071 1.33344L67.9824 2.18824Z" fill="white"/><path fill-rule="evenodd" clip-rule="evenodd" d="M143.979 1.33344C150.548 1.33344 155.868 3.09393 159.555 6.17588C163.25 9.26459 165.259 13.6431 165.259 18.7707C165.259 23.8981 163.25 28.2767 159.555 31.3654C155.868 34.4473 150.548 36.2079 143.979 36.2079H130.955V57.2766C130.955 58.4358 130.962 58.8969 131.329 59.6276C131.73 60.427 132.548 61.5091 134.286 63.8012L134.941 64.6655H118.458L118.982 63.8392C120.49 61.4599 121.203 60.3297 121.554 59.5029C121.881 58.7354 121.886 58.2574 121.886 57.0991V8.72125C121.886 7.56095 121.88 7.08627 121.557 6.34388C121.207 5.54177 120.496 4.45799 118.988 2.1671L118.437 1.33344L143.979 1.33344ZM130.955 30.5909H142.441C146.952 30.5909 150.263 29.332 152.443 27.2637C154.621 25.1963 155.737 22.2582 155.737 18.7707C155.737 15.2831 154.621 12.345 152.443 10.2776C150.263 8.20934 146.952 6.95037 142.441 6.95037L130.955 6.95037V30.5909Z" fill="white"/><path d="M215.074 13.1125L214.138 12.2228C211.437 9.65763 209.662 8.42303 208.218 7.81785C206.794 7.22146 205.654 7.21769 204.113 7.21769L184.573 7.21769V28.1893L200.403 28.1893C201.858 28.1893 202.906 28.1856 204.147 27.793C205.395 27.3982 206.876 26.5961 209.119 24.9212L210.006 24.2587V38.0052L209.119 37.3427C206.876 35.6677 205.395 34.8646 204.147 34.4697C202.906 34.0772 201.858 34.0735 200.403 34.0735H184.573V58.7812H202.575C204.289 58.7812 205.551 58.7778 207.298 58.0765C209.074 57.3636 211.375 55.9171 215.077 52.9456L216.158 52.0781L214.526 64.1953L214.464 64.6655L172.075 64.6655L172.599 63.8392C174.107 61.4599 174.82 60.3297 175.172 59.5029C175.498 58.7355 175.503 58.2573 175.503 57.0991V8.72125C175.503 7.56095 175.498 7.08627 175.174 6.34388C174.824 5.54175 174.113 4.45814 172.604 2.1671L172.055 1.33344L215.074 1.33344V13.1125Z" fill="white"/><path fill-rule="evenodd" clip-rule="evenodd" d="M249.455 1.33344C256.102 1.33347 261.509 2.95422 265.266 5.89799C269.036 8.85233 271.097 13.1025 271.097 18.2371C271.097 22.7752 269.46 26.6093 266.46 29.4635C263.652 32.1362 259.682 33.9182 254.828 34.6504L271.88 55.8714L272.993 57.2291C273.977 58.4221 274.614 59.1683 275.313 59.896C276.246 60.8675 277.292 61.8088 279.425 63.7262L280.471 64.6655H267.558L244.114 35.0519H236.793V57.1879C236.793 58.3928 236.8 58.8627 237.211 59.5948C237.664 60.4021 238.587 61.4827 240.555 63.7758L241.318 64.6655H224.295L224.82 63.8392C226.327 61.4599 227.04 60.3297 227.392 59.5029C227.718 58.7354 227.723 58.2574 227.723 57.0991V8.72125C227.723 7.56095 227.718 7.08627 227.394 6.34388C227.044 5.54175 226.333 4.45807 224.824 2.1671L224.275 1.33344L249.455 1.33344ZM236.793 29.7013L248.278 29.7013C252.583 29.7013 255.899 28.6546 258.131 26.7449C260.352 24.8446 261.574 22.027 261.574 18.3258C261.574 14.6247 260.352 11.807 258.131 9.90676C255.899 7.99709 252.583 6.95037 248.278 6.95037L236.793 6.95037V29.7013Z" fill="white"/><path d="M319.307 35.5697L342.18 1.33344L353.413 1.33344L352.909 2.15337C351.447 4.53226 350.755 5.66221 350.413 6.48969C350.096 7.2598 350.092 7.7411 350.092 8.89981V57.2766C350.092 58.4371 350.096 58.9146 350.411 59.6593C350.751 60.4621 351.441 61.5464 352.904 63.8371L353.432 64.6655H336.431L337.163 63.7832C339.174 61.3579 340.125 60.2072 340.592 59.373C341.016 58.6159 341.021 58.1636 341.021 57.0104V14.2061L317.861 49.3827L294.789 14.2938V57.2766C294.789 58.4357 294.796 58.8945 295.171 59.6234C295.583 60.4224 296.422 61.5039 298.206 63.7959L298.883 64.6655L284.736 64.6655L285.26 63.8392C286.768 61.4599 287.481 60.3297 287.832 59.5029C288.159 58.7354 288.164 58.2574 288.164 57.0991V8.72125C288.164 7.56094 288.158 7.08627 287.834 6.34388C287.484 5.54175 286.773 4.45809 285.264 2.1671L284.715 1.33344L296.345 1.33344L319.307 35.5697Z" fill="white"/><path fill-rule="evenodd" clip-rule="evenodd" d="M462.146 1.33344C471.31 1.33344 479.53 4.12535 485.465 9.50314C491.407 14.8866 495.011 22.8181 495.011 32.9989C495.011 43.1799 491.407 51.1123 485.465 56.4958C479.53 61.8734 471.31 64.6655 462.146 64.6655H437.349L437.873 63.8392C439.381 61.4599 440.094 60.3297 440.445 59.5029C440.772 58.7355 440.777 58.2574 440.777 57.0991V8.72125C440.777 7.56095 440.771 7.08627 440.447 6.34388C440.098 5.54175 439.386 4.45814 437.877 2.1671L437.328 1.33344L462.146 1.33344ZM449.846 58.87H460.698C467.229 58.87 473.374 56.7321 477.884 52.4532C482.388 48.179 485.308 41.7235 485.308 32.9989C485.308 24.2296 482.388 17.7755 477.885 13.513C473.376 9.24538 467.23 7.12792 460.698 7.12788L449.846 7.12788V58.87Z" fill="white"/><path d="M546.694 13.1125L545.756 12.2228C543.055 9.65771 541.281 8.42305 539.837 7.81785C538.413 7.22128 537.272 7.21769 535.732 7.21769L516.191 7.21769V28.1893H532.021C533.476 28.1893 534.525 28.1857 535.766 27.793C537.014 27.3982 538.494 26.5958 540.737 24.9212L541.625 24.2587V38.0052L540.737 37.3427C538.494 35.6678 537.014 34.8646 535.766 34.4697C534.525 34.077 533.476 34.0735 532.021 34.0735H516.191V58.7812H534.193C535.907 58.7812 537.169 58.7778 538.917 58.0765C540.693 57.3636 542.994 55.9175 546.696 52.9456L547.776 52.0781L546.082 64.6655L503.694 64.6655L504.218 63.8392C505.725 61.4599 506.438 60.3297 506.79 59.5029C507.116 58.7354 507.121 58.2574 507.121 57.0991V8.72125C507.121 7.56095 507.116 7.08627 506.792 6.34388C506.442 5.54177 505.732 4.45798 504.223 2.1671L503.673 1.33344L546.694 1.33344V13.1125Z" fill="white"/><path d="M571.402 2.16499C569.848 4.54437 569.114 5.67416 568.752 6.50026C568.417 7.26545 568.412 7.74154 568.412 8.89981V58.6037H585.056C586.861 58.6037 588.208 58.5998 590.017 57.8261C591.855 57.04 594.199 55.444 597.9 52.1627L599 51.1874L597.311 64.6655H555.914L556.438 63.8392C557.945 61.4599 558.659 60.3297 559.011 59.5029C559.337 58.7355 559.342 58.2573 559.342 57.0991V8.72125C559.342 7.56096 559.337 7.08627 559.013 6.34388C558.663 5.54174 557.952 4.4582 556.443 2.1671L555.894 1.33344L571.945 1.33344L571.402 2.16499Z" fill="white"/></svg></span>
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
      <nav class="breadcrumb" aria-label="Breadcrumb">
        <a href="/">Home</a><span>/</span>{{escape .Name}}
      </nav>
      <div class="entity-header">
        <h1>{{escape .Name}}</h1>
        <p>{{escape .Desc}}</p>
        <div class="card-meta">
          {{if .Pill}}<span class="{{pillClass .PillClass}}">{{escape .Pill}}</span>{{end}}
          {{if .Upstream}}<img class="star-badge" src="{{shieldsURL .Upstream}}" alt="GitHub Stars" loading="lazy">{{end}}
        </div>
        <a href="{{archDocsURL .Name}}" class="btn btn-primary">
          <svg width="16" height="16" viewBox="0 0 90 78" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M90 61.1124C75.9375 73.4694 59.8419 78 44.7554 78C29.669 78 11.8614 72.6122 0 61.1011V16.9458C11.6168 6 29.891 0 44.9887 0C62.77 0 78.8723 6.97959 89.9887 16.9458V61.1124H90ZM88.1881 38.9553C77.7923 22.8824 59.8983 15.7959 44.7554 15.7959C29.6126 15.7959 13.4515 21.9008 1.556 38.9444C12.5382 54.69 26.9 62.5085 44.7554 62.0944C67.6297 61.5639 77.6495 51.9184 88.1881 38.9553ZM44.7554 16.3475C32.4756 16.3475 22.3888 26.6879 22.2554 38.9388C34.3765 38.9162 44.7554 29.1429 44.7554 16.3475C44.7554 29.1429 55.1344 38.9162 67.2554 38.9388C67.1202 26.5216 57.1141 16.3475 44.7554 16.3475ZM44.7554 61.5639C44.7554 48.4898 34.3765 38.9613 22.2554 38.9388C22.3888 51.1897 32.4756 61.5639 44.7554 61.5639C57.0352 61.5639 67.122 51.1897 67.2554 38.9388C55.1344 38.9613 44.7554 48.4898 44.7554 61.5639Z" fill="#61B8BC"/></svg>
          View Architecture Docs
        </a>
      </div>
    </div>
  </main>

  <footer class="site-footer">
    <div class="container">
      <p>Generated with <a href="https://github.com/supermodeltools/arch-docs">arch-docs</a> by <a href="https://supermodeltools.com">supermodeltools</a></p>
    </div>
  </footer>
</body>
</html>
`

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Supermodel Tools — Architecture Docs</title>
  <meta name="description" content="Architecture documentation for popular open source repositories. Browse code graphs, dependency diagrams, and codebase structure.">
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Public+Sans:wght@200;300;400;500;600;700&family=Martian+Mono:wght@300;400;500&family=Lexend+Peta:wght@400&display=swap" rel="stylesheet">
  <style>
:root {
  --bg: #000000;
  --bg-card: #161616;
  --bg-hover: #08191C;
  --border: #202020;
  --text: #FFFFFF;
  --text-muted: #808080;
  --accent: #71B9BC;
  --accent-light: #8CC6C9;
  --green: #7CCE86;
  --orange: #D0A27D;
  --red: #E589C6;
  --blue: #8E8CE9;
  --font: 'Public Sans', -apple-system, BlinkMacSystemFont, sans-serif;
  --mono: 'Martian Mono', 'Fira Code', monospace;
  --max-w: 1200px;
  --radius: 0px;
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
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.site-brand:hover { text-decoration: none; opacity: 0.8; }
.logo-icon { height: 20px; display: flex; align-items: center; }
.logo-icon svg { height: 100%; width: auto; }
.logo-wordmark { height: 16px; display: flex; align-items: center; }
.logo-wordmark svg { height: 100%; width: auto; }
.site-nav { display: flex; gap: 16px; align-items: center; }
.site-nav a { color: var(--text-muted); font-size: 0.68rem; font-weight: 300; font-family: var(--mono); text-transform: uppercase; letter-spacing: .08em; white-space: nowrap; }
.site-nav a:hover { color: var(--text); text-decoration: none; }
.hero {
  padding: 64px 0 48px;
  text-align: center;
}
.hero h1 {
  font-size: 36px;
  font-weight: 200;
  letter-spacing: -0.04em;
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
  flex-wrap: wrap;
  gap: 16px 32px;
  margin-top: 32px;
}
.hero-stat { text-align: center; }
.hero-stat .num {
  font-size: 28px;
  font-weight: 300;
  letter-spacing: -0.02em;
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
  font-weight: 300;
  letter-spacing: -0.02em;
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
}
  </style>
</head>
<body>
  <header class="site-header">
    <div class="container">
      <a href="https://supermodeltools.com" class="site-brand">
        <span class="logo-icon"><svg viewBox="0 0 90 78" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M90 61.1124C75.9375 73.4694 59.8419 78 44.7554 78C29.669 78 11.8614 72.6122 0 61.1011V16.9458C11.6168 6 29.891 0 44.9887 0C62.77 0 78.8723 6.97959 89.9887 16.9458V61.1124H90ZM88.1881 38.9553C77.7923 22.8824 59.8983 15.7959 44.7554 15.7959C29.6126 15.7959 13.4515 21.9008 1.556 38.9444C12.5382 54.69 26.9 62.5085 44.7554 62.0944C67.6297 61.5639 77.6495 51.9184 88.1881 38.9553ZM44.7554 16.3475C32.4756 16.3475 22.3888 26.6879 22.2554 38.9388C34.3765 38.9162 44.7554 29.1429 44.7554 16.3475C44.7554 29.1429 55.1344 38.9162 67.2554 38.9388C67.1202 26.5216 57.1141 16.3475 44.7554 16.3475ZM44.7554 61.5639C44.7554 48.4898 34.3765 38.9613 22.2554 38.9388C22.3888 51.1897 32.4756 61.5639 44.7554 61.5639C57.0352 61.5639 67.122 51.1897 67.2554 38.9388C55.1344 38.9613 44.7554 48.4898 44.7554 61.5639Z" fill="#61B8BC"/></svg></span>
        <span class="logo-wordmark"><svg viewBox="0 0 599 66" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M23.0873 0C30.6289 0 36.7264 2.17281 41.8071 6.16637L42.0132 6.32909V19.4152L40.986 17.713C36.1927 9.7765 29.3085 6.24042 21.73 6.24033C16.2897 6.24033 10.9687 9.40925 10.9686 15.2141C10.9686 17.3774 11.548 19.1892 12.8179 20.7592C14.0976 22.3411 16.1111 23.7184 19.0499 24.9434L31.2658 30.0109C35.4509 31.7279 38.6869 34.0259 40.876 36.9242C43.0688 39.8276 44.1861 43.3005 44.1862 47.317C44.1862 52.893 41.9482 57.5803 38.0664 60.8659C34.1913 64.1458 28.7144 66 22.2727 66C13.2811 66 6.33602 63.2904 1.32495 58.0934L1.19332 57.9571L1.17605 57.769L0 44.6079L1.02716 44.2878C7.38706 54.937 15.1949 59.6708 23.7207 59.6709C27.3779 59.6709 30.3373 58.6763 32.3739 56.9269C34.4021 55.1845 35.5686 52.6484 35.5686 49.4514C35.5685 47.2029 34.9232 45.1173 33.5175 43.2724C32.1079 41.4221 29.9103 39.785 26.7633 38.4785L15.0934 33.7671L15.088 33.765C10.7294 31.9618 7.58148 29.7323 5.5253 27.008C3.46381 24.2765 2.53229 21.09 2.53229 17.4372C2.53236 12.2578 4.65951 7.88309 8.3133 4.81074C11.9608 1.74391 17.097 9.1588e-06 23.0873 0Z" fill="white"/><path fill-rule="evenodd" clip-rule="evenodd" d="M395.406 0C415.003 0 428.633 14.1288 428.633 33C428.633 51.8712 415.003 66 395.406 66C375.806 65.9998 362.27 51.8691 362.27 33C362.27 14.1309 375.806 0.00023386 395.406 0ZM395.406 6.15052C388.851 6.15064 383.007 8.54118 378.798 13.066C374.587 17.5936 371.973 24.3021 371.973 33C371.973 41.6979 374.587 48.4064 378.798 52.934C383.007 57.4588 388.851 59.8494 395.406 59.8495C401.961 59.8495 407.805 57.459 412.014 52.934C416.225 48.4064 418.839 41.6979 418.839 33C418.839 24.3021 416.225 17.5936 412.014 13.066C407.805 8.54103 401.961 6.15052 395.406 6.15052Z" fill="white"/><path d="M67.9824 2.18824C66.2453 4.56874 65.4252 5.69742 65.0218 6.52139C64.6517 7.27742 64.6463 7.74277 64.6463 8.89981V40.914C64.6463 46.7288 66.314 51.3205 69.2631 54.4534C72.2074 57.5809 76.485 59.3148 81.834 59.3148C87.0424 59.3148 91.2052 57.5841 94.0714 54.4587C96.944 51.3262 98.5684 46.7325 98.5685 40.914V8.72125C98.5685 7.56246 98.5621 7.1043 98.1865 6.37558C97.7746 5.57652 96.9356 4.49442 95.1514 2.20197L94.476 1.33344L108.803 1.33344L108.28 2.15971C106.772 4.53886 106.058 5.66822 105.707 6.49497C105.38 7.26259 105.376 7.7413 105.376 8.89981V40.3804C105.376 48.5221 103.132 54.942 98.8652 59.3286C94.595 63.719 88.3701 65.9989 80.5662 65.9989C72.7212 65.9989 66.4518 63.7651 62.1464 59.4089C57.8436 55.0551 55.5767 48.6559 55.5767 40.4692L55.5767 8.72125C55.5767 7.56095 55.5714 7.08627 55.2476 6.34388C54.8977 5.54174 54.1864 4.45823 52.6775 2.1671L52.1283 1.33344L68.6071 1.33344L67.9824 2.18824Z" fill="white"/><path fill-rule="evenodd" clip-rule="evenodd" d="M143.979 1.33344C150.548 1.33344 155.868 3.09393 159.555 6.17588C163.25 9.26459 165.259 13.6431 165.259 18.7707C165.259 23.8981 163.25 28.2767 159.555 31.3654C155.868 34.4473 150.548 36.2079 143.979 36.2079H130.955V57.2766C130.955 58.4358 130.962 58.8969 131.329 59.6276C131.73 60.427 132.548 61.5091 134.286 63.8012L134.941 64.6655H118.458L118.982 63.8392C120.49 61.4599 121.203 60.3297 121.554 59.5029C121.881 58.7354 121.886 58.2574 121.886 57.0991V8.72125C121.886 7.56095 121.88 7.08627 121.557 6.34388C121.207 5.54177 120.496 4.45799 118.988 2.1671L118.437 1.33344L143.979 1.33344ZM130.955 30.5909H142.441C146.952 30.5909 150.263 29.332 152.443 27.2637C154.621 25.1963 155.737 22.2582 155.737 18.7707C155.737 15.2831 154.621 12.345 152.443 10.2776C150.263 8.20934 146.952 6.95037 142.441 6.95037L130.955 6.95037V30.5909Z" fill="white"/><path d="M215.074 13.1125L214.138 12.2228C211.437 9.65763 209.662 8.42303 208.218 7.81785C206.794 7.22146 205.654 7.21769 204.113 7.21769L184.573 7.21769V28.1893L200.403 28.1893C201.858 28.1893 202.906 28.1856 204.147 27.793C205.395 27.3982 206.876 26.5961 209.119 24.9212L210.006 24.2587V38.0052L209.119 37.3427C206.876 35.6677 205.395 34.8646 204.147 34.4697C202.906 34.0772 201.858 34.0735 200.403 34.0735H184.573V58.7812H202.575C204.289 58.7812 205.551 58.7778 207.298 58.0765C209.074 57.3636 211.375 55.9171 215.077 52.9456L216.158 52.0781L214.526 64.1953L214.464 64.6655L172.075 64.6655L172.599 63.8392C174.107 61.4599 174.82 60.3297 175.172 59.5029C175.498 58.7355 175.503 58.2573 175.503 57.0991V8.72125C175.503 7.56095 175.498 7.08627 175.174 6.34388C174.824 5.54175 174.113 4.45814 172.604 2.1671L172.055 1.33344L215.074 1.33344V13.1125Z" fill="white"/><path fill-rule="evenodd" clip-rule="evenodd" d="M249.455 1.33344C256.102 1.33347 261.509 2.95422 265.266 5.89799C269.036 8.85233 271.097 13.1025 271.097 18.2371C271.097 22.7752 269.46 26.6093 266.46 29.4635C263.652 32.1362 259.682 33.9182 254.828 34.6504L271.88 55.8714L272.993 57.2291C273.977 58.4221 274.614 59.1683 275.313 59.896C276.246 60.8675 277.292 61.8088 279.425 63.7262L280.471 64.6655H267.558L244.114 35.0519H236.793V57.1879C236.793 58.3928 236.8 58.8627 237.211 59.5948C237.664 60.4021 238.587 61.4827 240.555 63.7758L241.318 64.6655H224.295L224.82 63.8392C226.327 61.4599 227.04 60.3297 227.392 59.5029C227.718 58.7354 227.723 58.2574 227.723 57.0991V8.72125C227.723 7.56095 227.718 7.08627 227.394 6.34388C227.044 5.54175 226.333 4.45807 224.824 2.1671L224.275 1.33344L249.455 1.33344ZM236.793 29.7013L248.278 29.7013C252.583 29.7013 255.899 28.6546 258.131 26.7449C260.352 24.8446 261.574 22.027 261.574 18.3258C261.574 14.6247 260.352 11.807 258.131 9.90676C255.899 7.99709 252.583 6.95037 248.278 6.95037L236.793 6.95037V29.7013Z" fill="white"/><path d="M319.307 35.5697L342.18 1.33344L353.413 1.33344L352.909 2.15337C351.447 4.53226 350.755 5.66221 350.413 6.48969C350.096 7.2598 350.092 7.7411 350.092 8.89981V57.2766C350.092 58.4371 350.096 58.9146 350.411 59.6593C350.751 60.4621 351.441 61.5464 352.904 63.8371L353.432 64.6655H336.431L337.163 63.7832C339.174 61.3579 340.125 60.2072 340.592 59.373C341.016 58.6159 341.021 58.1636 341.021 57.0104V14.2061L317.861 49.3827L294.789 14.2938V57.2766C294.789 58.4357 294.796 58.8945 295.171 59.6234C295.583 60.4224 296.422 61.5039 298.206 63.7959L298.883 64.6655L284.736 64.6655L285.26 63.8392C286.768 61.4599 287.481 60.3297 287.832 59.5029C288.159 58.7354 288.164 58.2574 288.164 57.0991V8.72125C288.164 7.56094 288.158 7.08627 287.834 6.34388C287.484 5.54175 286.773 4.45809 285.264 2.1671L284.715 1.33344L296.345 1.33344L319.307 35.5697Z" fill="white"/><path fill-rule="evenodd" clip-rule="evenodd" d="M462.146 1.33344C471.31 1.33344 479.53 4.12535 485.465 9.50314C491.407 14.8866 495.011 22.8181 495.011 32.9989C495.011 43.1799 491.407 51.1123 485.465 56.4958C479.53 61.8734 471.31 64.6655 462.146 64.6655H437.349L437.873 63.8392C439.381 61.4599 440.094 60.3297 440.445 59.5029C440.772 58.7355 440.777 58.2574 440.777 57.0991V8.72125C440.777 7.56095 440.771 7.08627 440.447 6.34388C440.098 5.54175 439.386 4.45814 437.877 2.1671L437.328 1.33344L462.146 1.33344ZM449.846 58.87H460.698C467.229 58.87 473.374 56.7321 477.884 52.4532C482.388 48.179 485.308 41.7235 485.308 32.9989C485.308 24.2296 482.388 17.7755 477.885 13.513C473.376 9.24538 467.23 7.12792 460.698 7.12788L449.846 7.12788V58.87Z" fill="white"/><path d="M546.694 13.1125L545.756 12.2228C543.055 9.65771 541.281 8.42305 539.837 7.81785C538.413 7.22128 537.272 7.21769 535.732 7.21769L516.191 7.21769V28.1893H532.021C533.476 28.1893 534.525 28.1857 535.766 27.793C537.014 27.3982 538.494 26.5958 540.737 24.9212L541.625 24.2587V38.0052L540.737 37.3427C538.494 35.6678 537.014 34.8646 535.766 34.4697C534.525 34.077 533.476 34.0735 532.021 34.0735H516.191V58.7812H534.193C535.907 58.7812 537.169 58.7778 538.917 58.0765C540.693 57.3636 542.994 55.9175 546.696 52.9456L547.776 52.0781L546.082 64.6655L503.694 64.6655L504.218 63.8392C505.725 61.4599 506.438 60.3297 506.79 59.5029C507.116 58.7354 507.121 58.2574 507.121 57.0991V8.72125C507.121 7.56095 507.116 7.08627 506.792 6.34388C506.442 5.54177 505.732 4.45798 504.223 2.1671L503.673 1.33344L546.694 1.33344V13.1125Z" fill="white"/><path d="M571.402 2.16499C569.848 4.54437 569.114 5.67416 568.752 6.50026C568.417 7.26545 568.412 7.74154 568.412 8.89981V58.6037H585.056C586.861 58.6037 588.208 58.5998 590.017 57.8261C591.855 57.04 594.199 55.444 597.9 52.1627L599 51.1874L597.311 64.6655H555.914L556.438 63.8392C557.945 61.4599 558.659 60.3297 559.011 59.5029C559.337 58.7355 559.342 58.2573 559.342 57.0991V8.72125C559.342 7.56096 559.337 7.08627 559.013 6.34388C558.663 5.54174 557.952 4.4582 556.443 2.1671L555.894 1.33344L571.945 1.33344L571.402 2.16499Z" fill="white"/></svg></span>
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
      </div>

      <div id="no-results" class="no-results">No repositories match your search.</div>

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
    var input = document.getElementById('search');
    var cards = document.querySelectorAll('.card');
    var sections = document.querySelectorAll('.section');
    var noResults = document.getElementById('no-results');

    input.addEventListener('input', function() {
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
  })();
  </script>
</body>
</html>
`
