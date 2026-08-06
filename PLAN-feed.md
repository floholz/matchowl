# Feed, Navigation & Forecast Rework — Plan

> Status: planned, not yet implemented. Written 2026-08-06 after the
> multi-tournament rework (PLAN-rework.md, phases 0–4 shipped). Decisions
> below were made with floholz in conversation.

## The mental-model shift

Tournaments are **not workspaces you switch between** — they are **sources
you subscribe to**. Matchowl becomes a ticker app: you choose to *play* a
tournament, its matches join your feed, and the daily loop is simply "open
the app, here are today's matches across everything I play, pick my
scores". A WC quarter-final sits next to a Serie A matchday in one scroll.

## Decisions

1. **Feed-centric IA, three destinations**: Feed · Tournaments · Leagues
   (+ user menu). Mobile keeps the bottom tab bar (3 tabs now). Desktop
   drops the side rail for a **top navbar** (logo · tabs · user) — scales
   across window sizes and shares one layout system with mobile.
2. **Playing a tournament (hybrid subscription)**: an explicit **Play**
   button on the tournament card subscribes you; submitting your first
   tip/forecast in a tournament auto-subscribes; league activity only ever
   *suggests* ("3 league-mates play Euro 2028"), never auto-joins.
3. **Forecast becomes per-tournament config.** The one-shot pre-tournament
   Hail-Mary stays sacred, but its *shape* is an admin decision per
   tournament:
   - **`full`** — today's ceremonial builder (groups + extra qualifiers +
     bracket). Feels special for a WC or Euro.
   - **`calls`** — a small list of admin-defined headline calls. For a
     league season: champion, UEFA-competition spots, relegation,
     relegation play-off. For a cup it can be "champion + final four".
     Five-ish decisions, everyone gets it.
   Rationale: with several simultaneous tournaments, full forecasts
   everywhere would be pick-fatigue; and for a Serie A season nobody cares
   who lands 10th.
4. **League seasons are tournaments too** (e.g. `seriea-2026-27`): one
   group-kind stage with ~20 teams and 38 games each, no knockout stages,
   forecast mode `calls`. The structure model supports this shape; caps
   need raising (see below).

## Data model

- **`tournament_players`** (new collection): `tournament` (relation,
  cascade), `user` (relation, cascade), `source` select `manual|auto`,
  `joinedAt`. Unique on `(tournament, user)`. Rules: list/view own +
  league-mates; create/delete own (Play/Leave). Migration: every existing
  user is backfilled as a wc2026 player (`source: auto`).
- **`tournaments.forecastSpec`** (new JSON field):
  ```jsonc
  {"mode": "full"}                            // wc2026 backfill
  {"mode": "calls", "calls": [
    {"key": "champion",  "name": "Champion",              "type": "team",    "points": 13},
    {"key": "ucl",       "name": "Champions League spots","type": "teamset", "count": 4, "points": 3},
    {"key": "relegated", "name": "Relegated",             "type": "teamset", "count": 3, "points": 4}
  ]}
  {"mode": "none"}                            // tips-only tournaments
  ```
  `team` = exact single pick (champion = final position 1 for league
  shapes, knockout winner for cups). `teamset` = unordered membership
  against a position range; ranges come from **zones** (next point).
- **`structure.zones`** (optional, group/league shapes): named position
  ranges used for table display *and* as `teamset` targets:
  ```jsonc
  "zones": [
    {"key": "ucl", "name": "Champions League", "from": 1, "to": 4},
    {"key": "rel-po", "name": "Relegation play-off", "from": 16, "to": 17},
    {"key": "rel", "name": "Relegated", "from": 18, "to": 20}
  ]
  ```
- **Structure caps**: `groupSize` limit 2..8 → 2..24;
  `tournament_groups.teams` relation MaxSelect 8 → 24; `gamesPerTeam` up
  to double round-robin (38+). A league season is one group with no
  `directQualifiers`/`extraQualifiers` semantics (zones replace them).
