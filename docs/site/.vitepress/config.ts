import { defineConfig } from "vitepress";

// Project Pages are served from /hikma-ai/. Override with VITEPRESS_BASE for another host.
const base = process.env.VITEPRESS_BASE ?? "/hikma-ai/";

export default defineConfig({
  lang: "en-US",
  title: "Hikma",
  description:
    "Set up repositories for AI agents and manage reusable skills through a registry you own.",
  base,
  cleanUrls: true,
  lastUpdated: true,
  head: [["link", { rel: "icon", href: `${base}favicon.svg` }]],

  themeConfig: {
    logo: "/logo.svg",
    nav: [
      { text: "Get Started", link: "/getting-started" },
      { text: "Commands", link: "/commands/" },
      { text: "Guides", link: "/guides/" },
      {
        text: "v0.1",
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
          { text: "skill remove", link: "/commands/skill-remove" },
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
