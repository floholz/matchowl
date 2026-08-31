# Matchowl Rework — Implementation Plan

> Status: in progress. Written 2026-08-06, after WC 2026 concluded.
> Progress: Phases 0–3 done (rename; tournaments entity + migrations,
> verified against fresh and populated DBs; backend generalization,
> verified end-to-end via the dev simulator; frontend generalization,
> svelte-check + build clean). Phase 4 (Matchowl identity) in progress;
> Phase 5 (bots, screenshots, tests for non-WC structures) open.
> Scope decided with floholz: **(1) generalize the app to host arbitrary
> tournaments**, **(2) fresh Matchowl visual identity**, with **persistent
> leagues** that live across tournaments and **full migration** of the
> existing WC 2026 data as the archived first tournament.

## Why

wm-pickems was built for exactly one event. There is no tournament entity
anywhere: `teams` / `tournament_groups` / `matches` are flat unscoped tables,
`forecasts` is unique on `user` alone, "tournament start" is
`MIN(matches.kickoff)` over the whole DB, the stage list
`group|R32|R16|QF|SF|3RD|FINAL` is baked into the `matches` schema, the
best-third logic hard-requires 8-of-12 groups, scoring assumes 4-team
groups × 3 games, API-Football is pinned to `league=1`, and the stage
vocabulary is copy-pasted in 7 frontend files. Matchowl is the successor
that must host Euro 2028, club cups, and whatever comes after — without a
rewrite per event.

## Key design decisions

1. **`tournaments` is the new root entity.** Everything tournament-shaped
   (`teams`, `tournament_groups`, `matches`, `forecasts`, `forecast_scores`)
   gains a `tournament` relation. `tips` / `match_scores` stay keyed by
   `match` (tournament derivable; PB filters can join `match.tournament`).
2. **Structure is data, not schema.** `matches.stage` becomes a plain text
   field; the valid stage list, group shape, and qualifier rules move into a
   `structure` JSON on the tournament record, validated in Go:
   ```jsonc
   {
     "stages": [                       // ordered; replaces every hardcoded list
       {"code": "group", "name": "Group stage", "kind": "group"},
       {"code": "R32",   "name": "Round of 32", "kind": "knockout"},
       // ... 3RD marked {"consolation": true}; last knockout stage crowns champion
     ],
     "groupSize": 4, "gamesPerTeam": 3, "directQualifiers": 2,
     "extraQualifiers": {"fromPosition": 3, "count": 8, "tableKey": "wc2026"},
     "pointsWin": 3, "pointsDraw": 1
   }
   ```
   `extraQualifiers: null` disables the whole best-third path (Euro-style
   without thirds, straight-KO cups skip groups entirely). The FIFA Annex-C
   table becomes one entry in a keyed registry
   (`internal/bracket/tables/<key>.json`); the existing greedy fallback in
   `internal/sync/resolve.go` is the universal fallback when a tournament has
   no official table.
3. **Sync config is per-tournament.** A `sync` JSON on the record:
   `{"provider": "api-football|openfootball|manual", "apiFootballLeague": 1,
   "season": 2026, "openfootballURL": "..."}`. The results cron iterates
   tournaments with `status = "active"`. `ExtID` prefixes come from a
   per-tournament `extIdPrefix` (existing rows keep `WC2026-`).
4. **Leagues are persistent.** No tournament FK on `leagues` — a league is a
   lasting friend group. Leaderboards become per-`(league, tournament)`:
   the leaderboard endpoint takes a tournament param, scores are already
   per-user rows that now carry/derive a tournament. No enrollment table —
   every league implicitly participates in every tournament its members play.
   The Global league stays, persistent. League chat stays league-scoped
   (not per-tournament).
5. **One forecast per `(user, tournament)`** — the unique index on
   `forecasts.user` is replaced by `(user, tournament)`. The three-part JSON
   shape (groupOrder / thirdQualifiers / bracket) stays but each section is
   optional per structure.
6. **`status` drives the app.** `draft → upcoming → active → finished →
   archived`. The frontend's "current" tournament = the newest non-archived
   one; archived tournaments stay browsable (read-only history). Locks,
   countdowns, notify detectors, and sync all scope to one tournament instead
   of the global match table.
