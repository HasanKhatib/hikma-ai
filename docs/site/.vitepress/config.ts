import { readFileSync } from "node:fs";
import { defineConfig } from "vitepress";

// Project Pages are served from /hikma-ai/. Override with VITEPRESS_BASE for another host.
const base = process.env.VITEPRESS_BASE ?? "/hikma-ai/";

// The nav shows the latest released version, read from the changelog so it never goes stale.
const latest =
  readFileSync(new URL("../../../CHANGELOG.md", import.meta.url), "utf8").match(/^## \[(\d[^\]]*)\]/m)?.[1] ?? "";

export default defineConfig({
  lang: "en-US",
  title: "Hikma",
  description:
    "Skills follow you. Define your source, publish to a registry repo you own, and install into any agent with one CLI.",
  base,
  cleanUrls: true,
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
    ["meta", { property: "og:image", content: "https://hasankhatib.github.io/hikma-ai/og.png" }],
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
          { text: "What's new", link: "https://github.com/HasanKhatib/hikma-ai/blob/main/CHANGELOG.md" },
          { text: "Releases", link: "https://github.com/HasanKhatib/hikma-ai/releases" },
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
          { text: "skill list", link: "/commands/skill-list" },
          { text: "skill info", link: "/commands/skill-info" },
          { text: "skill install", link: "/commands/skill-install" },
          { text: "skill update", link: "/commands/skill-update" },
          { text: "skill diff", link: "/commands/skill-diff" },
          { text: "skill verify", link: "/commands/skill-verify" },
          { text: "skill remove", link: "/commands/skill-remove" },
          { text: "skill adopt", link: "/commands/skill-adopt" },
          { text: "sync", link: "/commands/sync" },
          { text: "skill create", link: "/commands/skill-create" },
          { text: "skill push", link: "/commands/skill-push" },
          { text: "skill import", link: "/commands/skill-import" },
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
          { text: "Agents and folders", link: "/guides/agents" },
          { text: "Skill format", link: "/guides/skill-format" },
          { text: "Run a registry", link: "/guides/registry" },
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
      copyright: "Hikma means wisdom.",
    },
  },
});
