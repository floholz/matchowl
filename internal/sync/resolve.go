package sync

import (
	"sort"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/bracket"
	"github.com/floholz/matchowl/internal/tournaments"
)

// ResolveBracket fills knockout matches' homeTeam/awayTeam from their
// placeholder labels once the referenced results are known, for every
// tournament with a knockout stage. This is what makes a knockout Tip become
// available: a Tip opens as soon as both teams of a matchup are resolved.
//
// Resolvable labels (digit = group position per the tournament's structure):
//   - "1A".."2L"       group position (once that group is complete)
//   - "3A/B/C/D/F"     an extra-qualifier slot (position digit + candidate
//     groups); allocated via the structure's official table
//     when registered, else a greedy interim fill
//   - "W73" / "L101"   winner / loser of a finished knockout match
func ResolveBracket(app core.App) error {
	recs, err := tournaments.All(app)
	if err != nil {
		return err
	}
	for _, t := range recs {
		st, err := tournaments.StructureOf(t)
		if err != nil || len(st.KnockoutStages()) == 0 {
			continue
		}
		if err := resolveTournament(app, t.Id, st); err != nil {
			return err
		}
	}
	return nil
}

func resolveTournament(app core.App, tournamentID string, st *tournaments.Structure) error {
	matches, err := app.FindRecordsByFilter("matches",
		"tournament = {:t}", "num", 0, 0, map[string]any{"t": tournamentID})
	if err != nil {
		return err
	}

	byNum := map[int]*core.Record{}
	for _, m := range matches {
		if n := m.GetInt("num"); n > 0 {
			byNum[n] = m
		}
	}

	positions, extras, extraTeam := groupStandings(matches, st)

	// Resolve the extra-qualifier slots (WC2026: the 8 best thirds → R32).
	// With the full qualifier set known, use the structure's official table;
	// otherwise fall back to a deterministic greedy fill (only hit while the
	// group stage is still incomplete, when the bracket can't be resolved yet
	// anyway — or for tournaments without an official table).
	eq := st.ExtraQualifiers
	slotPrefix := ""
	if eq != nil {
		slotPrefix = strconv.Itoa(eq.FromPosition)
	}
	isSlotLabel := func(lbl string) bool {
		return eq != nil && strings.HasPrefix(lbl, slotPrefix) && strings.Contains(lbl, "/")
	}

	quals := make([]string, 0, len(extras))
	for _, s := range extras {
		quals = append(quals, s.group)
	}
	thirdByNum := map[int]string{}
	if eq != nil {
		if tbl, ok := bracket.Lookup(eq.TableKey, quals); ok && len(quals) == eq.Count {
			for _, m := range matches {
				home, away := m.GetString("homeLabel"), m.GetString("awayLabel")
				if !isSlotLabel(home) && !isSlotLabel(away) {
					continue
				}
				if w, ok := bracket.WinnerLetter(home, away); ok {
					thirdByNum[m.GetInt("num")] = extraTeam[tbl[w]]
				}
			}
		} else {
			thirdQueue := make([]string, len(quals))
			copy(thirdQueue, quals)
			slotted := []*core.Record{}
			for _, m := range matches {
				if isSlotLabel(m.GetString("homeLabel")) || isSlotLabel(m.GetString("awayLabel")) {
					slotted = append(slotted, m)
				}
			}
			sort.Slice(slotted, func(i, j int) bool {
				return slotted[i].GetInt("num") < slotted[j].GetInt("num")
			})
			for _, m := range slotted {
				for _, lbl := range []string{m.GetString("homeLabel"), m.GetString("awayLabel")} {
					if !isSlotLabel(lbl) {
						continue
					}
					allowed := strings.Split(strings.TrimPrefix(lbl, slotPrefix), "/")
					for i, g := range thirdQueue {
						if g == "" {
							continue
						}
						ok := false
						for _, a := range allowed {
							if g == a {
								ok = true
								break
							}
						}
						if ok {
							thirdByNum[m.GetInt("num")] = extraTeam[g]
							thirdQueue[i] = ""
							break
						}
					}
				}
			}
		}
	}

	resolve := func(label string, num int) string {
		if label == "" {
			return ""
		}
		switch c := label[0]; {
		case c >= '1' && c <= '9':
			if isSlotLabel(label) {
				return thirdByNum[num]
			}
			return positions[int(c-'0')][label[1:]]
		case c == 'W' || c == 'L':
			n, err := strconv.Atoi(label[1:])
			if err != nil {
				return ""
			}
			src, ok := byNum[n]
			if !ok || src.GetString("finalizedAt") == "" {
				return ""
			}
			adv := src.GetString("advancer")
			if c == 'W' {
				return adv
			}
			// loser = the side that is not the advancer
			h, a := src.GetString("homeTeam"), src.GetString("awayTeam")
			if adv == h {
				return a
			}
			if adv == a {
				return h
			}
			return ""
		}
		return ""
	}

	for _, m := range matches {
		if !st.IsKnockout(m.GetString("stage")) {
			continue
		}
		changed := false
		num := m.GetInt("num")
		if m.GetString("homeTeam") == "" {
			if id := resolve(m.GetString("homeLabel"), num); id != "" {
				m.Set("homeTeam", id)
				changed = true
			}
		}
		if m.GetString("awayTeam") == "" {
			if id := resolve(m.GetString("awayLabel"), num); id != "" {
				m.Set("awayTeam", id)
				changed = true
			}
		}
		if changed {
			if err := app.Save(m); err != nil {
				return err
			}
		}
	}
	return nil
}

