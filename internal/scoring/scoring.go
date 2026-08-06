// Package scoring computes match (Tip) and tournament (Forecast) points from
// a per-League scoring config, recomputes on every result change, and builds
// League leaderboards with the agreed tiebreakers. All tournament shape
// (group size, qualifier rules, stage list) comes from the tournament's
// structure — nothing here is specific to one event.
//
// Scale is tiny (friends app: a handful of users, ~100 matches per
// tournament), so every result change triggers a full, idempotent recompute —
// simplest and always correct.
package scoring

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/bracket"
	"github.com/floholz/matchowl/internal/tournaments"
)

// ---- Config ----

type Config struct {
	Match struct {
		Tendency   int `json:"tendency"`   // correct result (group 1/X/2; KO = who advances)
		Exact      int `json:"exact"`      // exact reference score
		TotalGoals int `json:"totalGoals"` // correct total goals
		GoalDiff   int `json:"goalDiff"`   // correct goal difference
	} `json:"match"`
	Forecast struct {
		GroupPosition     int            `json:"groupPosition"`     // per exact final position
		PerfectGroupBonus int            `json:"perfectGroupBonus"` // whole group perfect
		Advance           int            `json:"advance"`           // per predicted advancer that advances
		Round             map[string]int `json:"round"`             // predicted team reaching a KO round
	} `json:"forecast"`
}

func loadConfig(rec *core.Record) Config {
	var c Config
	_ = json.Unmarshal([]byte(rec.GetString("config")), &c)
	// Backward-compat default for configs predating the "advance" rule.
	if c.Forecast.Advance == 0 {
		c.Forecast.Advance = 1
	}
	return c
}

// configsInUse returns every scoring config referenced by a League plus the
// default, so per-(user,match,config) scores cover all Leagues.
func configsInUse(app core.App) (map[string]Config, string, error) {
	out := map[string]Config{}
	def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
	if err != nil {
		return nil, "", err
	}
	out[def.Id] = loadConfig(def)
	leagues, err := app.FindRecordsByFilter("leagues", "id != ''", "", 0, 0)
	if err != nil {
		return nil, "", err
	}
	for _, l := range leagues {
		cid := l.GetString("scoringConfig")
		if _, done := out[cid]; cid == "" || done {
			continue
		}
		if cr, err := app.FindRecordById("scoring_configs", cid); err == nil {
			out[cid] = loadConfig(cr)
		}
	}
	return out, def.Id, nil
}

func sign(n int) int {
	if n > 0 {
		return 1
	}
	if n < 0 {
		return -1
	}
	return 0
}

// ---- Match (Tip) scoring ----

type tipComponents struct {
	Tendency   int `json:"tendency"` // correct result / who advances
	Exact      int `json:"exact"`
	TotalGoals int `json:"totalGoals"`
	GoalDiff   int `json:"goalDiff"`
	GdDev      int `json:"gdDev"` // |predicted GD - actual GD| (tiebreaker only)
}

// points — max 6 per game (3 + 1 + 1 + 1).
func (c tipComponents) points() int {
	return c.Tendency + c.Exact + c.TotalGoals + c.GoalDiff
}

// MaxMatchPoints returns the maximum points a single Tip can earn under cfg
// (i.e. a "perfect" tip: correct result + exact reference score).
func (c Config) MaxMatchPoints() int {
	return c.Match.Tendency + c.Match.Exact + c.Match.TotalGoals + c.Match.GoalDiff
}

// DefaultConfig returns the loaded default scoring config.
func DefaultConfig(app core.App) (Config, error) {
	def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
	if err != nil {
		return Config{}, err
	}
	return loadConfig(def), nil
}

// ScoreTip returns the points a Tip earns for a finished match under cfg. It
// uses the same reference-score rule as the leaderboard, so knockout games
// scored in extra time are compared against the after-ET score. knockout
// comes from the tournament structure (see tournaments.StructureCache).
func ScoreTip(cfg Config, knockout bool, match, tip *core.Record) int {
	return scoreTip(cfg, knockout, match, tip).points()
}

