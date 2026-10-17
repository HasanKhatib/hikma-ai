import { defineConfig } from "vitepress";

// Project Pages are served from /hikma-ai/. Override with VITEPRESS_BASE for another host.
const base = process.env.VITEPRESS_BASE ?? "/hikma-ai/";

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
  ],

  themeConfig: {
    logo: "/logo/Logo.png",
    nav: [
      { text: "Get Started", link: "/getting-started" },
      { text: "Commands", link: "/commands/" },
      { text: "Guides", link: "/guides/" },
      {
        text: "v0.2",
        items: [
          { text: "Releases", link: "https://github.com/HasanKhatib/hikma-ai/releases" },
          { text: "Changelog", link: "https://github.com/HasanKhatib/hikma-ai/blob/main/CHANGELOG.md" },
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
          { text: "completion", link: "/commands/completion" },
        ],
      },
      {
        text: "Guides",
        items: [
          { text: "Overview", link: "/guides/" },
          { text: "Agents and folders", link: "/guides/agents" },
          { text: "Skill format", link: "/guides/skill-format" },
          { text: "Run a registry", link: "/guides/registry" },
          { text: "Set up a team repo", link: "/guides/team-setup" },
          { text: "Menu and wizard", link: "/guides/menu" },
          { text: "Global installs", link: "/guides/global" },
          { text: "Lockfile and updates", link: "/guides/lockfile" },
          { text: "Using Hikma with gh skill", link: "/guides/gh-skill" },
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
