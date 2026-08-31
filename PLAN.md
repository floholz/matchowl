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

#### 7. Friends (leagues)
- ⬜ `/friends` list, create/join with invite codes, Global league semantics
- ⬜ League page `/friends/{id}`: leaderboard (tournament selector), members, roles
- ⬜ League chat (+ GIFs), notifications from chat
- ⬜ Leaderboard tabs (overall / tips / forecast), tiebreakers

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

## Decision log

Record walkthrough verdicts and any directional decisions here, newest first,
one line each with a date.

- 2026-08-31 — Plans consolidated: historical PLAN-*.md files archived to
  `docs/plans/`; this file becomes the single living plan.
