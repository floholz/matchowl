# App layout rework — design canvas (2026-09-12)

Mockups for the shell restructure decided on 2026-09-12. The live, editable
canvas is on claude.ai (owner: floholz):

**https://claude.ai/code/artifact/86f644c6-5d76-4790-b02a-ac30e4892c17**

`screens/` holds a PNG of every artboard so the design is readable straight
from the repo. The decisions themselves are in [`/PLAN.md`](../../../PLAN.md)
("Now: app layout rework").

## Files

| File | What |
|------|------|
| `build.mjs` | Generates every artboard (`*.dc.html`) and `canvas.json` from the real theme tokens (`frontend/src/lib/theme.css`). Edit this, not the HTML. |
| `dseg7.woff2` | DSEG7 Classic Bold (SIL OFL) — the seven-segment face for the score board. Embedded as base64 by the build. |
| `*.dc.html` | Generated artboards (Design Components format). Each renders as a plain HTML page in a browser too. |
| `canvas.json` | Artboard positions, pages and sticky notes for the canvas. |
| `screens/*.png` | Rendered artboards. |

## Artboards

Page "Screens": `Home`, `Main` (= Matches), `Competition` (hub), `Match`
(upcoming), `MatchFT` (after full time, penalties, 2nd leg), `Friends`,
`RowStates` (match row anatomy), `KOStates` (extra time / pens / two legs),
`DesktopMatches`, `DesktopHome`.

Open items, drawn 2026-09-12 (second row of "Screens", proposals awaiting
confirmation): `HubOverview`, `HubTable` (UCL zones), `HubKnockout`,
`HubForecast` (calls mode), `League` (friends league page with tabs),
`Tablet` (Matches at 768px).

Friends section pass, 2026-09-13 (third row of "Screens", supersedes
`Friends` and `League`): `Pools`, `FriendsTab`, `PoolStart`, `PoolDone`,
`PoolPage`, `PoolMembers`.

Page "Score vs tip": `OptionA`–`OptionD` — the four candidates for telling the
result apart from the tip. **A (stadium board) was chosen.**

## Regenerating

```sh
cd docs/design/app-layout-2026-09
node build.mjs          # rewrites *.dc.html + canvas.json
# quick look, no tooling needed:
google-chrome-stable --headless=new --hide-scrollbars --screenshot=out.png --window-size=390,844 "file://$PWD/Main.dc.html"
```

To push changes back to the canvas from Claude Code: run `/design`, point it
at this folder, and ask it to re-seed and update the artifact URL above.
