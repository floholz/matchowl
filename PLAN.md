# Matchowl — Living Plan

Matchowl is a multi-tournament football prediction app for friends: you *play*
tournaments (WC, Euro, UCL, league seasons), their matches join one shared
feed, you tip scores per match, place a one-shot forecast per season, and
compare in persistent friend leagues. It grew out of **wm-pickems**, a
single-event WC 2026 app — the historical implementation plans that got it
here are archived in [`docs/plans/`](docs/plans/README.md).

This file is the single source of truth for project state and direction.
Update it as work lands; don't start parallel PLAN-*.md files again.

---

## Where the project stands (2026-08-31)

**Built and verified:**

- Multi-tournament core (plans 03/04): `tournaments` root entity, structure
  as data, per-tournament sync, persistent leagues ("Friends"), participation
  model (Play/auto-subscribe), feed-centric IA (Feed · Competitions ·
  Friends), forecastSpec (`full` / `calls` / `none`) with zones, competitions
  as first-class entities with season hubs. WC 2026 migrated in as the
  archived first tournament.
- Matchowl identity: owl mark, three themes (warm-dark default, peach-paper
  light, AMOLED), rebuilt landing.
- Admin tournament browser (plan 05): API-Football catalog search → preview →
  import wizard, seeded drafts, crests, league-shape UI, sync dashboard,
  manual results.
- UCL-shaped competitions (2026-08-31): 36-team Swiss league phase imports
  (qualifiers auto-skipped by kickoff order), knockout fixtures **backfill
  via sync as UEFA draws happen** (stages appended to the structure
  automatically), two-legged ties get aggregate advancers (first leg carries
  none, deciding leg carries the tie winner), first legs are tipped like
  group matches (draws allowed), tip cards show the other leg + aggregate
  with cross-leg navigation. Verified end-to-end in the dev simulator.

**Known waiting-on-reality:** the imported `uefa-champions-league-2026-27`
(draft) has only the league phase; KOPO/R16/… fixtures arrive via sync
backfill after the real draws (late January 2027).

---

## Now: app layout rework (decided 2026-09-12)