// MatchResult / TipPrediction are the plain inputs to the pure scorer, so the
// rules are unit-testable without a database.
type MatchResult struct {
	Knockout bool
	FtH, FtA int
	EtH, EtA int
	Advancer string
}
type TipPrediction struct {
	FtH, FtA int
	EtH, EtA int
	Advancer string
}

// scoreValues is the pure scoring core (see scoring_test.go). Max 6 per game:
//   - "correct result" (Tendency): group = 1/X/2 on 90'; knockout = the team
//     that advances (no draw outcome).
//   - exact / total goals / goal difference (1 each) compare the reference
//     score: 90' for group and KO decided in 90'; the after-extra-time score
//     when a KO goes to extra time (using the user's ET prediction if they
//     predicted a 90' draw, else their decisive 90' prediction).
func scoreValues(cfg Config, m MatchResult, p TipPrediction) tipComponents {
	var r tipComponents

	// Reference scores for the accuracy components.
	aH, aA := m.FtH, m.FtA
	pH, pA := p.FtH, p.FtA
	if m.Knockout {
		wentET := m.EtH != 0 || m.EtA != 0
		if wentET {
			aH, aA = m.EtH, m.EtA
			if p.FtH == p.FtA { // user foresaw a draw -> use their ET guess
				pH, pA = p.EtH, p.EtA
			}
		}
	}

	if !m.Knockout {
		if sign(p.FtH-p.FtA) == sign(m.FtH-m.FtA) {
			r.Tendency = cfg.Match.Tendency
		}
	} else if m.Advancer != "" && m.Advancer == p.Advancer {
		r.Tendency = cfg.Match.Tendency
	}

	if pH == aH && pA == aA {
		r.Exact = cfg.Match.Exact
	}
	if pH+pA == aH+aA {
		r.TotalGoals = cfg.Match.TotalGoals
	}
	if pH-pA == aH-aA {
		r.GoalDiff = cfg.Match.GoalDiff
	}
	if d := (pH - pA) - (aH - aA); d < 0 {
		r.GdDev = -d
	} else {
		r.GdDev = d
	}
	return r
}

func scoreTip(cfg Config, knockout bool, match, tip *core.Record) tipComponents {
	return scoreValues(cfg,
		MatchResult{
			Knockout: knockout,
			FtH:      match.GetInt("ftHome"),
			FtA:      match.GetInt("ftAway"),
			EtH:      match.GetInt("etHome"),
			EtA:      match.GetInt("etAway"),
			Advancer: match.GetString("advancer"),
		},
		TipPrediction{
			FtH:      tip.GetInt("ftHome"),
			FtA:      tip.GetInt("ftAway"),
			EtH:      tip.GetInt("etHome"),
			EtA:      tip.GetInt("etAway"),
			Advancer: tip.GetString("advancer"),
		},
	)
}

// ---- Group standings (final, from finalized group matches) ----

type teamAgg struct {
	id                 string
	pts, gd, gf, games int
}

// finalGroups returns, for each fully-finished group of the tournament, the
// ordered team ids (1st..groupSize) and collects each group's
// extra-qualifier-position team for the qualifier rank.
func finalGroups(app core.App, tournamentID string, st *tournaments.Structure) (order map[string][]string, thirds []teamAgg) {
	order = map[string][]string{}
	if !st.HasGroups() {
		return order, nil
	}
	ms, _ := app.FindRecordsByFilter("matches",
		"tournament = {:t} && stage = {:g} && finalizedAt != ''", "", 0, 0,
		map[string]any{"t": tournamentID, "g": st.GroupStage().Code})
	groups := map[string]map[string]*teamAgg{}
	for _, m := range ms {
		g := m.GetString("groupLetter")
		if groups[g] == nil {
			groups[g] = map[string]*teamAgg{}
		}
		h, a := m.GetString("homeTeam"), m.GetString("awayTeam")
		hg, ag := m.GetInt("ftHome"), m.GetInt("ftAway")
		for _, id := range []string{h, a} {
			if groups[g][id] == nil {
				groups[g][id] = &teamAgg{id: id}
			}
		}
		H, A := groups[g][h], groups[g][a]
		H.games++
		A.games++
		H.gf += hg
		A.gf += ag
		H.gd += hg - ag
		A.gd += ag - hg
		switch {
		case hg > ag:
			H.pts += st.PointsWin
		case ag > hg:
			A.pts += st.PointsWin
		default:
			H.pts += st.PointsDraw
			A.pts += st.PointsDraw
		}
	}
	for g, tbl := range groups {
		if len(tbl) < st.GroupSize {
			continue
		}
		arr := make([]teamAgg, 0, len(tbl))
		complete := true
		for _, v := range tbl {
			arr = append(arr, *v)
			if v.games < st.GamesPerTeam {
				complete = false
			}
		}
		if !complete {
			continue
		}
		sortAggs(arr)
		ids := make([]string, len(arr))
		for i, v := range arr {
			ids[i] = v.id
		}
		order[g] = ids
		if eq := st.ExtraQualifiers; eq != nil && len(arr) >= eq.FromPosition {
			thirds = append(thirds, arr[eq.FromPosition-1])
		}
	}
	return order, thirds
}

