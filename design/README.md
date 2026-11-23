# Design

A [FigCraft](https://figcraft.app) mirror of the MatchOwl UI, used to iterate on
the design before (or alongside) changing the code.

Pages: **Tokens** (the `Theme` variable collection with Dark / Light / Black
modes, taken from `frontend/src/lib/theme.css`), **Mobile · Dark**,
**Mobile · Light** and **Desktop · Dark**. It was built from the live app's DOM,
captured as a made-up demo user (Mira Novak). Keep it that way: no real
names, emails or avatars in here, because the repo is public.

## Layout

```
design/
  matchowl/              committed source of truth
    document.json        the document, pretty-printed so it diffs and merges
    images/<sha256>      image fills, named by content hash (deduped)
    thumbnail.png
  matchowl.figcraft      gitignored, built from matchowl/
  tools/figcraft_sync.py pack / unpack / status (python3, stdlib only)
```

Git can't diff or merge a `.figcraft` because it's a zip. That's why the repo
stores it unpacked and treats the `.figcraft` as a local build.

## Workflow

First time on a machine:

```sh
make design-hooks   # once per clone: keeps the .figcraft in sync automatically
make design-pack    # builds design/matchowl.figcraft
```

Then open `design/matchowl.figcraft` in FigCraft.

- **After editing in FigCraft:** save, then run `make design-unpack` and commit
  `design/matchowl/`. With the hooks on, `git commit` runs the unpack for you
  and tells you to stage the result. It doesn't stage anything itself.
- **After a pull or branch switch:** the hooks rebuild the `.figcraft`.
  Without them, run `make design-pack`. Reopen the file in FigCraft if it was
  open.
- **`make design-status`** tells you which side changed since the last sync.

Neither direction overwrites unsynced work. `pack` refuses if the `.figcraft`
has edits that weren't unpacked, and `unpack` refuses if `design/matchowl/`
changed since the last sync (for example, a pull you haven't packed yet). If
both sides changed, unpack your edits to a branch with `--force`, then merge.

`unpack` resets the selected page to the first one, so browsing around in
FigCraft doesn't show up as a change. Vector path data (`network`) stays on
one line per node. Everything else gets one property per line, so a change
like a corner radius going from 8 to 12 shows up as a one-line diff.

## Proposals

UI suggestions go on their own page, as frames placed side by side and labeled
by variant (for example, "Match card — A / B / C"). Pick one or tweak it, then
the change lands in the code and the mirror frame gets updated to match.

## Fonts

To see the real type, install Figtree, Baloo 2, Red Hat Mono and DSEG7 (on
Linux, for example, in `~/.local/share/fonts/`).
