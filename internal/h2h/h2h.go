package h2h

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/pools"
	"github.com/floholz/matchowl/internal/scoring"
	"github.com/floholz/matchowl/internal/tournaments"
)

// CloseGrace is how long after a round's last scheduled kick-off the round
// closes. A match that kicks off later than that (postponed, rescheduled)
// is ignored for the head-to-head; it still counts for the points table.
const CloseGrace = 24 * time.Hour

// PairResult is one decided duel: both round scores and the 3 / 1 / 0
// each side took. B is Ghost for the odd member out.
type PairResult struct {
	A      string `json:"a"`
	B      string `json:"b"`
	ScoreA int    `json:"scoreA"`
	ScoreB int    `json:"scoreB"`
	PtsA   int    `json:"ptsA"`
	PtsB   int    `json:"ptsB"`
}

// Results is what a closed round freezes (and what an open round shows
// provisionally): the duels, every member's round score, the Ghost's,
// the matches that counted and the per-member, per-match points behind
// the scores.
type Results struct {
	Pairs     []PairResult              `json:"pairs"`
	Scores    map[string]int            `json:"scores"`
	Ghost     int                       `json:"ghost"`
	Counted   []string                  `json:"counted"`
	Breakdown map[string]map[string]int `json:"breakdown"`
	// Every member's save calls and ban in the round (all of them at close;
	// the API reveals only kicked-off ones while the round is open).
	Picks map[string]*UserPicks `json:"picks"`
}

// Tick opens rounds that have kicked off and closes rounds past their
// close time, for every head-to-head pool. Idempotent; runs on a cron,
// on reads and after the dev simulator moves the clock.
func Tick(app core.App) {
	recs, err := app.FindRecordsByFilter("pools", "mode = {:m}", "", 0, 0,
		map[string]any{"m": pools.ModeH2H})
	if err != nil {
		log.Printf("[h2h] list pools: %v", err)
		return
	}
	for _, lg := range recs {
		if err := TickPool(app, lg); err != nil {
			log.Printf("[h2h] pool %s: %v", lg.Id, err)
		}
	}
}

// TickPool is Tick for one pool.
func TickPool(app core.App, lg *core.Record) error {
	if lg.GetString("mode") != pools.ModeH2H {
		return nil
	}
	tid := pools.Season(lg)
	if tid == "" {
		return nil
	}
	now := clock.Now(app)
	rows, err := roundRows(app, lg.Id)
	if err != nil {
		return err
	}
	byKey := map[string]*core.Record{}
	for _, r := range rows {
		byKey[r.GetString("key")] = r
	}
	// Open: every playable round (from the pool's first round on) whose
	// first match has kicked off and that has no row yet, in play order so
	// ordinals follow the schedule.
	for _, r := range playable(app, lg, tid) {
		if r.First.After(now) || byKey[r.Key] != nil {
			continue
		}
		row, err := open(app, lg, r, len(rows))
		if err != nil {
			return fmt.Errorf("open %s: %w", r.Key, err)
		}
		rows = append(rows, row)
		byKey[r.Key] = row
	}
	// Close: open rows past their close time.
	for _, row := range rows {
		if row.GetString("status") != "open" || row.GetDateTime("closesAt").Time().After(now) {
			continue
		}
		if err := closeRound(app, lg, row, now); err != nil {
			return fmt.Errorf("close %s: %w", row.GetString("key"), err)
		}
	}
	return nil
}

// playable lists the rounds a pool plays: the season's rounds from the
// first one that had not started when the pool was created.
func playable(app core.App, lg *core.Record, tid string) []tournaments.Round {
	rounds, err := tournaments.Rounds(app, tid)
	if err != nil {
		return nil
	}
	created := lg.GetDateTime("created").Time()
	for i := range rounds {
		if rounds[i].First.After(created) {
			return rounds[i:]
		}
	}
	return nil
}

func roundRows(app core.App, poolID string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("h2h_rounds", "pool = {:p}", "ordinal", 0, 0,
		map[string]any{"p": poolID})
}

// roster is the pool's members in a stable order (join time, then id).
func roster(app core.App, poolID string) []string {
	ms, _ := app.FindRecordsByFilter("pool_members", "pool = {:p}", "joinedAt,id", 0, 0,
		map[string]any{"p": poolID})
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m.GetString("user"))
	}
	return out
}

