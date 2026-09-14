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

## Next: head-to-head pools (decided 2026-09-14)

Season-long tip games bleed engagement: by matchday 12 the table stops
mattering for everyone outside the top three. The pool is the engine of
the app, and banter is its fuel, so pools get a **mode**. Head-to-head
gives every member a fresh, winnable duel each matchday, plus two moves
aimed at a person: the *save call* and the *ban*. No players, no
transfers, nothing beyond the tips people already place. Built before the
walkthrough continues, because it shapes the Friends section and the alpha
run is the best place to learn how it feels. Needs a running domestic
league season in the app as the test bed.

### Decisions

**Pool = one season + a mode**
- A pool binds exactly **one season** (`leagues.tournaments` goes from up to
  20 to 1). Bundesliga and UCL with the same friends are two pools. The
  lifecycle stays as built: upcoming → live → finished (read-only, chat
  grace 30 days) → "Set up next season" clones members, settings and mode.
- `mode` is `classic` (the points table, as today) or `h2h`. Chosen in the
  create sheet with a one-line explanation of each; the default comes from
  the season's shape: round-robin with no knockout after it → h2h, cups
  and tournaments (WC, Euro, UCL) → classic.
- The **first round** of a pool is the first matchday whose first kick-off
  is after the pool was created. Mode and h2h settings are editable until
  that kick-off, then locked (wrong mode after that: make a new pool).
  Tips are global per user and match, so a pool created mid-season has
  points at once; the h2h table simply starts at the first round while the
  Points tab shows the whole season.

**Rounds = matchdays**
- A round is the matchday key the hub already filters on
  (`stage|roundLabel`). Knockout stages of a league season (relegation
  play-off) are rounds too.
- A round **closes 24 h after the last scheduled kick-off** of its matches,
  as scheduled at that moment. A match that kicks off later (postponed,
  rescheduled) is ignored for the h2h and still counts for points.

**Pairing**
- Circle-method rotation over the roster as it stands at the round's first
  kick-off, indexed by round, so everyone can see in advance who they play
  and pairings repeat in the same order when the season is longer than the
  roster. Past rounds never change; late joiners enter at the next round
  with an empty record.
- Odd roster: one member plays the **Ghost**, the rounded mean of the other
  members' round scores. Every member plays every round.

**Round result and table**
- Round score = the member's tip points over the round's matches, after
  save calls and bans. Win 3, draw 1 each, loss 0.
- The h2h table is W-D-L + h2h points, tiebreak by season tip points, then
  the classic tiebreakers. In h2h pools it is the leaderboard; the points
  table is the second tab (Head-to-head · Points).

**Save calls** (`saveCalls` per pool, default 1 when a matchday has 6 or
fewer matches, else 2)
- A saved match counts **double** for its owner in that round. Placed
  until that match's kick-off, hidden from everyone until then, edits
  allowed like a tip. Saves apply in Ghost rounds too.

**Ban** (one per round, aimed at the current rival)
- Removes the banned match from the rival's round score. Placed until the
  match's kick-off, revealed to the rival at kick-off, not before. A ban on
  a save call cancels the double and the match counts normal (the save
  call saves it). A ban on a match the rival never tipped is wasted. No
  ban in a Ghost round. Season points are never touched by a ban.
- With two saves and one ban a member always keeps one guaranteed double.

**Surfaces**
- Home: "Matchday 5 · you vs Anna · 14–9 · 3 matches left" card (before
  the round: the rival and your open picks; after: the result).
- Pool page: the h2h table, and a **round view** with the two members side
  by side, per-match points, saves marked, bans revealed after kick-off.
  Members can browse past rounds.
- Match row / match page: a save marker on the capsule (star), the ban on
  the rival's picks after kick-off. Friends' picks show saves after
  kick-off.
- Pool chat: an automatic post when a round closes ("Anna beat Flo 14–9").
  Notifications: round result; "your rival banned X" at that kick-off.

**Data (sketch)**
- `leagues.mode` (select), `leagues.saveCalls` (number).
- `h2h_picks`: pool, user, round key, match, kind (`save` | `ban`). Per
  pool, since the rival and the allowance are the pool's.
- `h2h_rounds`: pool, round key, closesAt, pairings, results (written by
  the close job, so history is frozen). Round scores reuse the per-tip
  scoring; the multipliers live in the h2h layer, never in `tips`.

**Parked:** autopilot tips (auto 1-0 / 1-1 / 0-1 from the table for untipped
matches) — the flood of meaningless tips outweighs the inactivity fix for
now; see the backlog. Inactive members are free wins, as in fantasy.