type standing struct {
	group string
	team  string
	pts   int
	gd    int
	gf    int
}

// groupStandings computes, from finished group matches only:
//   - positions[pos][letter] = team id at that 1-based position, only once the
//     group is complete (all teams played structure.gamesPerTeam games)
//   - the globally ranked extra-qualifier candidates (the fromPosition-placed
//     team of each complete group, capped at extraQualifiers.count)
//   - extraTeam[letter] = that group's fromPosition-placed team id
func groupStandings(matches []*core.Record, st *tournaments.Structure) (positions map[int]map[string]string, extras []standing, extraTeam map[string]string) {
	positions = map[int]map[string]string{}
	extraTeam = map[string]string{}
	if !st.HasGroups() {
		return positions, nil, extraTeam
	}
	groupCode := st.GroupStage().Code

	type agg struct{ pts, gd, gf, played int }
	groups := map[string]map[string]*agg{} // letter -> teamId -> agg

	for _, m := range matches {
		if m.GetString("stage") != groupCode || m.GetString("finalizedAt") == "" {
			continue
		}
		g := m.GetString("groupLetter")
		if groups[g] == nil {
			groups[g] = map[string]*agg{}
		}
		h, a := m.GetString("homeTeam"), m.GetString("awayTeam")
		hg, ag := m.GetInt("ftHome"), m.GetInt("ftAway")
		for _, id := range []string{h, a} {
			if groups[g][id] == nil {
				groups[g][id] = &agg{}
			}
		}
		ha, aa := groups[g][h], groups[g][a]
		ha.played++
		aa.played++
		ha.gf += hg
		aa.gf += ag
		ha.gd += hg - ag
		aa.gd += ag - hg
		switch {
		case hg > ag:
			ha.pts += st.PointsWin
		case ag > hg:
			aa.pts += st.PointsWin
		default:
			ha.pts += st.PointsDraw
			aa.pts += st.PointsDraw
		}
	}

	for g, tbl := range groups {
		order := make([]standing, 0, len(tbl))
		complete := true
		for id, v := range tbl {
			order = append(order, standing{group: g, team: id, pts: v.pts, gd: v.gd, gf: v.gf})
			if v.played < st.GamesPerTeam {
				complete = false
			}
		}
		if len(tbl) < st.GroupSize {
			complete = false
		}
		sort.Slice(order, func(i, j int) bool {
			if order[i].pts != order[j].pts {
				return order[i].pts > order[j].pts
			}
			if order[i].gd != order[j].gd {
				return order[i].gd > order[j].gd
			}
			return order[i].gf > order[j].gf
		})
		if !complete {
			continue
		}
		for pos, s := range order {
			if positions[pos+1] == nil {
				positions[pos+1] = map[string]string{}
			}
			positions[pos+1][g] = s.team
		}
		if eq := st.ExtraQualifiers; eq != nil && len(order) >= eq.FromPosition {
			extras = append(extras, order[eq.FromPosition-1])
			extraTeam[g] = order[eq.FromPosition-1].team
		}
	}

	sort.Slice(extras, func(i, j int) bool {
		if extras[i].pts != extras[j].pts {
			return extras[i].pts > extras[j].pts
		}
		if extras[i].gd != extras[j].gd {
			return extras[i].gd > extras[j].gd
		}
		return extras[i].gf > extras[j].gf
	})
	if eq := st.ExtraQualifiers; eq != nil && len(extras) > eq.Count {
		extras = extras[:eq.Count]
	}
	return positions, extras, extraTeam
}
