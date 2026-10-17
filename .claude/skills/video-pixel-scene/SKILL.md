---
name: video-pixel-scene
description: "Builds or changes the pixel-art, game-style animated scene for a web page or video: a character and a pet walking through a CLI's workflow, with speech balloons, a terminal overlay, clickable stages, and light and dark themes. Use when the user asks for the home page animation, a hero scene, a game-style demo, a pixel-art loop, or to change its story, pacing, colors, or sprites. For a plain terminal recording use video-tech-demo."
---

# Pixel scene

A short story played as pixel art: a developer and a bot pet go through the product's steps, the terminal shows the real commands, and the viewer can jump between steps. Hikma's version lives in `docs/site/.vitepress/theme/` (`heroScene.ts`, `HeroScene.vue`, `InstallLine.vue`) and tells this story: start a repo (`hikma init`), find and install a skill (the bot turns blue), create and push a new one (the bot turns violet), fall in a hole and lose everything, respawn on a new laptop where `hikma sync` restores the skills.

## Rules

- Propose the storyboard and sprite sketches first and wait for approval. Do not build or render before that.
- Show real commands and real output. Run the built CLI in a temp directory (`HOME` and `XDG_CONFIG_HOME` set to temp paths, `NO_COLOR=1`). Shorten names for the page and say what was approximated (the push confirmation comes from the docs, not a live run).
- No org-specific names or private registries. Use `myghaccount/myrepo`.
- Keep the scene readable with the sound off and with no motion: it needs a still frame for reduced motion and a Pause button.
- Keep brand elements the user removed removed. Ask before bringing one back.
- The home page must still fit one screen. Never add a section below a full-height hero; put new content inside the hero.

## How it is built

- **A pure function of time.** `drawScene(ctx, seconds, dark)` draws every frame from the time alone, so the same code draws the live canvas, seeks when a stage is clicked, and renders stills or video frames. No per-frame state.
- **Canvas for the world, DOM for text.** Sprites and ground are drawn at a small logical size (240x135) with `image-rendering: pixelated`. Balloons and the terminal overlay are DOM text so they are crisp, selectable, and accessible. The canvas gets a text description for screen readers.
- **Sprites are string grids** with a palette (`"..hhhh.."`), drawn at 2x. Pet color is a palette override per skill count, so state is shown by color.
- **Timeline in scene seconds**, with `SLOW` to stretch it and `INTRO` for a title screen. Positions, balloons (`BALLOONS`), and terminal lines (`CLI`, with groups that clear between steps) are tables you edit.
- **Stages** (`STAGES`) hold a start time in real seconds. The active one lights, later ones dim, and a click sets the clock to its start. The scene state is correct anywhere because it derives from time.
- **Theme.** `useData().isDark` selects the palette (night with stars, or day with sun and clouds). CSS `.dark` rules switch the overlay and controls.
- **`?t=12.6`** freezes the scene at that second, for stills and video frames.

## Workflow

1. **Brief and storyboard.** One screen, 20 to 40 seconds, a table of beats: time, what is pictured, balloon (8 words or fewer), terminal line. Draw the sprites as ASCII grids. Wait for approval.
2. **Capture real output** from the built CLI and put it in the `CLI` table.
3. **Build** the scene and the Vue component, then wire the hero. Remove anything the scene replaces, including its CSS.
4. **Check frames, not just the build.** `npm run docs:build` only proves it compiles. Render stills (below) at every beat, in dark and light, and at 1366x650 and 1280x800 to confirm the hero still fits.
5. **Report** what was and was not checked. Do not claim it plays smoothly unless it was watched in a real browser.

## Checking frames

- Serve a copy of `.vitepress/dist` under `/hikma-ai/` on a port you own (check the port is free; never kill a process on a port without confirming it is yours).
- Screenshot with Chrome: `--headless=new --window-size=1366,650 --virtual-time-budget=1500 --screenshot=out.png "http://localhost:PORT/hikma-ai/?t=12.6"`.
- For light mode, inject `localStorage.setItem("vitepress-theme-appearance","light")` into the copy's `<head>`. Headless Chrome ignores the preferred-color-scheme flag here.
- CSS transitions do not advance under virtual time, so a stage button looks half-switched. Add `<style>*{transition:none!important}</style>` to the test copy.
- Crop with `sips -c H W --cropOffset Y X`, tile with `ffmpeg -filter_complex hstack/vstack` into a contact sheet, and read it.
- Headless Chrome will not size a window narrower than about 500px, so phone layouts need a real device or a browser's device mode.

## Making a video

The scene is a function of time, so a video is a sequence of frames: load `?t=` at a fixed step (15 fps), screenshot each, and assemble with `ffmpeg` into an MP4 and a GIF (`fps=15`, `scale=960`, palettegen and paletteuse). Not built yet; see board task E7.6-T8. For a terminal-only clip use `video-tech-demo` instead.

## Gotchas

- A PR that changes after review can be merged without the last commit. Check the merge result: Hikma's `main` carried a scrolling home page until the scene replaced it.
- Task's built-in shell has no `kill`; call it through `env kill`.
- Negative scene time happens on the title screen; use `((t % n) + n) % n` for modulo.
- Keep the scene's dark panel and text colors independent of any hard-coded page color, or light mode breaks.

## Social card

`assets/og-template.html` renders the 1200x630 social card (`docs/site/public/og.png`) from a crop of the scene: screenshot the home page at `?t=13` in dark mode, crop the scene box to `scene.png` beside the template, and screenshot the template with Chrome at `--window-size=1200,630`.