func sortAggs(a []teamAgg) {
	sort.Slice(a, func(i, j int) bool {
		if a[i].pts != a[j].pts {
			return a[i].pts > a[j].pts
		}
		if a[i].gd != a[j].gd {
			return a[i].gd > a[j].gd
		}
		return a[i].gf > a[j].gf
	})
}

func bestThirdSet(thirds []teamAgg, count int) map[string]bool {
	sortAggs(thirds)
	set := map[string]bool{}
	for i, t := range thirds {
		if i >= count {
			break
		}
		set[t.id] = true
	}
	return set
}

// ---- Forecast scoring ----

// actualRoundTeams maps stage -> set(teamId) of teams that actually reached
// that round, plus the actual champion (winner of the structure's champion
// stage).
func actualRoundTeams(app core.App, tournamentID string, st *tournaments.Structure) (map[string]map[string]bool, string) {
	res := map[string]map[string]bool{}
	champion := ""
	champCode := ""
	if c := st.ChampionStage(); c != nil {
		champCode = c.Code
	}
	ms, _ := app.FindRecordsByFilter("matches",
		"tournament = {:t}", "num", 0, 0, map[string]any{"t": tournamentID})
	for _, m := range ms {
		stage := m.GetString("stage")
		if !st.IsKnockout(stage) {
			continue
		}
		if res[stage] == nil {
			res[stage] = map[string]bool{}
		}
		for _, f := range []string{"homeTeam", "awayTeam"} {
			if id := m.GetString(f); id != "" {
				res[stage][id] = true
			}
		}
		if stage == champCode && m.GetString("finalizedAt") != "" {
			champion = m.GetString("advancer")
		}
	}
	return res, champion
}

type fcResolver struct {
	order      map[string][]string
	thirdByNum map[int]string // slot match num -> chosen extra-qualifier teamId
	bracket    map[string]string
	ko         map[int]*core.Record
	slotPrefix string // extra-qualifier position digit ("" = no extra qualifiers)
}

