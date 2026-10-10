package scoring

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/forecast"
)

// Row is one player's standing in a League.
type Row struct {
	UserID         string `json:"userId"`
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`       // file name in the users.avatar field; "" => none
	AvatarPreset   string `json:"avatarPreset"` // drawn picture when there is no photo
	Role           string `json:"role"`         // "admin" | "bot"; empty => normal member
	Total          int    `json:"total"`
	TipsPoints     int    `json:"tipsPoints"`
	ForecastPoints int    `json:"forecastPoints"`
	Predicted      int    `json:"predicted"` // # matches the user has tipped
	// Tiebreakers (also returned for transparency).
	ExactScores    int `json:"exactScores"`
	CorrectWinners int `json:"correctWinners"`
	GdDeviation    int `json:"gdDeviation"`
	// Forecast correct-pick counts (groups/advance/champion + R32..FINAL).
	Forecast map[string]int `json:"forecast"`
	lastEdit string         // earliest-wins; not serialized
}

// Leaderboard builds a League's standings for one tournament using the
// league's scoring config and the agreed tiebreakers: points → #exact →
// #correct winners → smaller aggregate goal-difference deviation → fewer
// tips submitted → earliest last edit. Users who never submitted a tip are
// sorted to the bottom regardless. Leagues are persistent across
// tournaments, so every standing is per-(league, tournament).
//
// Note: the sort order below is hardcoded — the scoring_configs.tiebreakers
// list is consumed only by the frontend legend for display. Keep the two in
// sync when changing tiebreakers (update this function, the seeded default
// in internal/seed, and add a migration for existing DBs).
//
// A pool with a startDate counts only from then on (see BoardSince).
func Leaderboard(app core.App, leagueID string, tournamentIDs []string) (map[string]any, error) {
	league, err := app.FindRecordById("pools", leagueID)
	if err != nil {
		return nil, err
	}
	cfgID := league.GetString("scoringConfig")
	members, err := app.FindRecordsByFilter("pool_members",
		"pool = {:l}", "", 0, 0, map[string]any{"l": leagueID})
	if err != nil {
		return nil, err
	}
	userIDs := make([]string, 0, len(members))
	for _, m := range members {
		userIDs = append(userIDs, m.GetString("user"))
	}
	rows := BoardSince(app, userIDs, cfgID, tournamentIDs, league.GetDateTime("startDate").Time())
	return map[string]any{
		"pool": map[string]any{"id": league.Id, "name": league.GetString("name")},
		"rows": rows,
	}, nil
}

// PoolTournaments returns the seasons a pool's board counts: its bound
// seasons, else the fallback (the current tournament — Global has none).
func PoolTournaments(app core.App, leagueID, fallback string) []string {
	if lg, err := app.FindRecordById("pools", leagueID); err == nil {
		if b := lg.GetStringSlice("tournaments"); len(b) > 0 {
			return b
		}
	}
	return []string{fallback}
}

// Board ranks a set of users over one or more tournaments (a pool's bound
// seasons summed, or a single one) under a scoring config ("" = default).
// Shared by pool leaderboards and the friends board.
func Board(app core.App, userIDs []string, cfgID string, tournamentIDs []string) []Row {
	return BoardSince(app, userIDs, cfgID, tournamentIDs, time.Time{})
}

// BoardSince is Board counting only matches that kick off at or after
// since (zero = all). A season whose Forecast closed before since leaves
// its Forecast out: whoever joined after the start never had one to make.
func BoardSince(app core.App, userIDs []string, cfgID string, tournamentIDs []string, since time.Time) []Row {
	if cfgID == "" {
		if def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true"); err == nil {
			cfgID = def.Id
		}
	}
	rows := make([]Row, 0, len(userIDs))
	for _, uid := range userIDs {
		u, err := app.FindRecordById("users", uid)
		if err != nil {
			continue
		}
		row := Row{UserID: uid, Name: u.GetString("name"), Avatar: u.GetString("avatar"), AvatarPreset: u.GetString("avatarPreset"), Role: u.GetString("role")}
		for _, tournamentID := range tournamentIDs {
			addTournament(app, &row, uid, cfgID, tournamentID, since, forecastCounts(app, tournamentID, since))
		}
		row.Total = row.TipsPoints + row.ForecastPoints
		rows = append(rows, row)
	}
	sortRows(rows)
	return rows
}

// forecastCounts reports whether a season's Forecast still counts on a
// board starting at since: only when since is not after the season's start.
func forecastCounts(app core.App, tournamentID string, since time.Time) bool {
	if since.IsZero() {
		return true
	}
	t, err := app.FindRecordById("tournaments", tournamentID)
	if err != nil {
		return true
	}
	start, err := forecast.StartOf(app, t)
	return err != nil || start.IsZero() || !since.After(start)
}

// addTournament accumulates one tournament's tips, forecast and tiebreak
// counters onto the row. Forecast breakdown counters sum across seasons.
// Matches kicking off before since are left out (zero = none).
func addTournament(app core.App, row *Row, uid, cfgID, tournamentID string, since time.Time, withForecast bool) {
	sinceFilter := ""
	params := map[string]any{"u": uid, "c": cfgID, "t": tournamentID}
	if !since.IsZero() {
		sinceFilter = " && match.kickoff >= {:s}"
		params["s"] = since.UTC().Format("2006-01-02 15:04:05.000Z")
	}
	ms, _ := app.FindRecordsByFilter("match_scores",
		"user = {:u} && config = {:c} && match.tournament = {:t}"+sinceFilter, "", 0, 0, params)
	for _, s := range ms {
		row.TipsPoints += s.GetInt("points")
		var comp tipComponents
		_ = json.Unmarshal([]byte(s.GetString("components")), &comp)
		if comp.Exact > 0 {
			row.ExactScores++
		}
		if comp.Tendency > 0 {
			row.CorrectWinners++
		}
		row.GdDeviation += comp.GdDev
	}

	if withForecast {
		addForecast(app, row, uid, cfgID, tournamentID)
	}

	if tps, _ := app.FindRecordsByFilter("tips",
		"user = {:u} && match.tournament = {:t}"+sinceFilter, "", 0, 0, params); len(tps) > 0 {
		row.Predicted += len(tps)
		// Earliest last-edit across this user's tips (earlier = better).
		for _, t := range tps {
			if u := t.GetString("updated"); row.lastEdit == "" || u > row.lastEdit {
				row.lastEdit = u
			}
		}
	}
}

// addForecast adds one season's Forecast points and correct-pick counters.
func addForecast(app core.App, row *Row, uid, cfgID, tournamentID string) {
	fs, err := app.FindFirstRecordByFilter("forecast_scores",
		"user = {:u} && config = {:c} && tournament = {:t}",
		map[string]any{"u": uid, "c": cfgID, "t": tournamentID})
	if err != nil {
		return
	}
	row.ForecastPoints += fs.GetInt("points")
	var bd struct {
		GroupsCorrect   int            `json:"groupsCorrect"`
		AdvanceCorrect  int            `json:"advanceCorrect"`
		RoundCorrect    map[string]int `json:"roundCorrect"`
		ChampionCorrect int            `json:"championCorrect"`
		CallCorrect     map[string]int `json:"callCorrect"`
	}
	if json.Unmarshal([]byte(fs.GetString("breakdown")), &bd) == nil {
		if row.Forecast == nil {
			row.Forecast = map[string]int{}
		}
		row.Forecast["groups"] += bd.GroupsCorrect
		row.Forecast["advance"] += bd.AdvanceCorrect
		row.Forecast["champion"] += bd.ChampionCorrect
		for k, v := range bd.RoundCorrect {
			row.Forecast[k] += v
		}
		for k, v := range bd.CallCorrect {
			row.Forecast["call:"+k] += v
		}
	}
}

func sortRows(rows []Row) {
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		aNone, bNone := a.Predicted == 0, b.Predicted == 0
		if aNone != bNone {
			return !aNone
		}
		if a.Total != b.Total {
			return a.Total > b.Total
		}
		if a.ExactScores != b.ExactScores {
			return a.ExactScores > b.ExactScores
		}
		if a.CorrectWinners != b.CorrectWinners {
			return a.CorrectWinners > b.CorrectWinners
		}
		if a.GdDeviation != b.GdDeviation {
			return a.GdDeviation < b.GdDeviation
		}
		if a.Predicted != b.Predicted {
			return a.Predicted < b.Predicted
		}
		return a.lastEdit < b.lastEdit
	})
}
