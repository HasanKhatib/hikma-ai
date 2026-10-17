import DefaultTheme from "vitepress/theme";
import type { Theme } from "vitepress";
import { h } from "vue";
import InstallPanel from "./InstallPanel.vue";
import DevBadge from "./DevBadge.vue";
import "./custom.css";

export default {
  extends: DefaultTheme,
  Layout: () => h(DefaultTheme.Layout, null, { "layout-bottom": () => h(DevBadge) }),
  enhanceApp({ app }) {
    app.component("InstallPanel", InstallPanel);
  },
} satisfies Theme;
