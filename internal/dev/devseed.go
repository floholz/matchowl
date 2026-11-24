package dev

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"regexp"
	"sort"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"

	"github.com/floholz/matchowl/internal/challenges"
	"github.com/floholz/matchowl/internal/h2h"
	"github.com/floholz/matchowl/internal/importer"
	"github.com/floholz/matchowl/internal/pools"
	"github.com/floholz/matchowl/internal/scoring"
	"github.com/floholz/matchowl/internal/seed"
	"github.com/floholz/matchowl/internal/teams"
	"github.com/floholz/matchowl/internal/tips"
	"github.com/floholz/matchowl/internal/tournaments"
)

// The dev seed (`make dev-reset`): a fresh dev database with useful data.
//
//  1. The sports snapshot (snapshot.go), moved by whole weeks so its moment
//     lands on today — weekdays stay right — then brought to now with the
//     simulator: real results before the snapshot, simulated ones between it
//     and now, live matches where a kick-off is under way, and everything
//     later still open.
//  2. Made-up people on top (nothing here comes from a real account):
//     Mira Novak is you (owner, all logins below use DemoPassword), with
//     friends, a pending request each way, a classic and a head-to-head pool,
//     tips around today, pool chat, challenges in every state, followed
//     teams, a country and a pinned competition. Bots fill the pools.
//
// It only runs on an empty database and with MATCHOWL_DEV=1.

// DemoPassword is the password of every seeded account.
const DemoPassword = "matchowl-dev"

type demoUser struct {
	key, name, email, role, country string
}

var demoUsers = []demoUser{
	{"mira", "Mira Novak", "mira@example.com", "owner", "at"},
	{"max", "Max Demo", "max@example.com", "", "de"},
	{"lena", "Lena Berg", "lena@example.com", "", "at"},
	{"jonas", "Jonas Huber", "jonas@example.com", "", "at"},
	{"sofia", "Sofia Rossi", "sofia@example.com", "", "it"},
	{"tom", "Tom Baker", "tom@example.com", "", "gb-eng"},
}

// SeedCommand is `matchowl dev-seed`.
func SeedCommand(app core.App) *cobra.Command {
	var snapPath string
	var noLogos bool
	cmd := &cobra.Command{
		Use:   "dev-seed",
		Short: "Fill an empty dev database with the sports snapshot and demo people (MATCHOWL_DEV=1)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !Enabled() {
				return fmt.Errorf("dev-seed only runs with MATCHOWL_DEV=1")
			}
			if err := app.RunAllMigrations(); err != nil {
				return err
			}
			return Seed(app, snapPath, !noLogos)
		},
	}
	cmd.Flags().StringVar(&snapPath, "snapshot", SnapshotPath, "sports snapshot to load")
	cmd.Flags().BoolVar(&noLogos, "no-logos", false, "skip fetching crests from the provider CDN")
	return cmd
}