- **`forecasts.calls`** (new JSON field): `{callKey: teamId | [teamIds]}`
  for `calls`-mode tournaments; the existing three fields keep serving
  `full` mode. Scoring: exact-pick points for `team`; per-correct-member
  points for `teamset`, evaluated from final standings/knockout results
  when the tournament finishes (progressively where determinable).

## Competitions & seasons

A season is just a tournament (`seriea-2026-27`); competitions are a
first-class entity linking seasons (decided: full collection, not a field —
"not that much bigger and gives us opportunities"):

- **`competitions` collection**: `key` (slug, unique), `name`,
  `shortName`, `country`, `teamKind` select `national|club` (drives
  flag-vs-crest rendering), `logo` (file upload), `apiFootballLeague`
  (the provider's stable league id — the season lives on the tournament).
  Public read; admin CRUD under `/api/admin/competitions`. wc2026's
  competition ("world-cup", FIFA World Cup, national, league 1) is
  inserted by migration.
- **`tournaments.competition`** relation (optional, backfilled for
  wc2026). The catalog groups by competition (current season card +
  archive underneath); re-play suggestions fire when a new season of a
  competition you played goes upcoming. A tournament's sync config may
  omit `apiFootballLeague` and inherit the competition's.
- **Clone-season admin action** `POST /api/admin/tournaments/{id}/clone`:
  copies structure, zones, forecastSpec, competition, and sync config with
  slug/extIdPrefix/season/dates from the request; lands as `draft`;
  fixtures seeded separately as usual.
- **`teams.clubKey`** (optional text, e.g. `juventus`): stable club
  identity across season rows — enables per-club crest assets (crests by
  clubKey instead of ISO flag) and cross-season history later. National
  teams keep using `fifaCode`/`iso2`.

## Screens

- **Feed `/`** — the app's center, replacing today's Home + Tips pages:
  - matches of your played tournaments, grouped by day, tip inline
    (TipCard survives), each row tagged with its competition;
  - yesterday's results with points earned;
  - deadline cards: "Euro 2028 Hail-Mary locks in 3 days" (links to the
    tournament's forecast), "kickoff in 2h, 4 tips missing";
  - suggestion cards (league-mates play X — Play / dismiss);
  - empty state: point to the Tournaments catalog.
- **Tournaments `/tournaments`** — the catalog: active/upcoming cards with
  Play buttons, archive below. Detail `/tournaments/{slug}`: overview
  (status, your standing, Play/Leave), schedule, bracket/tables (zones
  colour league tables), forecast (full builder or calls card), read-only
  for archived.
- **Leagues** — unchanged as the persistent social layer; leaderboard gets
  a tournament selector (already server-side); league page can show the
  suggestion source ("this league plays: …").
- Old routes `/tips`, `/forecast`, `/tournament` → redirects into the new
  IA (installed-PWA muscle memory + bookmarked links).

## Build phases

| # | Phase | Scope |
|---|-------|-------|
| 1 | Participation | `tournament_players` + migration backfill, Play/Leave API + auto-subscribe hooks, suggestions endpoint; `competition` field + backfill, clone-season admin action |
| 2 | Feed backend | `/api/feed`: day-grouped matches across played tournaments, deadlines, results+points; notify detectors read participation (only nudge players) |
| 3 | Feed + nav frontend | Feed page, 3-tab bar, desktop top navbar (rail removed), Tournaments catalog (grouped by competition) + detail, route redirects |
| 4 | Forecast spec | `forecastSpec`/zones/caps migrations, calls scoring in internal/scoring, calls UI (feed card + tournament page), wc2026 backfilled as `full` |
| 5 | League-season proof | seed a 20-team league-shape test tournament through the admin API; zones on tables; dev-sim a season |

## Notes / risks

- Feed queries stay cheap: matches filtered by the user's played
  tournament ids (a handful), day-windowed.
- Scoring for `calls` needs final-standings resolution for league shapes —
  `groupStandings` already computes full position maps; "final" = all
  `gamesPerTeam` played.
- Bots: only ever play tournaments they're subscribed to (aligns with the
  participation model).
- The full builder stays untouched for wc2026's archive view.