### Implementation order

1. ✅ 2026-09-14 — Pools: one season per pool (migration 0042 keeps the
   running one), `mode` + `saveCalls` with defaults from the season's
   shape (`GET /api/pools/defaults`), `PoolSettings` component in the
   create sheet, the owner's Members tab and "Set up next season",
   `POST /api/pools/{id}/settings` refused once the first round kicked
   off (`pools.LockAt`), rounds helper in `tournaments/rounds.go`
   (ordered by round number, then median kick-off).
2. ✅ 2026-09-14 — Core h2h (`internal/h2h`): rounds open at their first
   kick-off with the pairings frozen from the roster (circle rotation by
   the pool's round ordinal, Ghost = rounded mean of the others), close
   24 h after the last scheduled kick-off with the results frozen in
   `h2h_rounds` (migration 0043); scores come from `match_scores` under
   the pool's config, matches kicking off after the close time are
   ignored. Job: cron every 5 min, on every read, after the dev clock
   moves. `GET /api/pools/{id}/h2h` (table, rounds with provisional
   scores, headline round, next pairing) and `/h2h/round?key=` (per-match
   breakdown). `H2HBoard` on the pool page: duel card, W-D-L table (tip
   points as tiebreak), round browser; Head-to-head · Points switch. Dev
   bots now tip the pool's own season.
3. ✅ 2026-09-14 — Save calls + bans: `h2h_picks` (migration 0044),
   `POST/GET /api/pools/{id}/h2h/picks` (placed or moved until the match
   kicks off; the allowance, one ban per round, no ban against the Ghost;
   a ban can be re-pointed until its match kicks off), round scoring
   applies ×2 / ×0 / cancel per duel, picks revealed at kick-off (round
   views carry revealed save counts and the ban flag). UI: "Your calls"
   panel under the round browser for the open round and the next one
   (dashed chip), marks on the pairs. **2026-09-14 (floholz):** tip and
   picks must live in one place — the tip drawer (match row and match
   page) now carries the star and the ban per h2h pool of that season
   (`GET /api/h2h/match/{id}`; the pool name shows only when there are
   several), and the board's panel is a matchday checklist ("8 to tip",
   tip state per match, rows link to the match). Picks stay per pool.
4. ✅ 2026-09-14 — Surfaces: Home's pool line shows the duel for h2h
   pools (score, rival, matches in; or the next rival), the round
   browser on the pool page is the round view, a **system chat message**
   posts the matchday's results when a round closes (migration 0045:
   `pool_messages.system`, user optional; rendered as a note; the chat
   notifier skips it), notifications `h2h_round` (on close, per paired
   member) and `h2h_ban` (at kick-off, from the scheduler; dedup per
   pick) with templates and settings entries. Fixed on the way: the
   chat page and the pool page still subscribed to the old
   `league_messages` collection, so live chat updates were dead since the
   pools rename.
5. ✅ 2026-09-14 — Verified: La Liga 26/27 simulated to season end on a
   copy of the dev DB (pool of 5 + a late joiner at matchday 7): 38 rounds
   closed, each member ghosted once while the roster was odd, the late
   joiner has 32 played, matchday 5 counted 9 of 10 (its postponed match
   ignored), 34 chat posts and result notifications, the ban notification
   at kick-off, ban-on-save scoring (×2 / ×0 / cancel) checked on the
   round detail. Found and fixed on the way: the close job raced the
   simulator's clock jump (the job is now serialised and held during
   `/api/dev/advance`), and a round now waits up to 12 h past its close
   time for a match that kicked off but has no result yet (late sync).
   **Next:** hand it to the alpha pools — a fresh h2h pool on a running
   league — and watch the first real round close.

---

## Next: full app validation walkthrough

The app accreted from the WC26-only base through three reworks. Before
building more, **every spot of the app gets validated**: either its current
shape is confirmed, or the final direction is worked out.

**Refocused 2026-09-13:** the layout rework is considered done, so the
walkthrough now hunts **migration artifacts** first (wording, routes, data
and behaviour inherited from the WC26 era, like the leaderboard key that
still read `league` after the rename). Each spot also gets a quick layout
confirmation in all sizes (phone, tablet, desktop) and all three colour ways.

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
- 🔨 Landing page (logged-out `/`) — moves out to the static site at
  matchowl.app (see backlog); signed-out `/` becomes login + site link
