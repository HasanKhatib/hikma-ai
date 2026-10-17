# Lessons

From building a Hikma demo loop. Opinions marked as such; the rest was observed.

## Scenarios

- Personal beats over feature tours. Each left-index line is a short question and an answer: "Got an idea? Create it on the spot." Keep all of them in the same pattern.
- Good set for a registry CLI: found a skill you like and keep it in your registry, got an idea and create it on the spot, new laptop and same skills. Drop scenarios that need a second agent or a teammate to make sense.
- One concrete, invented-but-plain name per example: `myghaccount/myrepo` for the registry, a creative skill name such as `bs-detector`. Real public repos such as `anthropics/skills` are fine as a source because the README already uses them.

## Length and shape

- 18 seconds for three scenarios (6 each). Fifteen was too tight when a scenario had two commands. Twenty left the last scene idle.
- The first frame equals the last frame: scene one's state is set at t=0 and the last 0.5 seconds return the index and clear the terminal. Check the seam by snapshotting 0 and the final time.
- 16:9, 1920x1080. A side-by-side layout does not stack well into square or vertical, so decide the format before building.
- Research on X video was vendor opinion, not data: lead with the result, keep it short, caption or make it readable muted, upload natively. Nothing found tests personal hooks.

## Layout sizes that fit at 1920x1080

- Terminal: 1100 wide, mono font 30px for commands and 25px for output. JetBrains Mono is 0.6em per character, so 30px is 18px per character and the 1028px inside the padding holds about 57 characters. Count the longest line before writing it.
- Index: 580 wide, titles 48px (two lines is fine), sub-line 28px. Give each item and the bar 226px or three-line items touch the bar's border.
- Pixel character: a 20x16 grid at 14px per cell, built from `<rect>` elements by a small helper. Layers: base, two arm frames that alternate while typing, a lightbulb, a prompt glyph and a check glyph on the screen, and a wave arm for the end.

## Hyperframes gotchas

- A `tl.set(..., 0)` inside the paused timeline does not render on frame 0. Set initial states with `gsap.set` before building the timeline, or the bulb and wave arm flash.
- Typing is deterministic: tween a counter and slice the string in `onUpdate`. Alternate the typing arm with `tl.set` calls every 0.16s.
- `npm run check` flags real text overlap (a two-line title ran into the next item). It also warns that scenes should be sub-compositions; that is a structure preference, not an error.
- Fonts are fetched and embedded automatically. The first render needs the network.
- `hyperframes init` offers skills install; set `HYPERFRAMES_SKIP_SKILLS=1`.

## Output honesty

- The CLI's help for subcommands may not resolve through `go run`. Build the binary and run that.
- A real `skill push` of a new skill warns about unfilled placeholders. If the video skips editing `SKILL.md`, say so in the report.
- Report the success line you wrote yourself (for example `✓ published to ...`) as approximated.
