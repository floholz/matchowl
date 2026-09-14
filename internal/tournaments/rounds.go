package tournaments

import (
	"regexp"
	"sort"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// RoundMatch is the slice of a match the round grouping needs.
type RoundMatch struct {
	ID      string
	Stage   string
	Label   string // roundLabel, e.g. "Regular Season - 12"
	Kickoff time.Time
}

// Round is one matchday (or one knockout round) of a season: every match
// that shares a stage and round label. It is the unit head-to-head pools
// play in and the key the hub's matchday filter uses (`stage|roundLabel`).
type Round struct {
	Key      string
	Stage    string
	Label    string
	Num      int // the trailing number of the label, 0 when it has none
	MatchIDs []string
	// Scheduled kick-offs as they stand now: earliest, latest and the median
	// one. A rescheduled match can sit weeks away from the rest, so the
	// median is what orders rounds.
	First, Last, Median time.Time
}

var trailingNum = regexp.MustCompile(`(\d+)\s*$`)

// RoundKey is the client-visible key of a match's round.
func RoundKey(stage, label string) string { return stage + "|" + label }

// GroupRounds groups matches into rounds and orders them by play order:
// stage order in the structure, then the round number, then the median
// kick-off. Matches without a kick-off are ignored.
func GroupRounds(st *Structure, ms []RoundMatch) []Round {
	stageIdx := map[string]int{}
	if st != nil {
		for i, s := range st.Stages {
			stageIdx[s.Code] = i
		}
	}
	byKey := map[string]*Round{}
	kicks := map[string][]time.Time{}
	var order []string
	for _, m := range ms {
		if m.Kickoff.IsZero() {
			continue
		}
		k := RoundKey(m.Stage, m.Label)
		r := byKey[k]
		if r == nil {
			num := 0
			if mm := trailingNum.FindStringSubmatch(m.Label); mm != nil {
				num, _ = strconv.Atoi(mm[1])
			}
			r = &Round{Key: k, Stage: m.Stage, Label: m.Label, Num: num}
			byKey[k] = r
			order = append(order, k)
		}
		r.MatchIDs = append(r.MatchIDs, m.ID)
		kicks[k] = append(kicks[k], m.Kickoff)
	}
	out := make([]Round, 0, len(order))
	for _, k := range order {
		r := byKey[k]
		ts := kicks[k]
		sort.Slice(ts, func(i, j int) bool { return ts[i].Before(ts[j]) })
		r.First, r.Last, r.Median = ts[0], ts[len(ts)-1], ts[len(ts)/2]
		out = append(out, *r)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		ia, oka := stageIdx[a.Stage]
		ib, okb := stageIdx[b.Stage]
		if oka && okb && ia != ib {
			return ia < ib
		}
		if oka != okb {
			return oka // known stages first
		}
		if a.Num > 0 && b.Num > 0 && a.Num != b.Num {
			return a.Num < b.Num
		}
		return a.Median.Before(b.Median)
	})
	return out
}

// Rounds loads a season's matches and groups them into rounds.
func Rounds(app core.App, tournamentID string) ([]Round, error) {
	t, err := app.FindRecordById(collection, tournamentID)
	if err != nil {
		return nil, err
	}
	st, _ := StructureOf(t)
	recs, err := app.FindRecordsByFilter("matches", "tournament = {:t}", "kickoff", 0, 0,
		map[string]any{"t": tournamentID})
	if err != nil {
		return nil, err
	}
	ms := make([]RoundMatch, 0, len(recs))
	for _, r := range recs {
		ms = append(ms, RoundMatch{
			ID: r.Id, Stage: r.GetString("stage"), Label: r.GetString("roundLabel"),
			Kickoff: r.GetDateTime("kickoff").Time(),
		})
	}
	return GroupRounds(st, ms), nil
}

// FirstRoundAfter returns the first round, in play order, that has not
// started by t (no match of it has kicked off), or nil when none is left.
func FirstRoundAfter(rounds []Round, t time.Time) *Round {
	for i := range rounds {
		if rounds[i].First.After(t) {
			return &rounds[i]
		}
	}
	return nil
}
