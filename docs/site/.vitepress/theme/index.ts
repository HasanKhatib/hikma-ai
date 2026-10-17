import DefaultTheme from "vitepress/theme";
import type { Theme } from "vitepress";
import { h } from "vue";
import HeroScene from "./HeroScene.vue";
import InstallLine from "./InstallLine.vue";
import DevBadge from "./DevBadge.vue";
import "./custom.css";

export default {
  extends: DefaultTheme,
  Layout: () => h(DefaultTheme.Layout, null, { "layout-bottom": () => h(DevBadge) }),
  enhanceApp({ app }) {
    app.component("HeroScene", HeroScene);
    app.component("InstallLine", InstallLine);
  },
} satisfies Theme;
