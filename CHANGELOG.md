# Changelog

All notable changes to Matchowl. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[SemVer](https://semver.org/). Pre-releases (`-alpha.N`, `-beta.N`, `-rc.N`) are
private or limited test runs and are marked as pre-releases on GitHub.

A release is a git tag `vX.Y.Z[-pre]` on `main`: CI builds and pushes
`ghcr.io/floholz/matchowl:<version>` and publishes the GitHub release with the
matching section of this file.

## [Unreleased]

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
