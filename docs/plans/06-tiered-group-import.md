# 06 — Tiered group import (UEFA Nations League)

Written 2026-09-24. Status: **open** — the live tournament was repaired by
hand (see "What was done"); the importer fix is not applied yet.

## The issue

Importing the UEFA Nations League 2026/27 (API-Football league 5, season
2026) through the admin import wizard produced five nonsense groups
(`1`, `2`, `3`, `4`, `M`) of 11–14 teams each instead of the fourteen real
ones (A1–A4, B1–B4, C1–C4, D1, D2). The structure came out as
`groupSize: 14, gamesPerTeam: 6`, and every match carried one of the five
wrong `groupLetter`s.

Separately, the PocketBase admin UI seemed to show duplicated teams
(Germany twice, Paris Saint Germain three times). That is not a bug — see
"Duplicate teams" below.

## Findings

### What the provider returns

- **Fixtures** (156): round labels carry the *tier* but not the group:
  `League A - 1` … `League D - 6`.
- **Standings** (15 tables): names carry the *group* but not the tier.
  Leagues A, B and C each have tables named `Group 1` … `Group 4`,
  League D has `Group 1` and `Group 2`, and there is one extra table named
  `Ranking of third-placed teams`. Nothing in a standings table says which
  tier it belongs to.
- **Teams** (54): no duplicates, ids match the fixtures exactly.

### What our importer did with it

`internal/importer/derive.go` classifies `League A - 1` as a *table* round
(no group letter in the label), so group membership comes from the
standings via `groupsFromStandings` / `groupLetter`. That code assumes table
names are unique per season:

1. `groupLetter("Group 1", i)` reduces the name to its suffix, so the
   `Group 1` tables of leagues A, B, C and D all became group `1` (14
   teams), and likewise `2`, `3`, `4`.
2. `Ranking of third-placed teams` has no `Group` suffix, so it fell back
   to the i-th letter — it was the 13th table, hence `M`. It was processed
   last and overwrote the real group of Luxembourg, Cyprus, Kazakhstan and
   Finland.
3. `GroupSize` is the largest group found → 14.

Everything else (teams, codes, flags, 156 matches, kickoffs) imported
correctly.

### The correct groups are derivable from data we already fetch

Tier from the round label head (`League A` → `A`) + suffix from the
standings table (`Group 1` → `1`) = `A1`. Skip standings tables that are
not groups when real group tables exist. Run against the real payloads this
yields exactly:

| Group | Teams |
|---|---|
| A1 | Belgium, France, Italy, Türkiye |
| A2 | Germany, Greece, Netherlands, Serbia |
| A3 | Croatia, Czechia, England, Spain |
| A4 | Denmark, Norway, Portugal, Wales |
| B1 | FYR Macedonia, Scotland, Slovenia, Switzerland |
| B2 | Georgia, Hungary, Northern Ireland, Ukraine |
| B3 | Austria, Israel, Kosovo, Rep. Of Ireland |
| B4 | Bosnia & Herzegovina, Poland, Romania, Sweden |
| C1 | Albania, Belarus, Finland, San Marino |
| C2 | Armenia, Cyprus, Latvia, Montenegro |
| C3 | Faroe Islands, Kazakhstan, Moldova, Slovakia |
| C4 | Bulgaria, Estonia, Iceland, Luxembourg |
| D1 | Andorra, Gibraltar, Malta |
| D2 | Azerbaijan, Liechtenstein, Lithuania |

`groupSize: 4, gamesPerTeam: 6`, shape `groups`, single stage. The `letter`
field is `Max: 2`, so two-character letters fit; the standings UI sorts
groups by string, which orders A1 … D2 correctly.

### Duplicate teams

Teams are one row **per tournament** by design (`teams.tournament`), so a
nation or club exists once per season it plays in: Germany is in WC 2026 and
in the Nations League; Paris Saint Germain is in Ligue 1 26/27, UCL 25/26
and UCL 26/27. 53 of 217 distinct names appear in more than one tournament.
The PocketBase relation picker lists all rows across all tournaments, which
is where the "duplicates" appear. No tournament has a duplicate name inside
it, and the provider team list has none either.

Risk when editing groups by hand in PocketBase: picking the WC "Germany" row
for a Nations League group silently breaks standings. Prefer editing groups
through an in-app admin UI (not built yet — see plan 05 follow-ups).

## Plan

### 1. Importer fix (`internal/importer/derive.go`)

Change `groupsFromStandings(standings, fixtures)`:

- Build `tierOf[teamID]` from the head of each table round label (the part
  before ` - N`, via `tableRoundRe`).
- Count standings table names. When a name repeats, prefix the group suffix
  with the tier letter: last word of the head if ≤ 2 chars, else its
  initial (`League A` → `A`, `League A` + `1` → `A1`).
- When at least one table name contains "group", skip tables that don't
  (`Ranking of third-placed teams`). Single-table leagues (`Premier League`)
  and the Swiss-model UCL (`League Stage`) are unaffected because they have
  no group tables to trigger the skip and no repeated names.

Add a regression test (Nations League shape: two tiers × two groups,
repeated `Group 1`/`Group 2` names, one ranking table) asserting the groups
`A1, A2, B1, B2` and `groupSize 4`.

A ready patch with exactly this change and test is at
`docs/plans/06-tiered-group-import.patch` (applies cleanly on
`e46752d`; `go vet` and `go test ./internal/importer/` pass).

### 2. Guard against silent merges

In `Derive`, warn when a standings-derived group is larger than the
largest round-labelled or component-derived group would allow — e.g. any
group with more than 8 teams in a non-single-table shape. Cheap early
signal in the preview for the next odd competition.

### 3. Later

- Nations League play-offs, promotion/relegation play-offs (March 2027) are
  not published yet; when they appear, sync backfills them under their own
  round labels. Check `classifyRound` handles "Play-offs - Quarter-finals"
  style labels as knockout and that they are not excluded as qualifiers
  (they come *after* the group stage, so `QualifierRounds` keeps them).
- Forecast spec for a tiered event (promotion/relegation calls per tier).

## What was done (2026-09-24)

Live DB (`play.matchowl.app`), tournament `uefa-nations-league-2026`
(`p2ybkeiyglv9qvm`, draft, no tips/forecasts yet):

- Deleted groups `1`, `2`, `3`, `4`, `M`.
- Kept the hand-made `A1`–`A4` (they matched the provider exactly).
- Created `B1`–`B4`, `C1`–`C4`, `D1`, `D2`.
- Rewrote `matches.groupLetter` for all 156 matches from the home team's
  group.
- Set structure `groupSize: 4`, `gamesPerTeam: 6`.