7. **Migration, not reset.** New migrations create `tournaments`, insert the
   `wc2026` record (structure + sync matching today's hardcoded behavior,
   `status = "archived"`), backfill the FK on every existing row, and swap
   unique indexes (`forecasts.user` → `(user, tournament)`;
   `tournament_groups.letter` → `(tournament, letter)`). Existing pb_data
   volumes upgrade in place; users keep accounts and history.
8. **Fresh visual identity.** Ground-up Matchowl look designed around the owl
   mark (`docs/matchowl-icon.svg`) — new palette, typography, and layout
   language; the current "Floodlight" theme is retired. Structural components
   (TipCard, stores, routes, nav shell) are kept and restyled; the Landing
   page is rebuilt (its WM-pun hero, 1:7 easter egg, and stat blocks are
   WC-specific one-offs). Design details decided at implementation time in
   Phase 4, including whether light mode ships in v1.

## Phases

### Phase 0 — Rename & plumbing (mechanical)

- Go module `github.com/floholz/wm-pickems` → `github.com/floholz/matchowl`
  (all `internal/` imports, `main.go`, bots module import of nothing — bots is
  standalone). Binary `wm-pickems` → `matchowl` in Makefile, Dockerfile,
  docker-compose. Bots binary `wm-pickems-bot` → `matchowl-bot`.
- Neutral strings that aren't UI copy: openfootball User-Agent, `.env.example`
  `MAIL_FROM_NAME`, `main.go` doc comment. Env var `WMP_DEV` keeps working but
  `MATCHOWL_DEV` becomes the documented name (accept both).
- UI copy / emails / manifest are NOT touched here (Phase 4 owns branding).

### Phase 1 — Tournament entity + migrations

- `migrations/00XX_tournaments.go`: create `tournaments` (slug unique, name,
  shortName, status select, startsAt/endsAt, structure JSON, sync JSON,
  extIdPrefix, scoringConfig relation). Insert `wc2026` record with structure
  `{stages: [group,R32,R16,QF,SF,3RD,FINAL], groupSize 4, gamesPerTeam 3,
  directQualifiers 2, extraQualifiers {pos 3, count 8, tableKey wc2026}}` and
  sync `{provider auto, apiFootballLeague 1, season 2026, openfootball URL}`,
  status `archived` (WC 2026 is over).
- `migrations/00XX_tournament_scope.go`: add `tournament` relation to `teams`,
  `tournament_groups`, `matches`, `forecasts`, `forecast_scores`; backfill all
  rows to `wc2026`; make the field required; swap unique indexes
  (`(tournament, letter)`, `(user, tournament)`,
  `(user, tournament, config)`); relax `matches.stage` from Select to Text.
- `internal/tournaments/` (new package): record accessors, structure
  parse/validate (`Structure`, `Stage`, `ExtraQualifiers` types), current/
  active lookups, `GET /api/tournaments` + `GET /api/tournaments/{slug}`
  public endpoints, admin CRUD (`POST/PATCH /api/admin/tournaments...`) with
  structure validation.

### Phase 2 — Backend generalization

- **seed** (`internal/seed`): `Run(app, tournament, fixtureJSON, teamMeta)` —
  parameterized by tournament; openfootball round-label → stage mapping comes
  from a per-source alias map in the seed call, not a package global. WC2026
  embed stays only as the compat boot-seed for empty DBs. Admin endpoint to
  seed a new tournament from an uploaded openfootball JSON + team meta.