- ✅ 2026-09-13 built (batch after the §1 walk): `AuthShell` front door
  (owl mark, new tagline) on sign-in, register, forgot, the confirm pages
  and the invite; `GoogleButton` on both pages with self-hosted Roboto;
  terms checkbox on register + `/accept-terms` interstitial (migration
  0040: `termsAcceptedAt`, `termsVersion`, `lang`; `lib/legal.ts` holds
  the version); `/legal/{terms,privacy,imprint}` drafts; `/help` replaces
  `/welcome` and the Landing component is gone (signed-out `/` → login);
  verification tiers server-side (`users.RequireVerified` on friends,
  pools and chat; Global pool = verified only; nightly purge after 180
  days) with a verify gate on the Friends page; app name + `APP_URL`
  applied at boot; "update ready" toast; join page states. Open for the
  release pass: the final legal wording, PWA screenshots.
  **Legal identity (decided 2026-09-13):** no imprint page and no street
  address, ever. `/legal/about` is the Austrian small media-law disclosure
  for a private, non-commercial site: name (`OPERATOR_NAME` env, kept out
  of the public repo) + "Vienna, Austria" + contact email, and it names the
  GDPR controller. Ko-fi stays; ads may come later, at which point the
  e-commerce rules get re-checked.
- 🔨 Register / login, Google OAuth — verdict 2026-09-13: taglines to fit
  the multi-competition scope; Google button on register too (self-host
  Roboto Medium, keep Google's branding rules); both pages carry the owl
  mark and become the app's front door; the "?" help link gets a real
  in-app help / how-to page instead of the landing copy. **Verification
  tiers:** anyone may register with any email; unverified accounts can tip,
  play competitions and forecast, get **no mail at all** and **no social
  features** (friends, pools, chat, people search, the everyone board)
  until verified; Google sign-in counts as verified (PocketBase marks it).
  Enforced server-side, with a banner + resend in the app. **Legal at
  register:** terms + privacy acceptance (checkbox, stored with timestamp
  and version; Google sign-ups accept on first login), imprint/privacy/terms
  reachable signed-out (on the site, linked from the app). Unverified
  accounts older than 180 days get deleted.
- 🔨 Email flows: verification, password reset, email change (`/confirm-*`,
  `/forgot-password`) + system email templates — verdict 2026-09-13: flows
  work; the confirm pages are well themed but bare (same front-door
  treatment as login: owl mark, wordmark). Subjects say "Acme" and links
  point at localhost:8090 because the PocketBase app name / app URL settings
  still hold the defaults — the squashed init sets the name to Matchowl and
  the URL from env at boot. Dev: the log mail sink now prints the links.
- 🔨 `/welcome` post-signup flow — it is the Landing component again; goes
  with the landing, replaced by the in-app help page (see register/login)
- 🔨 **Light theme** (app-wide, 2026-09-13): white-ish ground, peach
  surfaces — swapped in `theme.css`, to be confirmed screen by screen
- 🔨 `/join/{code}` pool invite deep link — verdict 2026-09-13: the signed-in
  path swallows the server's reason and shows "Couldn't join. Please try
  again" for everything (Bürocup is a finished pool → 409). The page must
  say the real reason: pool finished (already at preview, before asking a
  stranger to sign up), already a member (just open it), unverified
  (verify first, once the tiers land), invalid code. Same WC26 tagline and
  bare front door as login.
- 🔨 PWA: install banners/button, manifest, service worker, update flow —
  verdict 2026-09-13: install works (Pixel + desktop). Screenshots are
  WC26-era and get redone once the layout is confirmed; manifest
  description reworded (German variant with i18n). Black status bar in
  light mode is fine. Update flow: new build activates on the next launch
  (skipWaiting + claim already), plus a small "update ready · reload" toast
  for open sessions; big changes keep going out via announcements.

#### 2. Matches (`/matches`, the app's center) — ✅ verified 2026-09-13
- ✅ Day sections, per-competition grouping, today anchor, infinite scroll
  both ways (30-day cap each side). Fixed on the walk: landing on today
  after navigation and on a nav re-tap, the sticky-chrome offset, the
  strip following loaded-in days, the loader race, the upward load
  keeping the page still.
- ✅ Inline tipping via the match row (TipCard deleted — it was dead code)
- ✅ Deadline cards come with the feed and Home shows them; suggestions
  live on Competitions
- ⏭ Live score behaviour — check with the dev simulator when a live
  match is around (§4)
- ⬜ Empty states (plays nothing, no matches in window) — not yet eyeballed
- 🔨 Light theme: brown text, burnt-orange board tile (`--board-*` tokens)

#### 3. Competitions
- ✅ Catalog `/competitions` — pass done 2026-09-13 (`e8185d3`): search
  field, suggestions capped to two (expand for the rest), Play from the
  list, sections (playing · available · finished), Play toggles refresh the
  feed. Older dev competitions lacked country/short name (pre-importer
  records; filled by hand, prod imports fresh). Found: league badges never fetched
  for the leagues because the leagues→pools rename had rewritten the
  provider's badge URL (`football/pools/`) — fixed, missing badges are
  backfilled at boot. La Liga's forecast call said "Religation" (data;
  fixed in the dev DB, the prod import must not repeat it).