// SnapshotCommand is `matchowl dev-snapshot`: export the sports data of the
// database in --dir (use a copy of a real one) into the committed snapshot.
func SnapshotCommand(app core.App) *cobra.Command {
	var out string
	var noOrigins bool
	cmd := &cobra.Command{
		Use:   "dev-snapshot",
		Short: "Export competitions, seasons, teams and matches (no people) into the dev seed snapshot",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.RunAllMigrations(); err != nil {
				return err
			}
			// Team countries come from the provider's /teams (one request
			// per season); older databases never fetched them.
			if key := os.Getenv("API_FOOTBALL_KEY"); key != "" && !noOrigins {
				if _, err := teams.LinkAll(app); err != nil {
					return err
				}
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				fmt.Printf("team countries: %v\n", importer.BackfillOrigins(ctx, app, key))
				cancel()
			}
			s, err := ExportSnapshot(app, out)
			if err != nil {
				return err
			}
			fmt.Printf("snapshot %s taken at %s: %v\n", out, s.TakenAt.Format(time.RFC3339), s.Counts)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", SnapshotPath, "where to write the snapshot")
	cmd.Flags().BoolVar(&noOrigins, "no-origins", false, "don't fetch team countries from the provider first")
	return cmd
}

// Seed loads the snapshot and the demo people into an empty database.
func Seed(app core.App, snapPath string, logos bool) error {
	if n, _ := app.CountRecords("users"); n > 0 {
		return fmt.Errorf("the database already has %d users — dev-seed only fills an empty one (make dev-reset wipes pb_data first)", n)
	}
	s, err := readSnapshot(snapPath)
	if err != nil {
		return fmt.Errorf("read snapshot: %w", err)
	}
	now := time.Now().UTC()
	week := 7 * 24 * time.Hour
	shift := time.Duration(float64(now.Sub(s.TakenAt))/float64(week)+0.5) * week
	if now.Sub(s.TakenAt) < 0 {
		shift = -time.Duration(float64(s.TakenAt.Sub(now))/float64(week)+0.5) * week
	}
	log.Printf("[devseed] snapshot from %s, moved by %d weeks", s.TakenAt.Format("2006-01-02"), int(shift/week))
	if err := seed.EnsureDefaultScoringConfig(app); err != nil {
		return err
	}
	n, err := importSnapshot(app, s, shift)
	if err != nil {
		return err
	}
	// The server's boot seed expects the World Cup season; a snapshot
	// without it gets the embedded one.
	if err := seed.Run(app); err != nil {
		log.Printf("[devseed] world cup seed: %v", err)
	}
	log.Printf("[devseed] %d sports rows", n)
	if linked, err := teams.LinkAll(app); err == nil && linked > 0 {
		log.Printf("[devseed] linked %d team rows to identities", linked)
	}
	if logos {
		log.Printf("[devseed] fetching crests…")
		log.Printf("[devseed] %d crests", attachLogos(app, s))
	}
	// Results up to now: the snapshot's own before its moment, simulated
	// after it, live matches where a kick-off is under way.
	if err := simulate(app, now); err != nil {
		return fmt.Errorf("simulate: %w", err)
	}
	if err := seedPeople(app, now); err != nil {
		return fmt.Errorf("people: %w", err)
	}
	if err := scoring.Recompute(app); err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Dev database ready. Sign in as any of:")
	for _, u := range demoUsers {
		fmt.Printf("  %-12s %s\n", u.name, u.email)
	}
	fmt.Printf("password: %s\n", DemoPassword)
	return nil
}

// seasonPick is an active season with its matches.
type seasonPick struct {
	t       *core.Record
	matches []*core.Record
}

// activeSeasons lists the active seasons, the ones with the most matches
// around now first.
func activeSeasons(app core.App, now time.Time) []seasonPick {
	recs, _ := tournaments.Active(app)
	var out []seasonPick
	for _, t := range recs {
		ms, _ := app.FindRecordsByFilter("matches", "tournament = {:t}", "kickoff", 0, 0, map[string]any{"t": t.Id})
		if len(ms) > 0 {
			out = append(out, seasonPick{t, ms})
		}
	}
	near := func(p seasonPick) int {
		c := 0
		for _, m := range p.matches {
			if d := m.GetDateTime("kickoff").Time().Sub(now); d > -21*24*time.Hour && d < 14*24*time.Hour {
				c++
			}
		}
		return c
	}
	sort.SliceStable(out, func(i, j int) bool { return near(out[i]) > near(out[j]) })
	return out
}

func seedPeople(app core.App, now time.Time) error {
	rng := rand.New(rand.NewSource(20261124))
	tips.SetBypass(true)
	defer tips.SetBypass(false)

	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return err
	}
	ids := map[string]string{}
	for _, d := range demoUsers {
		u := core.NewRecord(usersCol)
		u.SetEmail(d.email)
		u.SetPassword(DemoPassword)
		u.Set("name", d.name)
		u.Set("verified", true)
		u.Set("role", d.role)
		u.Set("country", d.country)
		u.Set("termsVersion", termsVersion()) // accepted, so the terms screen doesn't block
		if err := app.Save(u); err != nil {
			return fmt.Errorf("user %s: %w", d.key, err)
		}
		ids[d.key] = u.Id
	}
	me := ids["mira"]

	// Friends: three accepted, one request each way.
	for _, f := range [][3]string{
		{"mira", "max", "accepted"}, {"lena", "mira", "accepted"}, {"mira", "jonas", "accepted"},
		{"max", "lena", "accepted"}, {"sofia", "mira", "pending"}, {"mira", "tom", "pending"},
	} {
		if err := save(app, "friendships", map[string]any{"requester": ids[f[0]], "receiver": ids[f[1]], "status": f[2]}); err != nil {
			return err
		}
	}

	seasons := activeSeasons(app, now)
	if len(seasons) == 0 {
		log.Printf("[devseed] no active season in the snapshot: people only")
		return nil
	}
	def, _ := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
	pool := func(name, code, owner string, season *core.Record, members []string, forceMode string) (*core.Record, error) {
		mode, saveCalls, _ := pools.Defaults(season)
		if forceMode != "" {
			mode = forceMode
		}
		vals := map[string]any{"name": name, "inviteCode": code, "owner": ids[owner], "tournaments": []string{season.Id}, "mode": mode, "saveCalls": saveCalls}
		if def != nil {
			vals["scoringConfig"] = def.Id
		}
		rec, err := saveRec(app, "pools", vals)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			role := "member"
			if m == owner {
				role = "owner"
			}
			if err := save(app, "pool_members", map[string]any{"pool": rec.Id, "user": ids[m], "role": role, "joinedAt": now.Add(-30 * 24 * time.Hour)}); err != nil {
				return nil, err
			}
		}
		return rec, nil
	}

	// Pool 1: classic, the busiest season. Pool 2: head-to-head, the next.
	a := seasons[0]
	classic, err := pool("Stammtisch", "DEVPOOL1", "mira", a.t, []string{"mira", "max", "lena", "jonas"}, "classic")
	if err != nil {
		return err
	}
	if _, err := makeBots(app, 3, []string{classic.Id}, a.t); err != nil {
		return err
	}
	poolSeasons := []seasonPick{a}
	var duel *core.Record
	if len(seasons) > 1 {
		b := seasons[1]
		if duel, err = pool("Bürocup", "DEVPOOL2", "lena", b.t, []string{"lena", "mira", "tom"}, "h2h"); err != nil {
			return err
		}
		if _, err := makeBots(app, 2, []string{duel.Id}, b.t); err != nil {
			return err
		}
		poolSeasons = append(poolSeasons, b)
	}

	// Tips: most past matches of the season, some upcoming ones.
	members := map[string][]string{a.t.Id: {"mira", "max", "lena", "jonas"}}
	if len(poolSeasons) > 1 {
		members[poolSeasons[1].t.Id] = []string{"lena", "mira", "tom"}
	}
	tipped := 0
	for _, p := range poolSeasons {
		for _, m := range p.matches {
			ko := m.GetDateTime("kickoff").Time()
			d := ko.Sub(now)
			// The whole season so far (bots tip every match, so a shorter
			// history would sink the demo people on the boards), and the
			// next ten days.
			if d > 10*24*time.Hour || m.GetString("homeTeam") == "" || m.GetString("awayTeam") == "" {
				continue
			}
			for _, who := range members[p.t.Id] {
				chance := 0.85
				if d > 0 {
					chance = 0.45
				}
				if who == "mira" && d > 0 && d < 3*24*time.Hour {
					chance = 0.3 // leave some for "Tip now"
				}
				if rng.Float64() > chance {
					continue
				}
				if err := save(app, "tips", map[string]any{"user": ids[who], "match": m.Id, "ftHome": rng.Intn(4), "ftAway": rng.Intn(3)}); err != nil {
					return err
				}
				tipped++
			}
		}
	}
	log.Printf("[devseed] %d tips from the demo people", tipped)

	// Pool chat.
	for _, line := range [][2]string{
		{"lena", "Who's watching tonight? 🍻"},
		{"max", "On my way, save me a seat"},
		{"jonas", "My tips are a disaster this week"},
		{"mira", "Told you to back the home side 😏"},
		{"max", "Rematch next round, loser buys"},
	} {
		if err := save(app, "pool_messages", map[string]any{"pool": classic.Id, "user": ids[line[0]], "text": line[1]}); err != nil {
			return err
		}
	}

	// Challenges: one invite to answer, one accepted and open, a few played.
	var upcoming, played []*core.Record
	for _, m := range a.matches {
		if m.GetString("homeTeam") == "" || m.GetString("awayTeam") == "" {
			continue
		}
		if m.GetString("status") == "scheduled" && m.GetDateTime("kickoff").Time().After(now.Add(2*time.Hour)) {
			upcoming = append(upcoming, m)
		} else if m.GetString("finalizedAt") != "" && m.GetDateTime("kickoff").Time().After(now.Add(-28*24*time.Hour)) {
			played = append(played, m)
		}
	}
	challenge := func(m *core.Record, from, to, status, stake string) error {
		_, err := saveRec(app, challenges.Collection, map[string]any{"match": m.Id, "challenger": ids[from], "opponent": ids[to], "status": status, "stake": stake})
		return err
	}
	if len(upcoming) > 1 {
		if err := challenge(upcoming[0], "max", "mira", challenges.Pending, "Loser buys the next round"); err != nil {
			return err
		}
		if err := challenge(upcoming[1], "mira", "lena", challenges.Accepted, "Coffee"); err != nil {
			return err
		}
	}
	for i, m := range played {
		if i >= 4 {
			break
		}
		from, to := "mira", "max"
		if i == 3 {
			from, to = "lena", "mira"
		}
		// Both sides tipped a played challenge.
		for _, who := range []string{from, to} {
			if _, err := app.FindFirstRecordByFilter("tips", "user = {:u} && match = {:m}", map[string]any{"u": ids[who], "m": m.Id}); err != nil {
				if err := save(app, "tips", map[string]any{"user": ids[who], "match": m.Id, "ftHome": rng.Intn(4), "ftAway": rng.Intn(3)}); err != nil {
					return err
				}
			}
		}
		if err := challenge(m, from, to, challenges.Accepted, []string{"", "Pizza", "Bragging rights", ""}[i]); err != nil {
			return err
		}
		if err := challenges.Settle(app, m); err != nil {
			return err
		}
	}

	// Followed teams: one from a season no pool plays (so "my matches"
	// shows a followed team's games), one from the classic pool's season.
	follow := func(teamRow string) {
		tm, err := app.FindRecordById("teams", teamRow)
		if err != nil || tm.GetString("identity") == "" {
			return
		}
		_ = save(app, teams.Follows, map[string]any{"user": me, "team": tm.GetString("identity")})
	}
	if len(seasons) > len(poolSeasons) {
		if ms := seasons[len(poolSeasons)].matches; len(ms) > 0 {
			follow(ms[len(ms)/2].GetString("homeTeam"))
		}
	}
	if len(a.matches) > 0 {
		follow(a.matches[0].GetString("awayTeam"))
	}

	// Pin the first competition no pool plays.
	for _, p := range seasons[len(poolSeasons):] {
		if c, err := app.FindRecordById("competitions", p.t.GetString("competition")); err == nil {
			if u, err := app.FindRecordById("users", me); err == nil {
				u.Set("pinnedCompetitions", []string{c.GetString("key")})
				_ = app.Save(u)
			}
			break
		}
	}

	// Head-to-head rounds for the duel pool.
	if duel != nil {
		if err := h2h.TickPool(app, duel); err != nil {
			log.Printf("[devseed] h2h: %v", err)
		}
	}
	return nil
}

func saveRec(app core.App, col string, vals map[string]any) (*core.Record, error) {
	c, err := app.FindCollectionByNameOrId(col)
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(c)
	for k, v := range vals {
		rec.Set(k, v)
	}
	if err := app.Save(rec); err != nil {
		return nil, fmt.Errorf("%s: %w", col, err)
	}
	return rec, nil
}

func save(app core.App, col string, vals map[string]any) error {
	_, err := saveRec(app, col, vals)
	return err
}

var termsRe = regexp.MustCompile(`TERMS_VERSION\s*=\s*'([^']+)'`)

// termsVersion is the current terms version the app asks people to accept
// (frontend/src/lib/legal.ts), so demo accounts don't land on the terms
// screen. Empty when the file isn't there.
func termsVersion() string {
	b, err := os.ReadFile("frontend/src/lib/legal.ts")
	if err != nil {
		return ""
	}
	if m := termsRe.FindSubmatch(b); m != nil {
		return string(m[1])
	}
	return ""
}
