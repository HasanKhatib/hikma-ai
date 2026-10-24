import { execFileSync } from "node:child_process";
import { existsSync, readFileSync, readdirSync, statSync, writeFileSync } from "node:fs";
import { join, relative, resolve } from "node:path";
import { defineConfig, type HeadConfig } from "vitepress";

// The site is served from the root of hikma.alkhatib.tech. Override with VITEPRESS_BASE for a sub-path.
const base = process.env.VITEPRESS_BASE ?? "/";

// The nav shows the latest released version, from the newest release tag so it never goes stale.
// Without tags (a shallow clone) it just says "Releases".
const latest = (() => {
  try {
    return execFileSync("git", ["describe", "--tags", "--abbrev=0", "--match", "v[0-9]*"], {
      encoding: "utf8",
      stdio: ["ignore", "pipe", "ignore"],
    })
      .trim()
      .replace(/^v/, "");
  } catch {
    return "";
  }
})();

const SITE = "https://hikma.alkhatib.tech";

// The first paragraph of a page, as plain text, for pages that do not set their own description.
// Search results and assistants quote it, so a page never inherits the home page's tagline.
function firstParagraph(source: string): string {
  const body = source.replace(/^---[\s\S]*?---\n/, "");
  let inFence = false;
  for (const block of body.split(/\n\s*\n/)) {
    const text = block.trim();
    if (text.startsWith("```")) inFence = !inFence;
    if (!text || inFence || /^(#|```|:::|<|\||-|\d+\.)/.test(text)) continue;
    const plain = text
      .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
      .replace(/[`*_]/g, "")
      .replace(/\s+/g, " ");
    return plain.length > 200 ? plain.slice(0, plain.lastIndexOf(" ", 197)) + "..." : plain;
  }
  return "";
}

// Every page's source, in one file, so an assistant can read the whole documentation in one request.
function writeLlmsFull(srcDir: string, outDir: string) {
  const files: string[] = [];
  const walk = (dir: string) => {
    for (const name of readdirSync(dir).sort()) {
      const p = join(dir, name);
      if (name.startsWith(".") || name === "node_modules" || name === "public") continue;
      if (statSync(p).isDirectory()) walk(p);
      else if (name.endsWith(".md")) files.push(p);
    }
  };
  walk(srcDir);
  const parts = ["# Hikma documentation\n\n> The whole of https://hikma.alkhatib.tech/ as one file. A short index is at /llms.txt.\n"];
  for (const f of files) {
    const rel = relative(srcDir, f).replace(/\\/g, "/");
    if (rel === "index.md") continue;
    const url = `${SITE}/${rel.replace(/(^|\/)index\.md$/, "$1").replace(/\.md$/, "")}`;
    parts.push(`\n---\nSource: ${url}\n\n${readFileSync(f, "utf8").replace(/^---[\s\S]*?---\n/, "").trim()}\n`);
  }
  writeFileSync(join(outDir, "llms-full.txt"), parts.join(""));
}

export default defineConfig({
  lang: "en-US",
  sitemap: { hostname: SITE },
  transformPageData(pageData, { siteConfig }) {
    if (pageData.frontmatter.description || pageData.isNotFound) return;
    const file = resolve(siteConfig.srcDir, pageData.relativePath);
    if (!existsSync(file)) return;
    const text = firstParagraph(readFileSync(file, "utf8"));
    if (text) pageData.description = text;
  },
  transformHead({ pageData }) {
    if (pageData.isNotFound) return [];
    const path = pageData.relativePath.replace(/(^|\/)index\.md$/, "$1").replace(/\.md$/, "");
    const url = `${SITE}/${path}`;
    const head: HeadConfig[] = [
      ["link", { rel: "canonical", href: url }],
      ["meta", { property: "og:url", content: url }],
      ["meta", { property: "og:site_name", content: "Hikma" }],
      ["meta", { property: "og:image:alt", content: "Hikma: a pixel-art scene of skills moving from a registry chest to a bot" }],
    ];
    if (pageData.relativePath === "index.md") {
      head.push([
        "script",
        { type: "application/ld+json" },
        JSON.stringify({
          "@context": "https://schema.org",
          "@type": "SoftwareApplication",
          name: "Hikma",
          alternateName: "hikma",
          description:
            "A command-line tool for scaffolding, sharing, and installing reusable AI agent skills through a GitHub repository you own.",
          applicationCategory: "DeveloperApplication",
          operatingSystem: "macOS, Linux, Windows",
          url: SITE,
          downloadUrl: "https://github.com/HasanKhatib/hikma-ai/releases",
          codeRepository: "https://github.com/HasanKhatib/hikma-ai",
          license: "https://opensource.org/licenses/MIT",
          author: { "@type": "Person", name: "Hasan Alkhatib" },
          offers: { "@type": "Offer", price: "0", priceCurrency: "USD" },
        }),
      ]);
    }
    return head;
  },
  buildEnd(siteConfig) {
    writeLlmsFull(siteConfig.srcDir, siteConfig.outDir);
  },
  title: "Hikma",
  description:
    "Skills follow you. Define your source, publish to a registry repo you own, and install into any agent with one CLI.",
  base,
  cleanUrls: true,
  // First-time visitors get the dark theme whatever their system setting; the toggle still wins and is remembered.
  appearance: "dark",
  lastUpdated: true,
  head: [
    ["link", { rel: "icon", href: `${base}logo/favicon.ico`, sizes: "48x48" }],
    ["link", { rel: "icon", type: "image/png", sizes: "32x32", href: `${base}logo/favicon-32x32.png` }],
    ["link", { rel: "icon", type: "image/png", sizes: "16x16", href: `${base}logo/favicon-16x16.png` }],
    ["link", { rel: "apple-touch-icon", sizes: "180x180", href: `${base}logo/apple-touch-icon.png` }],
    ["link", { rel: "manifest", href: `${base}logo/site.webmanifest` }],
    // Social card: shown when a page link is shared. Absolute URLs, as link previews require.
    ["meta", { property: "og:type", content: "website" }],
    ["meta", { property: "og:title", content: "Hikma: skills follow you" }],
    ["meta", { property: "og:description", content: "Define your source, publish to a registry you own, and install into any agent with one CLI." }],
    ["meta", { property: "og:image", content: "https://hikma.alkhatib.tech/og.png" }],
    ["meta", { name: "twitter:card", content: "summary_large_image" }],
  ],

  themeConfig: {
    logo: "/logo/Logo.png",
    nav: [
      { text: "Get Started", link: "/getting-started" },
      { text: "Commands", link: "/commands/" },
      { text: "Guides", link: "/guides/" },
      {
        text: latest ? `v${latest}` : "Releases",
        items: [
          { text: "What's new", link: "https://github.com/HasanKhatib/hikma-ai/releases/latest" },
          { text: "All releases", link: "https://github.com/HasanKhatib/hikma-ai/releases" },
        ],
      },
    ],

    sidebar: [
      {
        text: "Introduction",
        items: [
          { text: "What is Hikma?", link: "/" },
          { text: "Getting started", link: "/getting-started" },
        ],
      },
      {
        text: "Commands",
        items: [
          { text: "Overview", link: "/commands/" },
          { text: "init", link: "/commands/init" },
          { text: "config", link: "/commands/config" },
          { text: "doctor", link: "/commands/doctor" },
          { text: "registry validate", link: "/commands/registry-validate" },
          {
            text: "skill",
            collapsed: true,
            items: [
              { text: "list", link: "/commands/skill-list" },
              { text: "info", link: "/commands/skill-info" },
              { text: "install", link: "/commands/skill-install" },
              { text: "update", link: "/commands/skill-update" },
              { text: "diff", link: "/commands/skill-diff" },
              { text: "verify", link: "/commands/skill-verify" },
              { text: "remove", link: "/commands/skill-remove" },
              { text: "adopt", link: "/commands/skill-adopt" },
              { text: "create", link: "/commands/skill-create" },
              { text: "push", link: "/commands/skill-push" },
              { text: "import", link: "/commands/skill-import" },
            ],
          },
          { text: "sync", link: "/commands/sync" },
          { text: "completion", link: "/commands/completion" },
          { text: "upgrade", link: "/commands/upgrade" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "Exit codes and JSON", link: "/reference/output" },
          { text: "Configuration and environment", link: "/reference/configuration" },
          { text: "Flags and terminal behavior", link: "/reference/flags" },
          { text: "Versioning and deprecation", link: "/reference/policy" },
        ],
      },
      {
        text: "Guides",
        items: [
          { text: "Overview", link: "/guides/" },
          { text: "The menu", link: "/guides/menu" },
          { text: "Agents and folders", link: "/guides/agents" },
          { text: "Write and edit a skill", link: "/guides/write-a-skill" },
          { text: "Skill format", link: "/guides/skill-format" },
          { text: "Run a registry", link: "/guides/registry" },
          { text: "Use in CI", link: "/guides/use-in-ci" },
          { text: "Troubleshooting", link: "/guides/troubleshooting" },
          { text: "Glossary", link: "/guides/glossary" },
        ],
      },
    ],

    socialLinks: [{ icon: "github", link: "https://github.com/HasanKhatib/hikma-ai" }],
    search: { provider: "local" },
    editLink: {
      pattern: "https://github.com/HasanKhatib/hikma-ai/edit/main/docs/site/:path",
      text: "Edit this page on GitHub",
    },
    footer: {
      message: "Released under the MIT License.",
      copyright: "Hikma means wisdom. <a href=\"https://github.com/HasanKhatib/hikma-ai/releases\">Release history</a>",
    },
  },
});
