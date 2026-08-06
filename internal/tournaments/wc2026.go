package tournaments

// WC2026Slug is the slug of the migrated first tournament.
const WC2026Slug = "wc2026"

// WC2026Structure mirrors exactly the shape that was hardcoded across the
// codebase before the multi-tournament rework: 12 groups of 4 (3 games,
// top 2 advance, 8 best thirds via the FIFA Annex-C table) into a 32-team
// knockout with a third-place play-off. Used by the migration that creates
// the wc2026 record and by the compat boot-seed.
const WC2026Structure = `{
  "stages": [
    {"code": "group", "name": "Group stage", "kind": "group"},
    {"code": "R32",   "name": "Round of 32", "kind": "knockout"},
    {"code": "R16",   "name": "Round of 16", "kind": "knockout"},
    {"code": "QF",    "name": "Quarter-finals", "kind": "knockout"},
    {"code": "SF",    "name": "Semi-finals", "kind": "knockout"},
    {"code": "3RD",   "name": "Third-place play-off", "kind": "knockout", "consolation": true},
    {"code": "FINAL", "name": "Final", "kind": "knockout"}
  ],
  "groupSize": 4,
  "gamesPerTeam": 3,
  "directQualifiers": 2,
  "extraQualifiers": {"fromPosition": 3, "count": 8, "tableKey": "wc2026"},
  "pointsWin": 3,
  "pointsDraw": 1
}`

// WC2026Sync mirrors the previously hardcoded results-sync sources:
// API-Football league 1 season 2026, openfootball worldcup.json as the
// key-free fallback ("auto" picks between them like pickProvider always has).
const WC2026Sync = `{
  "provider": "auto",
  "apiFootballLeague": 1,
  "season": 2026,
  "openfootballURL": "https://raw.githubusercontent.com/openfootball/worldcup.json/master/2026/worldcup.json"
}`
