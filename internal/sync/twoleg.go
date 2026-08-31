package sync

import (
	"log"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/tournaments"
)

// FixTwoLeggedAdvancers corrects the advancer on two-legged knockout ties —
// two matches in the same knockout stage between the same two teams
// (UEFA-style home-and-away rounds). applyResult judges each leg on its own
// score, which is wrong for a tie: the first leg must carry no advancer, and
// the second leg's advancer is the aggregate winner. Cup replays fall out
// correctly too (draw + decided rematch = the rematch's winner). Returns how
// many records changed.
func FixTwoLeggedAdvancers(app core.App, tournamentID string, st *tournaments.Structure) int {
	ms, err := app.FindRecordsByFilter("matches",
		"tournament = {:t} && homeTeam != '' && awayTeam != ''", "kickoff,num", 0, 0,
		map[string]any{"t": tournamentID})
	if err != nil {
		return 0
	}
	pairs := map[string][]*core.Record{}
	for _, m := range ms {
		if !st.IsKnockout(m.GetString("stage")) {
			continue
		}
		h, a := m.GetString("homeTeam"), m.GetString("awayTeam")
		if a < h {
			h, a = a, h
		}
		pairs[m.GetString("stage")+"|"+h+"|"+a] = append(pairs[m.GetString("stage")+"|"+h+"|"+a], m)
	}
	fixed := 0
	for _, legs := range pairs {
		if len(legs) != 2 {
			continue
		}
		first, second := legs[0], legs[1] // kickoff order (query sort)
		for _, leg := range []struct {
			rec  *core.Record
			want string
		}{{first, ""}, {second, tieAdvancer(first, second)}} {
			if leg.rec.GetString("advancer") == leg.want {
				continue
			}
			leg.rec.Set("advancer", leg.want)
			if err := app.Save(leg.rec); err != nil {
				log.Printf("[sync] two-leg advancer %s: %v", leg.rec.GetString("extId"), err)
				continue
			}
			fixed++
		}
	}
	return fixed
}

// OtherLeg returns the other leg of a two-legged knockout tie — the one
// other match in the same knockout stage between the same two teams — and
// whether m is the first leg (earlier kickoff). Nil when m isn't part of a
// two-legged tie (single-leg round, teams unresolved, or no unique partner).
func OtherLeg(app core.App, st *tournaments.Structure, m *core.Record) (*core.Record, bool) {
	if !st.IsKnockout(m.GetString("stage")) {
		return nil, false
	}
	h, a := m.GetString("homeTeam"), m.GetString("awayTeam")
	if h == "" || a == "" {
		return nil, false
	}
	legs, err := app.FindRecordsByFilter("matches",
		"tournament = {:t} && stage = {:s} && id != {:id} && ((homeTeam = {:h} && awayTeam = {:a}) || (homeTeam = {:a} && awayTeam = {:h}))",
		"", 0, 0,
		map[string]any{"t": m.GetString("tournament"), "s": m.GetString("stage"), "id": m.Id, "h": h, "a": a})
	if err != nil || len(legs) != 1 {
		return nil, false
	}
	other := legs[0]
	mk, ok := m.GetDateTime("kickoff").Time(), other.GetDateTime("kickoff").Time()
	first := mk.Before(ok) || (mk.Equal(ok) && m.GetInt("num") < other.GetInt("num"))
	return other, first
}

// legGoals is one leg's final score: the cumulative after-120 score when the
// leg went to extra time (etHome/etAway 0/0 means "no ET" by convention),
// else the full-time score.
func legGoals(m *core.Record) (h, a int) {
	if m.GetInt("etHome") != 0 || m.GetInt("etAway") != 0 {
		return m.GetInt("etHome"), m.GetInt("etAway")
	}
	return m.GetInt("ftHome"), m.GetInt("ftAway")
}

// tieAdvancer returns the aggregate winner of a finished two-legged tie, or
// "" while either leg is unfinished or the data can't decide it. A level
// aggregate is broken by the second leg's shootout (no away-goals rule —
// UEFA abolished it in 2021).
func tieAdvancer(first, second *core.Record) string {
	if first.GetString("status") != "finished" || second.GetString("status") != "finished" {
		return ""
	}
	agg := map[string]int{}
	for _, m := range []*core.Record{first, second} {
		h, a := legGoals(m)
		agg[m.GetString("homeTeam")] += h
		agg[m.GetString("awayTeam")] += a
	}
	home, away := second.GetString("homeTeam"), second.GetString("awayTeam")
	switch {
	case agg[home] > agg[away]:
		return home
	case agg[away] > agg[home]:
		return away
	}
	if pH, pA := second.GetInt("penHome"), second.GetInt("penAway"); pH != pA {
		if pH > pA {
			return home
		}
		return away
	}
	return ""
}