- **bracket** (`internal/bracket`): table registry keyed by
  `structure.extraQualifiers.tableKey`; kill the 12-group `A..L` range check
  (derive valid letters from the tournament's groups); keep greedy fallback.
- **sync** (`internal/sync`): cron loops active tournaments; per-tournament
  provider pick; remove the package-global `thirdTeam` map (per-run state);
  `groupStandings` uses `gamesPerTeam` / `groupSize` / `extraQualifiers.count`
  from structure; `applyResult` advancer logic keyed on stage `kind`.
- **football** (`internal/football`): `leagueID` / `season` become call
  params from tournament sync config.
- **scoring** (`internal/scoring`): `scoreValues` branches on stage kind (not
  `!= "group"`); `finalGroups` / `bestThirdSet` / `scoreForecast` take the
  structure (group count from DB, groupSize, qualifier counts); champion =
  winner of the last non-consolation knockout stage (not literal `FINAL`);
  recompute + leaderboard scoped per tournament; `configsInUse` unchanged.
- **forecast** (`internal/forecast`): lock = tournament `startsAt` (fallback
  first kickoff *of that tournament*); `/api/tournaments/{slug}/structure`
  replaces `/api/forecast/structure` (returns stages, groups, KO skeleton,
  third slots/table, lock time); validation driven by structure.
- **tips / leagues / stats / notify**: leaderboard + friends-tips + notify
  detectors take a tournament scope; `stageOrder` / `stageName` from
  structure; invite-code word list decoupled from the 48-team WC field.
- **dev harness**: virtual-clock presets derived from the active tournament's
  fixtures instead of literal 2026 dates.

### Phase 3 — Frontend generalization

- `tournaments.svelte.ts` store: list + current tournament (newest
  non-archived, else newest), structure fetch; stores (`tipsStore`,
  `forecastStore`, `countdown`) parameterized by tournament id with a reset
  path; all collection reads filtered by tournament.
- One `stages.ts` derived from structure replaces the 7 copy-pasted stage
  maps; league leaderboard forecast columns render from `structure.stages`
  (fixes the `CHAMPION`/`3RD` divergence).
- Forecast builder sections (Groups / Best thirds / Bracket) render
  conditionally from structure; `maxThirds`, group-size loops, `i < 2`
  qualifier styling all read structure values.
- Tournament switcher UI: current tournament is the default everywhere; an
  archive view lists past tournaments (read-only tips/forecast/tables).
  League page gains a tournament selector on the leaderboard.
- `/tournament` group tables use `pointsWin`/`pointsDraw` + structure
  tiebreak assumptions from structure.

### Phase 4 — Matchowl identity (fresh redesign)

- New token set in `theme.css` (palette, type, radii, effects) designed
  around the owl mark; fix the token leaks (podium metals, confetti colors,
  scrims, `theme-color` in app.html/manifest, favicon SVG colors); self-host
  fonts (drop the render-blocking Google Fonts `@import`).
- Replace the personal `fhun` marks with Matchowl logo/wordmark; regenerate
  all PWA icons, maskable set, notification icons, badge, email mark from
  `docs/matchowl-icon.svg`.
- Rewrite all UI copy: "WM Tips" → "Matchowl", World-Cup-specific strings →
  tournament-name interpolation (landing, login/register/join, banners,
  push/notify templates, DB-persisted system-email templates via a new
  migration, service-worker fallbacks, manifest name/description).
- New Landing page for Matchowl (multi-tournament pitch, owl identity).
- PWA caveats handled deliberately: keep manifest `id: "/"` (changing it
  makes browsers treat it as a different installed app); service-worker cache
  key rename busts old caches; new screenshots after the redesign settles.
- Per-route `<svelte:head>` titles + meta description + OG/Twitter cards.

### Phase 5 — Bots, docs, hardening

- `bots/`: stage list / bracket mirror / 8-of-12 logic read from the app's
  structure endpoint instead of local constants; prompts take tournament
  name; rating table becomes per-tournament data.
- README / DEPLOY / TRADEMARK sweep; delete the rebrand caveat from README.
- Tests: scoring engine against a non-WC structure (6-group Euro shape +
  a no-groups KO cup) to prove the generalization; migration test on a copy
  of real pb_data.

## Risks / notes

- **Index swaps on live data** — the `forecasts.user` unique index must be
  dropped before backfill completes; migrations run in one transaction per
  PB migration, test against a pb_data copy before deploying.
- **`matches.stage` Select→Text** relaxation is one-way; Go validation
  replaces schema validation — keep it strict.
- **Manifest `id`** must not change or installed PWAs orphan; name/icons can.
- **Old push subscriptions** keep working (VAPID keys untouched); only icons
  and copy change.
- **Bots are a separate module** and can lag one phase behind the app —
  their API surface only gains a tournament param.