// assignThirds maps the user's chosen extra qualifiers ({groupLetter:
// teamId}) onto the bracket's qualifier slots. It uses the structure's
// official allocation table (e.g. FIFA Annex C) for the given qualifying-
// group combination; without a registered table (or an off-table
// combination) it falls back to a deterministic backtracking matching.
// Identical logic on the frontend so the predicted Forecast bracket and its
// scoring always agree.
func assignThirds(koList []*core.Record, thirds map[string]string, st *tournaments.Structure) map[int]string {
	eq := st.ExtraQualifiers
	if eq == nil {
		return map[int]string{}
	}
	slotPrefix := strconv.Itoa(eq.FromPosition)
	type slot struct {
		num     int
		winner  string
		allowed []string
	}
	var slots []slot
	for _, mt := range koList {
		home, away := mt.GetString("homeLabel"), mt.GetString("awayLabel")
		for _, lbl := range []string{home, away} {
			if strings.HasPrefix(lbl, slotPrefix) && strings.Contains(lbl, "/") {
				w, _ := bracket.WinnerLetter(home, away)
				slots = append(slots, slot{
					num:     mt.GetInt("num"),
					winner:  w,
					allowed: strings.Split(strings.TrimPrefix(lbl, slotPrefix), "/"),
				})
			}
		}
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i].num < slots[j].num })

	chosen := make([]string, 0, len(thirds))
	for letter := range thirds {
		chosen = append(chosen, letter)
	}
	sort.Strings(chosen)

	// Official table for this exact set of qualifying groups.
	if m, ok := bracket.Lookup(eq.TableKey, chosen); ok {
		out := map[int]string{}
		for _, s := range slots {
			if g, ok := m[s.winner]; ok {
				out[s.num] = thirds[g]
			}
		}
		return out
	}

	// Fallback: deterministic backtracking perfect matching.
	assign := make([]string, len(slots))
	var solve func(i int) bool
	solve = func(i int) bool {
		if i == len(slots) {
			return true
		}
		for _, letter := range chosen {
			taken := false
			for _, a := range assign {
				if a == letter {
					taken = true
					break
				}
			}
			if taken {
				continue
			}
			allowed := false
			for _, a := range slots[i].allowed {
				if a == letter {
					allowed = true
					break
				}
			}
			if !allowed {
				continue
			}
			assign[i] = letter
			if solve(i + 1) {
				return true
			}
			assign[i] = ""
		}
		return false
	}
	solve(0)

	out := map[int]string{}
	for i, s := range slots {
		if assign[i] != "" {
			out[s.num] = thirds[assign[i]]
		}
	}
	return out
}

func (r *fcResolver) resolve(label string, forNum int, seen map[int]bool) string {
	if label == "" {
		return ""
	}
	switch c := label[0]; {
	case c >= '1' && c <= '9':
		// Extra-qualifier slot ("3A/B/C/D/F") vs plain group position ("1A").
		if r.slotPrefix != "" && strings.HasPrefix(label, r.slotPrefix) && strings.Contains(label, "/") {
			return r.thirdByNum[forNum]
		}
		idx := int(c - '1')
		o := r.order[label[1:]]
		if len(o) > idx {
			return o[idx]
		}
		return ""
	case c == 'W' || c == 'L':
		n, _ := strconv.Atoi(label[1:])
		if seen[n] {
			return ""
		}
		seen[n] = true
		w := r.bracket[strconv.Itoa(n)]
		if c == 'W' {
			return w
		}
		src := r.ko[n]
		if src == nil || w == "" {
			return ""
		}
		h := r.resolve(src.GetString("homeLabel"), n, seen)
		a := r.resolve(src.GetString("awayLabel"), n, seen)
		if w == h {
			return a
		}
		if w == a {
			return h
		}
		return ""
	}
	return ""
}

func koStableKey(m *core.Record) string {
	if n := m.GetInt("num"); n > 0 {
		return strconv.Itoa(n)
	}
	return m.GetString("stage")
}

type fcBreakdown struct {
	// Points.
	Groups   int `json:"groups"`   // exact final positions (+ perfect bonus)
	Advance  int `json:"advance"`  // predicted advancers that actually advanced
	Knockout int `json:"knockout"` // predicted teams reaching KO rounds
	Champion int `json:"champion"`
	// Correct-pick counts (for the Forecast leaderboard view).
	GroupsCorrect   int            `json:"groupsCorrect"`
	AdvanceCorrect  int            `json:"advanceCorrect"`
	RoundCorrect    map[string]int `json:"roundCorrect"` // R32..FINAL
	ChampionCorrect int            `json:"championCorrect"`
}

func (b fcBreakdown) total() int {
	return b.Groups + b.Advance + b.Knockout + b.Champion
}

