# Changelog

All notable changes to Matchowl. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); versions follow
[SemVer](https://semver.org/). Pre-releases (`-alpha.N`, `-beta.N`, `-rc.N`) are
private or limited test runs and are marked as pre-releases on GitHub.

A release is a git tag `vX.Y.Z[-pre]` on `main`: CI builds and pushes
`ghcr.io/floholz/matchowl:<version>` and publishes the GitHub release with the
matching section of this file.

## [Unreleased]

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

[Unreleased]: https://github.com/floholz/matchowl/compare/v1.0.0-alpha.1...HEAD
[1.0.0-alpha.1]: https://github.com/floholz/matchowl/releases/tag/v1.0.0-alpha.1
