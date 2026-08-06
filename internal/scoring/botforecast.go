package scoring

import (
	"math/rand"
	"sort"
	"strconv"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/tournaments"
)

// RandomForecast builds a fully self-consistent random Forecast for one
// tournament (group order, extra qualifiers, and a bracket whose every
// winner is one of that match's resolved participants) using the same
// resolver the scorer uses — so bot players score coherently. Used by the
// dev bot generator.
func RandomForecast(app core.App, tournament *core.Record, rng *rand.Rand) (
	order map[string][]string,
	thirds map[string]string,
	bracket map[string]string,
	err error,
) {
	st, err := tournaments.StructureOf(tournament)
	if err != nil {
		return nil, nil, nil, err
	}
	eq := st.ExtraQualifiers

	groups, err := app.FindRecordsByFilter("tournament_groups",
		"tournament = {:t}", "letter", 0, 0, map[string]any{"t": tournament.Id})
	if err != nil {
		return nil, nil, nil, err
	}
	order = map[string][]string{}
	letters := make([]string, 0, len(groups))
	for _, g := range groups {
		ids := append([]string{}, g.GetStringSlice("teams")...)
		rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		order[g.GetString("letter")] = ids
		letters = append(letters, g.GetString("letter"))
	}

	// Pick the groups whose extra-qualifier-position team advances (WC2026:
	// 8 of the 12 third-placed teams).
	thirds = map[string]string{}
	if eq != nil {
		rng.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
		n := eq.Count
		if n > len(letters) {
			n = len(letters)
		}
		for _, l := range letters[:n] {
			if len(order[l]) >= eq.FromPosition {
				thirds[l] = order[l][eq.FromPosition-1]
			}
		}
	}

	stageRank := map[string]int{}
	for i, stg := range st.Stages {
		stageRank[stg.Code] = i
	}

	all, err := app.FindRecordsByFilter("matches",
		"tournament = {:t}", "num", 0, 0, map[string]any{"t": tournament.Id})
	if err != nil {
		return nil, nil, nil, err
	}
	koList := make([]*core.Record, 0, len(all))
	for _, m := range all {
		if st.IsKnockout(m.GetString("stage")) {
			koList = append(koList, m)
		}
	}
	koByNum := map[int]*core.Record{}
	for _, m := range koList {
		if n := m.GetInt("num"); n > 0 {
			koByNum[n] = m
		}
	}
	// Process feeders before dependents: by stage, then match number.
	sort.SliceStable(koList, func(i, j int) bool {
		si, sj := stageRank[koList[i].GetString("stage")], stageRank[koList[j].GetString("stage")]
		if si != sj {
			return si < sj
		}
		return koList[i].GetInt("num") < koList[j].GetInt("num")
	})

	slotPrefix := ""
	if eq != nil {
		slotPrefix = strconv.Itoa(eq.FromPosition)
	}
	bracket = map[string]string{}
	r := &fcResolver{
		order:      order,
		thirdByNum: assignThirds(koList, thirds, st),
		bracket:    bracket,
		ko:         koByNum,
		slotPrefix: slotPrefix,
	}
	for _, m := range koList {
		h := r.resolve(m.GetString("homeLabel"), m.GetInt("num"), map[int]bool{})
		a := r.resolve(m.GetString("awayLabel"), m.GetInt("num"), map[int]bool{})
		var pick string
		switch {
		case h != "" && a != "":
			if rng.Intn(2) == 0 {
				pick = h
			} else {
				pick = a
			}
		case h != "":
			pick = h
		case a != "":
			pick = a
		}
		if pick != "" {
			bracket[koStableKey(m)] = pick
		}
	}
	return order, thirds, bracket, nil
}
