# Admin Tournament Browser — Implementation Plan

> Status: v1 implemented 2026-08-15 (backend + `/admin/tournaments` UI;
> verified by importing WC 2022 and Bundesliga 2023/24 from the real API,
> activating, syncing results, and rendering tables + bracket). Decided
> with floholz: v1 = **manage tournaments in-app + import from
> API-Football**, on a dedicated `/admin/tournaments` route.
> Follow-ups: scoring-config picker (needs a list endpoint), team-level
> edits (codes/flags), more bundled flags, single-table heading ("Group A"
> for league seasons).

## Why

Since the multi-tournament rework, tournaments are records — but the only
way to add one is the admin API via curl (create → seed with openfootball
JSON). Admins should be able to browse a catalog (API-Football leagues +
seasons), pick one, and get a seeded draft tournament without leaving the
app; and manage the ones that exist (status, dates, structure, sync).

## Backend

### `internal/football` — catalog calls
- `Leagues(ctx, search)` → `GET /leagues?search=` (or `?id=`): id, name,
  type (League|Cup), logo, country {name, code}, seasons [{year, start,
  end, current}]. `Teams(ctx, league, season)` → `GET /teams`: id, name,
  code, country, national, logo. `Fixture` gains `HomeID/AwayID` and
  logos so the importer can identify teams without name matching.

### `internal/importer` (new)
- `GET /api/admin/football/leagues?search=` — proxied catalog search
  (needs `API_FOOTBALL_KEY`; 400 with a clear message otherwise).
- `GET /api/admin/football/preview?league=&season=` — fetches fixtures +
  teams (cached in-memory ~10 min per league/season to spare quota) and
  derives a proposal: teams, rounds, detected shape, `structure`,
  suggested slug/name/shortName/extIdPrefix/dates/sync. Never writes.
- `POST /api/admin/tournaments/import` — body = the (possibly edited)
  proposal + `competition?`; creates the tournament as **draft** through
  the same validation as `POST /api/admin/tournaments`, then seeds
  teams / groups / matches from the cached fixtures in one transaction.
  Refuses if the slug exists.

### Shape detection (round labels → stages)
- `Group X - n` → group stage with letters from the labels. **API-Football
  labels World Cup rounds `Group Stage - n` (no letter)**, so table-style
  rounds get their group membership from `/standings` (authoritative) or,
  when the season has no standings yet, from the connected components of
  the who-plays-whom graph (lettered A.. by first kickoff). One component
  → league season (letter `A`); several → WC/Euro groups.
- `groupSize` = teams per group; `gamesPerTeam` = matches per team;
  `directQualifiers` defaults 2 for multi-group shapes, 0 for a league;
  zones left for the admin.
- Knockout labels: `Round of 32/16` → R32/R16, `Quarter-finals` → QF,
  `Semi-finals` → SF, `3rd Place Final` → 3RD (consolation), `Final` →
  FINAL; anything else → knockout stage with a slug code, ordered by first
  kickoff. Nothing group-like → pure KO cup.
- Matches: `extId = <prefix>-AF-<fixtureId>`, `num` = 1..n in kickoff
  order, `roundLabel` = provider round; group matches carry
  `groupLetter` + team relations; KO matches carry team relations when the
  provider knows them, else `TBD` labels (resolved by sync later).
- Teams: `fifaCode` = provider `code` or 3 letters from the name (unique
  per tournament); national teams get `iso2` from a country-name map (49
  bundled flags; missing → code chip); clubs get `clubKey` = slugified
  name.
- Results are **not** applied on import — the regular sync does that once
  the tournament is active (SyncOnce matches on team-name pairs, which
  import guarantees since names come from the same provider).

### Small additions
- `payload.Competition` so the edit form can link a competition.
- `GET /api/admin/tournaments` returns `teams`/`matches` counts and
  `competition` id so the list can show "seeded / empty".

## Frontend

- `/admin/tournaments`: list of all tournaments (drafts included) with
  status pill, competition, dates, seeded counts; row actions: edit,
  status transitions, delete (draft only), sync now.
- **Add tournament** stepper: (1) search API-Football leagues → pick →
  season list; (2) preview: teams, rounds, detected shape, editable
  name/slug/prefix/dates + structure JSON; (3) import → draft created and
  seeded → opens the edit form. Also **Create manually** (blank form) and
  **Seed from JSON** (existing endpoint) for the no-key case.
- Edit form: name, shortName, slug, status, startsAt/endsAt, competition,
  structure / sync / forecastSpec JSON, scoring config.
- Link from the Admin area page.

## Verification
- Go tests for round-label → structure derivation on synthetic fixture
  sets (WC-style groups+KO, league season, pure KO).
- Manual: search "Bundesliga" → 2023 (free-plan-reachable) → preview →
  import → draft appears, teams/matches counts right, activate → sync
  fills results.