func scoreForecast(app core.App, cfg Config, fc *core.Record) (fcBreakdown, int) {
	b := fcBreakdown{RoundCorrect: map[string]int{}}

	trec, err := app.FindRecordById("tournaments", fc.GetString("tournament"))
	if err != nil {
		return b, 0
	}
	st, err := tournaments.StructureOf(trec)
	if err != nil {
		return b, 0
	}
	eq := st.ExtraQualifiers

	var order map[string][]string
	_ = fc.UnmarshalJSONField("groupOrder", &order)
	var thirds map[string]string
	_ = fc.UnmarshalJSONField("thirdQualifiers", &thirds)
	var bracket map[string]string
	_ = fc.UnmarshalJSONField("bracket", &bracket)

	actualOrder, thirdAggs := finalGroups(app, trec.Id, st)
	for g, actual := range actualOrder {
		pred := order[g]
		allCorrect := len(pred) == st.GroupSize
		for i := 0; i < st.GroupSize && i < len(actual); i++ {
			if i < len(pred) && pred[i] == actual[i] {
				b.Groups += cfg.Forecast.GroupPosition
				b.GroupsCorrect++
			} else {
				allCorrect = false
			}
		}
		if allCorrect {
			b.Groups += cfg.Forecast.PerfectGroupBonus
		}
	}

	// Advancement: +Advance for each predicted advancer (a group's direct
	// qualifiers, or one of the user's extra-qualifier picks) that actually
	// advances.
	best := map[string]bool{}
	if eq != nil {
		groupCount, _ := app.CountRecords("tournament_groups",
			dbx.HashExp{"tournament": trec.Id})
		if groupCount > 0 && len(thirdAggs) >= int(groupCount) { // all groups done -> qualifier set fixed
			best = bestThirdSet(thirdAggs, eq.Count)
		}
	}
	actualAdv := map[string]bool{}
	for _, actual := range actualOrder {
		for i := 0; i < st.DirectQualifiers && i < len(actual); i++ {
			actualAdv[actual[i]] = true
		}
	}
	for id := range best {
		actualAdv[id] = true
	}
	for g, pred := range order {
		for i := 0; i < st.DirectQualifiers && i < len(pred); i++ {
			if actualAdv[pred[i]] {
				b.Advance += cfg.Forecast.Advance
				b.AdvanceCorrect++
			}
		}
		// The extra-qualifier pick only counts if the user chose this group
		// as one of their qualifying groups.
		if eq != nil && len(pred) >= eq.FromPosition && thirds[g] != "" && actualAdv[pred[eq.FromPosition-1]] {
			b.Advance += cfg.Forecast.Advance
			b.AdvanceCorrect++
		}
	}

	actualRounds, actualChamp := actualRoundTeams(app, trec.Id, st)
	all, _ := app.FindRecordsByFilter("matches",
		"tournament = {:t}", "num", 0, 0, map[string]any{"t": trec.Id})
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
	slotPrefix := ""
	if eq != nil {
		slotPrefix = strconv.Itoa(eq.FromPosition)
	}
	r := &fcResolver{
		order:      order,
		thirdByNum: assignThirds(koList, thirds, st),
		bracket:    bracket,
		ko:         koByNum,
		slotPrefix: slotPrefix,
	}

	for _, m := range koList {
		stage := m.GetString("stage")
		w := cfg.Forecast.Round[stage]
		if w == 0 {
			continue
		}
		predHome := r.resolve(m.GetString("homeLabel"), m.GetInt("num"), map[int]bool{})
		predAway := r.resolve(m.GetString("awayLabel"), m.GetInt("num"), map[int]bool{})
		for _, pid := range []string{predHome, predAway} {
			if pid != "" && actualRounds[stage] != nil && actualRounds[stage][pid] {
				b.Knockout += w
				b.RoundCorrect[stage]++
			}
		}
	}

	if actualChamp != "" {
		champCode := ""
		if c := st.ChampionStage(); c != nil {
			champCode = c.Code
		}
		var champKey string
		for _, m := range koList {
			if champCode != "" && m.GetString("stage") == champCode {
				champKey = koStableKey(m)
			}
		}
		if champKey != "" && bracket[champKey] == actualChamp {
			b.Champion += cfg.Forecast.Round["CHAMPION"]
			b.ChampionCorrect = 1
		}
	}

	return b, b.total()
}