- ✅ Season hub `/competitions/{key}` — verified 2026-09-13 as rebuilt
  (compact header, underline tabs, season picker)
- ✅ Overview tab content — verified 2026-09-13
- ⬜ Play / Leave semantics, auto-subscribe on first tip
- ⬜ Missing/unknown-slug handling (`?t=`, TournamentMissing)

#### 4. Tipping (the match row + match page) — walked 2026-09-13
- ✅ Row states: upcoming / live / played / locked; points after FT. "Live"
  without the minute for now — the minute comes with the sync rework.
- ✅ Group-match entry (steppers, inline drawer + match page)
- ⏭ KO phased entry (FT → ET → pens) — no upcoming knockout fixture until
  the CL draw; check then (dev sim otherwise)
- ✅ Two-legged ties: tie strip, aggregate, cross-leg link (CL 2025-26)
- ✅ Friends' picks, bots, mini table on the match page
- 🔨 **Scoring** — verdict 2026-09-13, backed by the WC26 analysis
  (floholz.com/wm2026): the total-goals point was a coin flip (34% of
  those points went to tips with the wrong winner) → **dropped**; a
  perfect tip must stand out → **exact 1 → 3**. New default: result 3 ·
  goal difference +1 · exact +3 (perfect 7, right result + GD 4, result
  only 3). Applied to the dev default config; the seeded default follows.
  **Scoring v2** (backlog): lean into configurability — the tiebreaker
  order the leaderboard actually honours (today it ignores the config),
  per-stage multipliers (knockout, final), an extra-time / penalties
  bonus, additive vs tiered mode, exact-score rarity weighting (article:
  base / ×2 under 15% of the field / ×3 under 5%), an admin editor for
  named presets and a preset picker for pool owners (shown behind the
  pool's rules icon).

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
  owner tools, scoring config, **bound to one season** (2026-09-14: was a
  set of up to 20; one season per pool since the h2h decision, a second
  competition is a second pool) and a **mode** (classic or head-to-head,
  see "Next: head-to-head pools"). No auto-repeat: a manual "Set up next season"
  clones members + settings into a new pool for the next season and
  re-invites; the old pool stays as history.
- 🔨 Migration: existing leagues become pools bound to WC 2026; Global
  becomes the everyone board on Friends. (Launch itself is a clean start
  with imported data — see the backlog.)
- 🔨 Design pass 2026-09-13 (canvas boards `Pools` … `PoolMembers`):
  Friends tab = search on top, requests pinned with Accept / Decline, the
  board (medals for the top three, own row tinted), your friends. Pools tab
  = cards with seasons line + place · points · leader gap + a one-line
  "what happened"; Start / Join in a bottom action bar opening a sheet;
  creating ends on the invite code + share link. Pool page = back · name ·
  members + seasons · Invite; board chips All seasons / one season; own row
  pinned while scrolled out. Members = invite code first, members with
  roles and join dates, seasons, Next season.
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
- ⬜ `/dev` harness — clock, simulate, bots, reset. Found 2026-09-13: the
  harness's bots were created without `role=bot`, so they counted as
  humans (chat digest mailed them, people search found them) — fixed.

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

## Releases

- **v1.0.0-alpha.2** (2026-09-13): one-time registration links — an admin
  mints them under Admin → Registration links and each creates exactly one
  account while sign-up stays closed. Migration 0041 (`signup_links`).
- **v1.0.0-alpha.1** (2026-09-13): first build for private testing, live on
  the VPS at play.matchowl.app the same day (Traefik in front, own compose
  directory and volume next to the old WC app). Release flow: tag on `main` → CI pushes
  `ghcr.io/floholz/matchowl:<version>` and publishes the GitHub release
  (pre-release for alpha/beta/rc) from `CHANGELOG.md`; the host runs the
  image via compose with `MATCHOWL_VERSION`. `REGISTRATION_OPEN=0` keeps
  sign-up closed (testers get one-time registration links minted under
  Admin → Registration links); the version shows in the Home footer. The walkthrough
  paused after §5 for this release and resumes at §6 (Forecast).

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
  `/pools/{id}` routes with redirects, league wording → pools. Pool invites
  for friends (`pool_invites`, event `pool_invite`, pinned on the Pools tab)
  landed `9fe700e`. Still to do: friend-request notifications, the
  activity feed, docs sweep (README/plans still say leagues).
- **Release data plan** (decided 2026-09-13): v1 starts from an empty
  database. Before v1 the 39 migrations get **squashed into one initial
  setup** (schema, rules, mail templates, seeded config rows — the final
  state, not the history; verify by diffing a fresh database's collections
  against one migrated the long way). Once live, a **one-off import** copies
  selected data sets from the WC 2026 run, then a plain backup covers the
  rest. Nothing repeatable is needed beyond the importer itself.
  - *No importer code:* when it is time, read the latest WC 2026 backup and
    insert the chosen records by hand through the PocketBase API (users;
    tips, forecasts, leagues as pools bound to WC 2026). One off, done.
  - *Claimable accounts (proposed):* imported users are created with their
    email, name, avatar and role, `verified=false`, a random password, and a
    `claimed=false` marker. Claiming is the two paths that already exist:
    **Google sign-in** (PocketBase 0.38 links an OAuth login to the record
    with the same email) and **password reset** (mails the address). An auth
    hook flips `claimed` on the first successful login and routes to
    `/welcome`. Until then the account is dormant: no notification mail or
    push, hidden from people search and friend suggestions, but still on the
    imported pool boards with a "not back yet" marker. Login page gets a
    "Played WC 2026?" hint; optionally one owner-triggered invitation mail
    batch to the dormant addresses.
- **Domains + marketing site** (decided 2026-09-13): the app moves to
  `play.matchowl.app` (PWA scope `/`, mail links and OAuth redirect point
  there; signed-out `/` becomes login with a link to the site; the SPA's
  `Landing` component goes). `matchowl.app` is a static bilingual marketing
  site (Astro, in `site/`, content collections for dated spotlights such as
  a CL final), English at the root, German under `/de`; `matchowl.de`
  redirects to the German pages, `play.matchowl.de` to
  `play.matchowl.app/?lang=de`. The site follows the device theme; no
  theme or session sync across origins. Build after the walkthrough and the
  i18n pass. matchowl.com is not ours (premium resale) and is out.
- **Languages before launch** (decided 2026-09-13): English default, German
  second. Language is a user setting, never a route prefix: saved setting →
  one-time `?lang=` hint from the site → browser language → English. UI
  strings via Paraglide (inlang) with Intl for dates and numbers; a `lang`
  field on users picks the notification and system-mail templates per
  language (system-mail hook chooses the template). Admin-authored content
  (descriptions, announcements) stays single-language. One extraction pass
  after the walkthrough, when the wording is settled.
- **Autopilot tips** (parked 2026-09-14): opt-in auto tips at kick-off for
  untipped matches (1-1 within 20 % of the table, else 1-0 / 0-1 for the
  higher team; last season's table before matchday 3, else 1-1), flagged
  as auto, full points, no save calls or bans. Parked: it floods the game
  with meaningless tips; revisit if inactivity becomes a real problem.
- **Sync cadence** (2026-09-12): fit result syncs to the known fixtures
  (poll around kick-offs, idle otherwise) instead of a flat cron; the sync
  now also follows kick-off changes, so the schedule is trustworthy.

## Decision log

Record walkthrough verdicts and any directional decisions here, newest first,
one line each with a date.

- 2026-09-14 — Head-to-head pools: pools bind one season and get a mode
  (classic / h2h, default from the season's shape, locked at the pool's
  first round); h2h = matchday duels by rotation with a Ghost for odd
  rosters, 3/1/0, season points as tiebreak, save calls (double) and one
  ban per round aimed at the rival; rounds close 24 h after the last
  scheduled kick-off. Built next, before the walkthrough resumes.
  Autopilot tips parked.
- 2026-09-13 — §1 register/login: verification tiers (unverified = play
  only, no mail, no social), terms + privacy acceptance at register, Google
  on both pages, owl mark, real help page replaces `/welcome`.
- 2026-09-13 — Domains: app at play.matchowl.app, static bilingual site at
  matchowl.app (.de redirects to the German pages), no language in app
  URLs; i18n (en default, de) via Paraglide before launch, after the
  walkthrough. §1 landing page verdict: moves out to the site.
- 2026-09-13 — Release data: clean start, migrations squashed to one initial
  setup before v1, one-off import of selected WC 2026 data after go-live,
  imported accounts dormant until claimed (Google or password reset).
  Walkthrough resumes with migration artifacts as the main target; layout
  gets confirmed per spot in all sizes and colour ways.
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