func open(app core.App, lg *core.Record, r tournaments.Round, ordinal int) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("h2h_rounds")
	if err != nil {
		return nil, err
	}
	row := core.NewRecord(col)
	row.Set("pool", lg.Id)
	row.Set("key", r.Key)
	row.Set("stage", r.Stage)
	row.Set("label", r.Label)
	row.Set("num", r.Num)
	row.Set("ordinal", ordinal)
	row.Set("firstKickoff", r.First)
	row.Set("closesAt", r.Last.Add(CloseGrace))
	row.Set("status", "open")
	row.Set("pairings", Pairings(roster(app, lg.Id), ordinal))
	row.Set("matches", r.MatchIDs)
	if err := app.Save(row); err != nil {
		return nil, err
	}
	return row, nil
}

func closeRound(app core.App, lg *core.Record, row *core.Record, now time.Time) error {
	res, err := Compute(app, lg, row)
	if err != nil {
		return err
	}
	row.Set("results", res)
	row.Set("status", "closed")
	row.Set("closedAt", now)
	return app.Save(row)
}

// pairingsOf decodes a row's pairings.
func pairingsOf(row *core.Record) []Pair {
	var ps []Pair
	_ = row.UnmarshalJSONField("pairings", &ps)
	return ps
}

// ResultsOf decodes a closed row's frozen results (nil for an open row).
func ResultsOf(row *core.Record) *Results {
	if row.GetString("status") != "closed" {
		return nil
	}
	var res Results
	if err := row.UnmarshalJSONField("results", &res); err != nil {
		return nil
	}
	return &res
}

// Compute scores a round from what is known now: every match of the
// round that is finished and kicked off before the round's close time
// counts; a member's round score is the sum of their tip points over
// those matches, doubled on their save calls and zeroed on the match
// their rival banned (a banned save call counts normal). Frozen at
// close; provisional while the round is open.
func Compute(app core.App, lg *core.Record, row *core.Record) (*Results, error) {
	var ids []string
	_ = row.UnmarshalJSONField("matches", &ids)
	closesAt := row.GetDateTime("closesAt").Time()
	counted := make([]string, 0, len(ids))
	for _, id := range ids {
		m, err := app.FindRecordById("matches", id)
		if err != nil {
			continue
		}
		if m.GetString("status") != "finished" || m.GetString("finalizedAt") == "" {
			continue
		}
		if !m.GetDateTime("kickoff").Time().Before(closesAt) {
			continue
		}
		counted = append(counted, id)
	}
	pairs := pairingsOf(row)
	picks := picksFor(app, lg.Id, row.GetString("key"))
	res := &Results{
		Pairs:     make([]PairResult, 0, len(pairs)),
		Scores:    map[string]int{},
		Counted:   counted,
		Breakdown: map[string]map[string]int{},
		Picks:     picks,
	}
	members := make([]string, 0, 2*len(pairs))
	for _, p := range pairs {
		members = append(members, p.A)
		if p.B != Ghost {
			members = append(members, p.B)
		}
	}
	if len(counted) > 0 && len(members) > 0 {
		base, err := matchPoints(app, lg, members, counted)
		if err != nil {
			return nil, err
		}
		// Effective points depend on the duel: my saves, my rival's ban.
		score := func(uid, rival string) {
			perMatch := map[string]int{}
			sum := 0
			for _, m := range counted {
				p := effective(base[uid][m], picks[uid], picks[rival], m)
				if p != 0 || base[uid][m] != 0 || picks[uid].saved(m) {
					perMatch[m] = p
				}
				sum += p
			}
			res.Breakdown[uid] = perMatch
			res.Scores[uid] = sum
		}
		for _, p := range pairs {
			score(p.A, p.B)
			if p.B != Ghost {
				score(p.B, p.A)
			}
		}
	}
	for _, uid := range members {
		if _, ok := res.Scores[uid]; !ok {
			res.Scores[uid] = 0
		}
	}
	// The Ghost scores the rounded mean of everyone it is not playing.
	for _, p := range pairs {
		if p.B != Ghost {
			continue
		}
		sum, n := 0, 0
		for _, uid := range members {
			if uid != p.A {
				sum += res.Scores[uid]
				n++
			}
		}
		if n > 0 {
			res.Ghost = int(math.Round(float64(sum) / float64(n)))
		}
	}
	for _, p := range pairs {
		pr := PairResult{A: p.A, B: p.B, ScoreA: res.Scores[p.A]}
		if p.B == Ghost {
			pr.ScoreB = res.Ghost
		} else {
			pr.ScoreB = res.Scores[p.B]
		}
		pr.PtsA, pr.PtsB = outcome(pr.ScoreA, pr.ScoreB), outcome(pr.ScoreB, pr.ScoreA)
		res.Pairs = append(res.Pairs, pr)
	}
	return res, nil
}

