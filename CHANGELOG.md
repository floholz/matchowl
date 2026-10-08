# Changelog

All notable changes to Matchowl. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[SemVer](https://semver.org/). Pre-releases (`-alpha.N`, `-beta.N`, `-rc.N`) are
private or limited test runs and are marked as pre-releases on GitHub.

A release is a git tag `vX.Y.Z[-pre]` on `main`: CI builds and pushes
`ghcr.io/floholz/matchowl:<version>` and publishes the GitHub release with the
matching section of this file.

## [Unreleased]

**Deploy:** migration 0048 runs at boot.

### Added

- **Registration links join pools.** Admin → People: pick any of your
  pools (Global and finished pools aside) when minting a link, and the
  account it creates joins them as a member once its email is verified —
  the same moment it joins Global; a Google sign-up joins right away. The
  pick carries over to the next link, and each link lists its pools.
  Migration 0048 (`signup_links.pools`).

### Fixed

- Registration links point at the player app (`APP_URL`), not the admin
  host they were minted on.
- Emails carry the Matchowl mark instead of the old WM-Tips logo.

## [1.0.0-alpha.5] - 2026-09-28

The admin release. The admin tooling moves into its own app on its own
host, and gains the group editor, the teams registry, mailings and richer
announcements — the pieces the pre-launch outreach needs.

**Deploy:** route `admin.<your host>` to the same container (see
`ADMIN_URL` in DEPLOY.md); migrations 0046 and 0047 run at boot.

### Added

- **Admin app.** The admin tooling moved out of the phone app into its own
  desktop-first SPA (`admin/`), served by the same binary on the admin host
  (`ADMIN_URL`, default `admin.` in place of `play.`; a local binary answers
  on `admin.localhost:8090`). Own origin, own sign-in (an admin or owner
  account), no PWA. Sidebar: Dashboard (owner stats) · Competitions (the
  tournament browser and import wizard) · Sync & services · People
  (registration links) · Announcements · Notifications (the delivery policy) · Dev (the
  harness, when `MATCHOWL_DEV=1`). The app's user menu links to it; the
  in-app `/admin`, `/owner`, `/announcements` and `/dev` pages are gone.
  `GET /api/appconfig` now carries `appUrl` and `adminUrl`.
- **Group editor.** Under Competitions → Groups (and in the import wizard's
  preview): drag teams between groups, or tap a team and tap a group; add
  and remove groups; saving makes every group-stage match follow its
  teams and sets the structure's group size to the largest group. The
  merged-groups warning shows on the offending column. The import wizard
  sends the edited groups along, so a Nations League can be fixed before
  it is seeded.
- **Teams registry.** Admin → Teams: one row per club or nation across
  every season it plays in (grouped by provider id, club key or name), the
  seasons with group and status, fields that differ between seasons
  flagged, and an editor for name, code and flag that applies to all
  seasons at once.
- **Provider ids on teams** (migration 0046: `provider`, `providerId`),
  written at import; "Fetch logos" on a season now also links older rows
  by name.
- **Mailings.** Admin → Mailings: one email to an audience. Markdown body
  with `{{name}}` for the recipient's first name, a button, and an
  audience filter (role, plays a season, member of a pool, joined
  before/after, only or never these addresses) whose recipient count and
  names update as you narrow it. The real mail is previewed as you type,
  "Send me a test" mails it to you, and a mailing is sent once: delivery
  goes through the notification ledger, so a repeat is a no-op and the
  mailing keeps its result. Recipients can opt out under a new "News from
  Matchowl" setting; unverified addresses never get mail. Migration 0047.
- **Announcements** take Markdown (bold, italic, links, lists) and an
  optional button (text + link, an in-app path works). The banner in the
  app and the broadcast mail render both; the admin page previews the mail.

## [1.0.0-alpha.4] - 2026-09-28

The head-to-head pool is the focus of the app, and after two weeks of
alpha it was too easy to lose sight of: a thin line on Home, a pool page
stuck in either "what happened" or "what's next", nothing about the duel
where the tips are placed. This pass makes the duel present everywhere.

### Added

- **Duel cards on Home.** Your pools now come first, and every
  head-to-head pool is a full card: the matchday's duel with the score,
  the verdict, who is next and what is still to do (matches to tip, Save
  Call left, ban open), plus "All duels" to unfold the rest of the
  matchday. Classic pools keep their compact row.
- **Matchday strip on the pool page.** The duel card and the round browser
  became one horizontal strip of matchday cards — past ones with your
  result, the open one with the live score, the next one with your rival
  and your to-do — centred on the current matchday with the neighbours
  peeking in, so last week and next week are one flick apart. The selected
  matchday's duels and your checklist follow, then the table.
- **Pool tab.** Head-to-head pools open on a new *Pool* tab (the strip,
  the duels, your checklist and a short table: the top three plus you)
  and the full table moves to its own *Leaderboard* tab with the
  Head-to-head / Points switch. Tab and selected matchday live in the URL,
  so coming back from a match lands where you were.
- **Rival strip on the competition hub.** With a matchday selected on the
  Matches tab, each of your head-to-head pools on that season says who you
  play and what is left to do, and links to the pool.
- **Pick marks on match rows.** The tip capsule carries a shield for a Save
  Call (yours in orange, the rival's revealed one in grey) and a ban badge
  (outlined for yours on the rival, filled red once the rival's ban on you
  is revealed at kick-off). With several pools the badge shows if any pool
  has a pick; the drawer keeps the per-pool detail.
- `GET /api/h2h/me` — all your head-to-head pools at a glance, one request
  shared by Home, the match lists and the hub.

### Changed

- **Joining a pool plays its season.** A pool binds one season, so a new
  member is a player of it right away and its matches reach their feed and
  Home; the same happens for all members when the owner changes the pool's
  season before its first matchday.
- Home's results list is capped at five rows with a "See all results" line.

### Fixed

- **Announcement broadcasts reached one person.** The notification
  ledger's dedup key for a broadcast did not include the recipient, and the
  ledger is unique per key and channel, so "Send as notification" delivered
  to the first eligible user and skipped the rest. The key now carries the
  user; mailings use the same shape.
- **Importer: tiered competitions.** The UEFA Nations League publishes one
  "Group 1".."Group 4" standings table per league tier without naming the
  tier, plus a ranking of third-placed teams; the import wizard merged them
  into five groups of up to 14 teams. Repeated table names now take the
  tier from the round label (`League A - 1` + `Group 1` = `A1`), ranking
  tables are skipped when real group tables exist, and the preview warns
  when a group has more teams than its matches per team allow. The live
  Nations League 2026/27 was repaired by hand on 2026-09-24.

## [1.0.0-alpha.3] - 2026-09-14

The head-to-head release. A pool can now be played as matchday duels
instead of one long points table — the thing meant to keep a season-long
game alive past matchday 12.

### Added

- **Head-to-head pools.** A pool has a mode: *classic* (the points table, as
  before) or *head-to-head*, chosen when the pool is created and defaulted
  from the season's shape (leagues get head-to-head, cups and the Champions
  League stay classic). In head-to-head, every matchday pairs you with a
  pool mate by a fixed rotation; the duel is decided by your tip points over
  that matchday's matches; win 3, draw 1, loss 0. The table is W-D-L with
  the season's tip points as tiebreak; the classic points table stays one
  tap away. An odd roster plays **the Ghost**, who scores the mean of
  everyone else. A matchday closes 24 h after its last scheduled kick-off;
  a match postponed past that is ignored for the duel and still counts for
  points.
- **Save Calls.** Per matchday you mark one or two tips (the pool's setting)
  as the ones you'd bet the house on: they count double for you.
- **Bans.** One per matchday, aimed at your rival: that match does not count
  for them. They only see it once it kicks off. A ban on a Save Call cancels
  the double and the match counts normal.
- **Where you do it.** Save Call and ban sit in the tip drawer of every match
  row and on the match page, next to your tip. The pool page has the duel
  card, the W-D-L table, a round browser (every duel of every matchday,
  with revealed Save Calls and bans) and "Your matchday", a checklist of
  what you still have to tip and pick, each row opening the match. Every
  matchday heading links to the competition's matchday.
- **Round results in the chat.** When a matchday closes, the app posts the
  duels into the pool chat ("Matchday 5 is in · Anna beat Flo 14–9 · …").
- **Notifications**: how your duel went when a matchday closes, and, at
  kick-off, that your rival banned that match for you. Both can be switched
  off in the notification settings.
- Home shows your current duel per head-to-head pool (score, rival, matches
  in), or the next rival while you wait for your first matchday.

### Changed

- **One season per pool.** A pool plays exactly one season; a second
  competition with the same friends is a second pool. Existing pools with
  several seasons keep the running one. Season and mode can be changed until
  the pool's first matchday kicks off, then they are locked. "Set up next
  season" carries the mode over.
- **Home's "Tip now"** looks a week ahead (the current matchday, midweek
  games included) instead of jumping to the next open pairing months out,
  and shows the day on each row, not just the time.
- The competition hub's matchday dropdown is in play order again; a single
  match pulled forward no longer drags its whole matchday ahead of the
  previous one.
- Home's status column shows the competition's short name when one is set,
  else the full name cut with an ellipsis — no more invented codes.

### Fixed

- Live chat updates were dead since the pools rename (the pages still
  listened to the old collection); the chat and the unread badge follow new
  messages again.

## [1.0.0-alpha.2] - 2026-09-13

### Added

- **One-time registration links.** While sign-up is closed
  (`REGISTRATION_OPEN=0`), an admin mints links under Admin → Registration
  links (a note on who it's for, optional expiry) and shares them; each link
  creates exactly one account, email/password or Google, and the list shows
  who came in through which one. Unused links can be revoked.

## [1.0.0-alpha.1] - 2026-09-13

The first build of Matchowl as its own app, for private testing with a
handful of friends. It grew out of the World Cup 2026 game (wm-pickems, see
[the report](https://floholz.com/wm2026/)) and now runs any competition.

### Added

- **Competitions, not one tournament.** Leagues, cups and tournaments are
  imported from API-Football through an admin wizard; each season carries its
  structure (groups, a single table with zones, a UEFA-style league phase,
  knockout rounds, two-legged ties) as data. Play the ones you follow.
- **One feed of matches** across everything you play: day by day, one card
  per competition, live scores, results with points, infinite scroll both
  ways, a sticky day strip.
- **The match row**: a stadium-board score in seven-segment digits next to
  your tip in an orange capsule; inline steppers that save on close; a
  knockout advancer dot; a lug and strip for two-legged ties.
- **Match page** at `/m/{id}`, also the desktop detail panel: hero, your tip
  with a points breakdown, friends' picks, bots, mini table.
- **Competition hub** with Overview · Matches · Table/Groups · Knockout ·
  Forecast tabs that swipe like a pager on touch.
- **Forecasts per season**: the full builder (groups + bracket) for
  tournaments, "calls" (champion, zones, stages) for leagues and cups.
- **Friends** (mutual, request + accept) with a board per competition, and
  **Pools**: private tables bound to one or more seasons, invite code or
  link, chat with GIFs, owner tools, a read-only archive once every season
  is over, and one-tap "set up next season".
- **Home**: tip now, forecast deadline, live, your pools, yesterday's points.
- **Identity**: the owl, three themes (warm dark, peach paper light, AMOLED),
  a PWA with push notifications and an "update ready" toast.
- **Onboarding and legal**: terms acceptance at sign-up (and on first Google
  sign-in), an in-app help page, terms / privacy / about pages.
- **Verification tiers**: anyone may register; friends, pools, chat and every
  email wait until the address is verified. Unverified accounts are purged
  after 180 days.
- **Sign-up switch** (`REGISTRATION_OPEN=0`) for closed test runs, and the
  build version in the Home footer.

### Changed

- **Scoring**: correct result 3, goal difference +1, exact score +3; the
  total-goals point of the World Cup rules is gone (it was a coin flip, per
  the post-tournament analysis).
- Deployment: the container, network and volume are named `matchowl`; the
  image is `ghcr.io/floholz/matchowl:<version>`; `APP_URL` sets the origin
  used in mail links.

### Fixed

- League badges were never fetched after the leagues → pools rename had
  rewritten the provider's badge URL.
- Notifications could reach bot accounts and unverified addresses through
  the per-record paths (pool invites, chat).
- A finished match kept a stale kick-off date when the provider had moved it.
- Mail subjects said "Acme" and links pointed at localhost on a fresh
  database (PocketBase defaults); the app name and URL are applied at boot.

[Unreleased]: https://github.com/floholz/matchowl/compare/v1.0.0-alpha.2...HEAD
[1.0.0-alpha.2]: https://github.com/floholz/matchowl/releases/tag/v1.0.0-alpha.2
[1.0.0-alpha.1]: https://github.com/floholz/matchowl/releases/tag/v1.0.0-alpha.1