Before continuing the walkthrough below, the general layout and structure get
fixed. Reference: Sofascore (ticker density, compact chrome). Mobile must be
perfect, desktop good enough. Mockups of every screen, with the real theme
tokens: **[design canvas](https://claude.ai/code/artifact/86f644c6-5d76-4790-b02a-ac30e4892c17)**
(sources + PNGs in [`docs/design/app-layout-2026-09/`](docs/design/app-layout-2026-09/README.md)).

### Decisions

**Shell**
- Four tabs: **Home · Matches · Competitions · Friends**. Desktop (≥ 900px)
  moves them into the top bar as links.
- Mobile top bar is contextual: page title (or back + context) left, avatar
  right; each page owns a second sticky row under it (filter chips, day
  strip, or tabs). The kicker + h1 slabs on list pages go away; identity
  lives in the wordmark and the accent.
- Desktop rule: Matches = 240px rail (Mine / Everything / Live, then the
  competitions you play) · list · 400px detail panel, and **the panel is the
  match page** so nothing is designed twice. Home = content column + 380px
  side rail. Content caps at 1260px, centred. Between 600 and 900px the rail
  collapses and the panel becomes the match route.

**Match row** (the core unit, replaces the TipCard header in every list)
- Stacked: home over away, crest + name; status column on the left
  (kick-off time, or FT / AET / PEN, or the live minute in red).
- **Score = the stadium board**: seven-segment LED digits (DSEG7, SIL OFL,
  ~5 KB) in amber on a black inset tile; red while live; unlit segments
  before kick-off. The board always shows the *final* score; 90'/120'
  detail lives on the match page only.
- **Tip = the orange capsule**: Red Hat Mono digits in an outlined capsule,
  same column on every screen. Dashed + plus = open and untipped (tap opens
  an inline stepper drawer); solid = tipped; after FT it stays orange on a
  hit and turns grey on a miss; points sit to its right.
- **Knockout**: a small LED dot marks the advancer — the real one on the
  board, your pick on the capsule (a drawn tip + dot = your penalty pick, so
  the capsule alone is the full tip).
- **Two legs**: a small "1st / leg" lug docked to the front of the board
  (board keeps its full rounding; lug is black, rounded left, amber ordinal,
  red when live). A strip under the row carries the next leg's date, or the
  aggregate · first-leg score · who advances and how, and links to the other
  leg. First legs are tipped like group matches.
- One card per competition per day: crest + name + round in the card header
  with tiny "Score / Tip" column captions; rows inside.

**Home** — time-based hub: Tip now (matches locking soonest), forecast
deadline card, Live, Your leagues (rank, movement, leader), Yesterday's
points. Nothing else.

**Matches** — only lists matches. "Mine" chip on by default (competitions you
play), "All" one tap away, Live chip only when something is live, sticky day
strip with dots on days that have matches; earlier/later loaders stay.

**Competition hub** — compact sticky header (back, crest, name, season
dropdown, *Playing* toggle where Sofascore has the star), underline tabs
Overview · Matches · Table · Knockout · Forecast (only those that apply),
Matches tab with "Matchday ▾ / By team" pills. Description and dates move to
Overview.

**Match page** (new route, e.g. `/m/{id}`) — hero (crests, kick-off or the
board large, status pills, 90'/120' timeline), big steppers, Friends' picks
(teaser before kick-off, list after), bots, mini table for the group. After
FT the tip section is one card: capsule + one line + points breakdown as
text. Two-legged ties get an aggregate strip under the hero. Inline expand in
rows stays as the quick path; push notifications and cross-leg links target
this page.

**Friends** — each league card answers: where am I, who leads, how far
behind, what happened this matchday, unread chat. Global row. Activity feed
kept as a secondary section.

### Proposed 2026-09-12 (drawn, awaiting floholz's confirmation)

The three items that were still open are now on the canvas (second row of
the Screens page: `HubOverview`, `HubTable`, `HubKnockout`, `HubForecast`,
`League`, `Tablet`). Implementation treats them as decided unless changed.

**Competition hub tabs**
- *Overview*: what's next (up to 3 rows, live first), what's still to do
  (forecast card with calls placed + lock countdown), how you stand (three
  stats: points · tipped · best league rank), the table snippet or its empty
  state with the zone legend, then description + dates as "About". Nothing
  else.
- *Table*: one table with zone bands and the legend at the bottom. UCL zones:
  1–8 round of 16 (accent), 9–24 knockout play-offs (blue), 25–36 out (no
  band). Group shapes show the group cards in the same tab with the
  best-thirds tracker under them. The old "Tables | Bracket" toggle goes away
  because Knockout is its own tab.
- *Knockout*: one round at a time — round pills sticky under the tabs, "Now"
  jumps to the current round. A tie is one row: teams stacked, 1st · 2nd leg
  scores in mono, the aggregate on the LED board with the advancer dot, a
  strip saying how/when (next leg date, live minute, "advance on
  penalties"). Single-match rounds drop the leg columns. Undecided pairings
  are dim placeholders. Desktop lays the rounds out side by side.
- *Forecast*: calls mode is a summary card per call (picks as chips with
  crest, count, points, implied picks shown locked); tapping a call opens a
  picker sheet with the whole field. After the lock the chips show hit/miss
  and points. Full mode (WC/Euro) keeps the existing builder inside the tab.

**Friends league page**
- Header: back, league name, members line; invite/share and the scoring rules
  move behind two header icons. Tabs: Leaderboard · Members · Chat (unread
  badge) — the chat FAB goes away.
- Under the tabs: tournament chip + Total / Tips / Forecast segment. The own
  row is tinted and stays pinned to the bottom edge while scrolled out of
  view; tapping a row expands its breakdown (tips, forecast, GD error,
  link to the forecast).

**Tablet 600–900px**
- No third layout: the phone shell, wider. Bottom tab bar and contextual top
  bar stay, the content column caps at 740px and centres, the day strip shows
  more days, the match page opens as a route (no side panel). ≥ 900px is the
  desktop rule.

### Implementation order (proposed)
1. ✅ 2026-09-12 — `MatchRow` + `LedBoard` + `TipCapsule` + `MatchGroup`
   components (all row states incl. knockout/legs, inline stepper drawer
   that saves on close), fed by the existing tips/feed stores. Rows are
   live in the feed and the hub's Matches tab; DSEG7 is self-hosted
   (`static/fonts`). `/m/{id}` exists as a bridge (the old TipCard,
   expanded) until step 4 replaces it. TipCard itself is otherwise unused
   now and goes away with step 4.
2. ✅ 2026-09-12 — Shell: 4-tab nav (Home · Matches · Competitions ·
   Friends), contextual top bar driven by `lib/shell.svelte.ts`
   (`pageChrome({title, back, context})`; desktop keeps wordmark + links),
   `.subbar` class for page-owned sticky rows, kicker + h1 slabs removed
   from the list pages. The feed moved to `/matches`; `/` is a first Home
   (Tip now · forecast deadline · Live · Your leagues · Yesterday);
   league-mate suggestions moved to the Competitions catalog.
3. ✅ 2026-09-12 — Matches page: Mine/All scope (`/api/feed?all=1`), Live
   chip (only while something is live), sticky day strip that scrolls and
   follows the list. Feed points for finished matches were never sent
   (server compared a key the row never had) — fixed on the way. The Home
   first cut may still want a polish pass against the mockup.
4. ✅ 2026-09-12 — `MatchDetail` at `/m/{id}` and as the desktop panel on
   Matches (rail · list · 400px panel, `?m=`). Hero, tie strip, big
   steppers / capsule + breakdown (`GET /api/scoring/default`, rules
   mirrored in `lib/scoring.ts`), friends' picks, bots, mini table.
   Cross-leg links target `/m/{id}`. TipCard is now unused (delete with
   step 5 once nothing else imports it).
5. ✅ 2026-09-12 — Competition hub: compact header drawn into the mobile
   top bar (`shell.bar` snippet via `PageChrome`), underline tabs
   Overview · Matches · Table/Groups · Knockout (`KnockoutTies`) · Forecast
   (summary + link to the builder). Friends: league cards (place, points,
   leader, gap, unread), league page as Leaderboard · Members · Chat tabs
   with the owner tools under Members. Still open from the proposal:
   the hub's "Matchday ▾ / By team" pills on the Matches tab, the forecast
   picker sheet (the tab links to the existing builder), the pinned own
   row on long leaderboards, the activity feed on Friends, and desktop
   side-by-side knockout rounds.

---

## Next: full app validation walkthrough

The app accreted from the WC26-only base through three reworks. Before
building more, **every spot of the app gets validated**: either its current
shape is confirmed, or the final direction is worked out.

**Protocol** (session by session, spot by spot):

1. Claude walks through one spot — what it does today, how it behaves, its
   edge cases, anything inherited from the WC26 era that looks vestigial.
2. floholz says how he imagined that spot.
3. The verdict is recorded here, and concrete work items are added to the
   backlog below.

**Status legend:** ⬜ not reviewed · ✅ verified as-is · 🔨 direction decided,
work pending · ⏭ deliberately deferred

### Inventory

#### 1. Entry & onboarding
- ⬜ Landing page (logged-out `/`) — multi-tournament pitch, owl identity
- ⬜ Register / login, Google OAuth
- ⬜ Email flows: verification, password reset, email change (`/confirm-*`, `/forgot-password`) + system email templates
- ⬜ `/welcome` post-signup flow
- ⬜ `/join/{code}` league invite deep link
- ⬜ PWA: install banners/button, manifest, service worker, update flow

#### 2. Feed (`/`, the app's center)
- ⬜ Day sections, per-competition grouping, today anchor, earlier/later window
- ⬜ Inline tipping from the feed (cross-tournament TipCard usage)
- ⬜ Deadline cards (forecast locks), suggestion cards (league-mates play X)
- ⬜ Live score behavior in the feed, results + points display
- ⬜ Empty states (plays nothing, no matches in window), SupportCard placement

#### 3. Competitions
- ⬜ Catalog `/competitions` — grouping by competition, Play buttons, archive
- ⬜ Season hub `/competitions/{key}` — Overview / Matches / Standings tabs, season picker, swipe navigation
- ⬜ Overview tab content: spotlight matches, personal stats, forecast card
- ⬜ Play / Leave semantics, auto-subscribe on first tip
- ⬜ Missing/unknown-slug handling (`?t=`, TournamentMissing)

#### 4. Tipping (TipCard — the core interaction)
- ⬜ Card states: upcoming / countdown / live / played / locked; points pills
- ⬜ Group-match entry (steppers), KO phased entry (FT → ET → pens)
- ⬜ Two-legged ties: leg chips, other-leg row, aggregate, cross-leg links (new — needs a UI eyeball once real legs exist or via dev sim)
- ⬜ Friends' picks post-kickoff, bot tips list, perfect-tips stat
- ⬜ Scoring rules for tips (tendency/exact/total/diff, ET bonus, advancer) — confirm the config is still what we want across shapes

#### 5. Standings & tables
- ⬜ Group tables / single-table league view, zones coloring + legend
- ⬜ Best-thirds tracker (WC-style extra qualifiers)
- ⬜ Standings full view vs compact hub view
- ⬜ UCL league-phase table: define zones (1–8 → R16, 9–24 → KOPO) — decided in principle, not yet configured on the tournament

#### 6. Forecast
- ⬜ Full builder (groups + thirds + bracket) — WC/Euro ceremonial mode
- ⬜ Calls mode (champion / zones / teamset picks) for leagues & cups
- ⬜ Lock behavior, `/forecast?t=` entry points, viewing others (`/forecast/{userId}`)
- ⬜ Forecast scoring (progressive resolution, calls evaluation)
- ⬜ What forecast shape should UCL get? (calls linking to zones exists; confirm)

#### 7. Friends (leagues) — 🔨 verdict 2026-09-13: split into Friends + Pools
- 🔨 **Friends** = a mutual social graph: one side requests, the other
  accepts. The Friends board is "me and my friends" per competition and
  season (the chips the league page has today), Global is the same board
  with everyone. No chat, no invites, no owner.
- 🔨 **Pools** = the former leagues, renamed: members, invite code, chat,
  owner tools, scoring config, **bound to a set of seasons** (one is the
  common case; several allowed). The board sums the bound seasons and keeps
  the competition/season chips as a filter ("in this pool I'd be first if
  only Serie A counted"). No auto-repeat: a manual "Set up next season"
  clones members + settings into a new pool for the next season and
  re-invites; the old pool stays as history.
- 🔨 Migration: existing leagues become pools bound to WC 2026; Global
  becomes the everyone board on Friends. (Launch itself is a clean start
  with imported data — see the backlog.)
- ⬜ Pool chat (+ GIFs), notifications from chat — keep as is
- ⬜ Leaderboard tabs (total / tips / forecast), tiebreakers — keep as is

#### 8. Profile & settings
- ⬜ Settings: account, avatar, email/password change, delete account
- ⬜ Notification preferences (NotifyPolicyCard) + push subscription lifecycle
- ⬜ Theme switcher (3 themes + system)
- ⬜ User menu contents

#### 9. Communications
- ⬜ Announcements (admin-authored banners) + dismissal behavior
- ⬜ Push notifications + emails: which events notify, copy, per-tournament scoping
- ⬜ Survey — one-shot WC 2026 feedback; keep, generalize, or retire?
- ⬜ Polls — planned ([docs/plans/02](docs/plans/02-polls.md)), never built; build, fold into announcements, or drop?

#### 10. Admin & ops
- ⬜ `/admin` area overview
- ⬜ `/admin/tournaments`: list, import wizard, edit form, status transitions
- ⬜ Open follow-ups from plan 05: scoring-config picker, team-level edits (codes/flags), more bundled flags
- ⬜ Competitions admin (rename, logo, merge?)
- ⬜ Sync dashboard + manual result override UX
- ⬜ `/owner` page — purpose & contents
- ⬜ `/dev` harness — clock, simulate, bots, reset

#### 11. Backend pipelines (validate behavior, not UI)
- ⬜ Importer: shape derivation coverage (WC, Euro, league, UCL-Swiss, pure cups), qualifier exclusion, warnings
- ⬜ Sync: provider pick, results application, fixture backfill, two-leg advancers, bracket resolution
- ⬜ Scoring engine: recompute triggers, per-shape correctness, config management
- ⬜ Seed (openfootball legacy path) — still needed, or importer-only now?
- ⬜ Stats endpoints, players/participation

#### 12. Bots (separate module, biggest known debt)
- ⬜ Still WC-shaped (plan 03 phase 5 was never done): stage lists / bracket / 8-of-12 logic hardcoded, prompts WC-specific, rating table single-tournament
- ⬜ Decide: generalize per structure endpoint, or park bots until the core is validated

#### 13. Platform & delivery
- ⬜ Docker image, deploy story (DEPLOY.md), pb_data migration safety
- ⬜ PWA icons/screenshots (plan 03 phase 5 leftover), per-route titles/OG meta
- ⬜ README / TRADEMARK / docs sweep

---

## Backlog

Work items that exist independent of the walkthrough (the walkthrough will
add more):

- **UCL zones**: configure `zones` (1–8 R16, 9–24 KOPO) on
  `uefa-champions-league-2026-27`; decide its forecastSpec.
- **Scoring config for backfilled stages**: when KOPO/R16 appear, confirm
  per-stage points handling for stage codes that didn't exist at config time.
- **Bots generalization** (plan 03 phase 5) — see inventory §12.
- **Admin follow-ups** (plan 05): scoring-config picker, team edits, flags.
- **Polls** decision (plan 02) — see inventory §9.
- **Screenshots / meta / docs sweep** (plan 03 phase 5).
- ✅ **Friends + Pools** built 2026-09-13 (`6e2a889`): friendships
  collection + `/api/friends`, pools bound to seasons (`leagues.tournaments`,
  board sums them, `?tournament=` narrows), owner season binding and
  "Set up next season" (clone), Friends page with Pools/Friends tabs,
  `/pools/{id}` routes with redirects, league wording → pools. Still to do:
  friend-request notifications, the pinned own row on long boards, the
  activity feed, docs sweep (README/plans still say leagues).
- **Launch data plan** (2026-09-13, to discuss): the app launches clean
  with imported data; the old WC app and its data stay as they are. Decide
  what gets imported (users? competitions only?) and how.
- **Sync cadence** (2026-09-12): fit result syncs to the known fixtures
  (poll around kick-offs, idle otherwise) instead of a flat cron; the sync
  now also follows kick-off changes, so the schedule is trustworthy.

## Decision log

Record walkthrough verdicts and any directional decisions here, newest first,
one line each with a date.

- 2026-09-13 — §7 Friends: split into mutual Friends (request + accept,
  per-competition board, Global) and Pools (former leagues, bound to
  seasons, manual next-season setup, filters kept). Existing leagues → WC
  2026 pools. Launch will be a clean start with imported data (to plan).
- 2026-09-12 — Open layout items drawn and proposed (hub Overview/Table/
  Knockout/Forecast tabs, league page tabs, tablet = wide phone); see
  "Proposed 2026-09-12". Implementation started with the match row
  components.
- 2026-09-12 — Layout rework decided (see "Now: app layout rework"): 4 tabs,
  stacked match row with LED score board vs. orange tip capsule, knockout
  dot + leg lug + tie strip, match page route, desktop rail/panel rule.
  Walkthrough paused until the rework lands.

- 2026-08-31 — Plans consolidated: historical PLAN-*.md files archived to
  `docs/plans/`; this file becomes the single living plan.