// matchPoints returns user → match → tip points for the members over the
// matches, from the precomputed match_scores under the pool's scoring
// config. (Save calls and bans adjust these per pool — step 3.)
func matchPoints(app core.App, lg *core.Record, members, matches []string) (map[string]map[string]int, error) {
	cfgID := lg.GetString("scoringConfig")
	if cfgID == "" {
		def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
		if err != nil {
			return nil, err
		}
		cfgID = def.Id
	}
	params := map[string]any{"c": cfgID}
	mconds := make([]string, 0, len(matches))
	for i, id := range matches {
		k := fmt.Sprintf("m%d", i)
		params[k] = id
		mconds = append(mconds, "match = {:"+k+"}")
	}
	uconds := make([]string, 0, len(members))
	for i, id := range members {
		k := fmt.Sprintf("u%d", i)
		params[k] = id
		uconds = append(uconds, "user = {:"+k+"}")
	}
	filter := "config = {:c} && (" + strings.Join(mconds, " || ") + ") && (" + strings.Join(uconds, " || ") + ")"
	recs, err := app.FindRecordsByFilter("match_scores", filter, "", 0, 0, params)
	if err != nil {
		return nil, err
	}
	out := map[string]map[string]int{}
	for _, r := range recs {
		uid := r.GetString("user")
		if out[uid] == nil {
			out[uid] = map[string]int{}
		}
		out[uid][r.GetString("match")] = r.GetInt("points")
	}
	return out, nil
}

// TableRow is one member's line in the head-to-head table.
type TableRow struct {
	UserID       string `json:"userId"`
	Name         string `json:"name"`
	Avatar       string `json:"avatar"`
	Role         string `json:"role"`
	Played       int    `json:"played"`
	Won          int    `json:"won"`
	Drawn        int    `json:"drawn"`
	Lost         int    `json:"lost"`
	Points       int    `json:"points"`     // 3 / 1 / 0 over closed rounds
	TipsPoints   int    `json:"tipsPoints"` // the season's tip points: the tiebreak
	ScoreFor     int    `json:"scoreFor"`   // round scores summed
	ScoreAgainst int    `json:"scoreAgainst"`
}

// Table builds the pool's head-to-head standings over its closed rounds:
// h2h points, then the season's tip points, then the classic tiebreakers
// (the order the points board already has). Current members only;
// members without a round yet sit at the bottom with 0 played.
func Table(app core.App, lg *core.Record, rows []*core.Record) []TableRow {
	ids := roster(app, lg.Id)
	cfgID := lg.GetString("scoringConfig")
	board := scoring.Board(app, ids, cfgID, []string{pools.Season(lg)})
	rank := map[string]int{}
	out := make([]TableRow, 0, len(board))
	byID := map[string]*TableRow{}
	for i, b := range board {
		rank[b.UserID] = i
		out = append(out, TableRow{UserID: b.UserID, Name: b.Name, Avatar: b.Avatar, Role: b.Role, TipsPoints: b.TipsPoints})
	}
	for i := range out {
		byID[out[i].UserID] = &out[i]
	}
	for _, row := range rows {
		res := ResultsOf(row)
		if res == nil {
			continue
		}
		for _, p := range res.Pairs {
			apply := func(uid string, mine, theirs, pts int) {
				t := byID[uid]
				if t == nil {
					return // left the pool
				}
				t.Played++
				t.Points += pts
				t.ScoreFor += mine
				t.ScoreAgainst += theirs
				switch pts {
				case 3:
					t.Won++
				case 1:
					t.Drawn++
				default:
					t.Lost++
				}
			}
			apply(p.A, p.ScoreA, p.ScoreB, p.PtsA)
			if p.B != Ghost {
				apply(p.B, p.ScoreB, p.ScoreA, p.PtsB)
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Points != out[j].Points {
			return out[i].Points > out[j].Points
		}
		return rank[out[i].UserID] < rank[out[j].UserID]
	})
	return out
}
