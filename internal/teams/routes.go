package teams

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/tournaments"
)

// Routes wires the team page, following and search endpoints:
//
//	GET    /api/teams/mine          the teams I follow, each with its next match
//	GET    /api/teams/{key}         a team's page header and its seasons
//	POST   /api/teams/{key}/follow  follow
//	DELETE /api/teams/{key}/follow  unfollow
//	GET    /api/search?q=           competitions and teams
//
// A team's matches come from /api/feed?team={key}.
func Routes(app core.App, se *core.ServeEvent) {
	g := se.Router.Group("/api/teams")
	g.Bind(apis.RequireAuth())

	g.GET("/mine", func(e *core.RequestEvent) error {
		follows, err := app.FindRecordsByFilter(Follows, "user = {:u}", "created", 0, 0, map[string]any{"u": e.Auth.Id})
		if err != nil {
			return err
		}
		now := clock.Now(app)
		out := make([]map[string]any, 0, len(follows))
		for _, f := range follows {
			id, err := app.FindRecordById(Collection, f.GetString("team"))
			if err != nil {
				continue
			}
			v := view(app, id)
			v["next"] = nextMatch(app, id, now)
			out = append(out, v)
		}
		// Soonest next match first; teams without one at the end.
		sort.SliceStable(out, func(i, j int) bool {
			a, _ := out[i]["next"].(map[string]any)
			b, _ := out[j]["next"].(map[string]any)
			switch {
			case a == nil:
				return false
			case b == nil:
				return true
			}
			return a["kickoff"].(string) < b["kickoff"].(string)
		})
		return e.JSON(http.StatusOK, map[string]any{"teams": out})
	})

	g.GET("/{key}", func(e *core.RequestEvent) error {
		id, err := byKey(app, e.Request.PathValue("key"))
		if err != nil {
			return apis.NewNotFoundError("no such team", nil)
		}
		following := false
		if _, err := app.FindFirstRecordByFilter(Follows, "user = {:u} && team = {:t}",
			map[string]any{"u": e.Auth.Id, "t": id.Id}); err == nil {
			following = true
		}
		followers, _ := app.CountRecords(Follows, dbx.HashExp{"team": id.Id})
		return e.JSON(http.StatusOK, map[string]any{
			"team":      view(app, id),
			"following": following,
			"followers": followers,
			"seasons":   seasons(app, id),
		})
	})

	g.POST("/{key}/follow", func(e *core.RequestEvent) error {
		id, err := byKey(app, e.Request.PathValue("key"))
		if err != nil {
			return apis.NewNotFoundError("no such team", nil)
		}
		if _, err := app.FindFirstRecordByFilter(Follows, "user = {:u} && team = {:t}",
			map[string]any{"u": e.Auth.Id, "t": id.Id}); err == nil {
			return e.JSON(http.StatusOK, map[string]any{"following": true})
		}
		col, err := app.FindCollectionByNameOrId(Follows)
		if err != nil {
			return err
		}
		rec := core.NewRecord(col)
		rec.Set("user", e.Auth.Id)
		rec.Set("team", id.Id)
		if err := app.Save(rec); err != nil {
			return err
		}
		return e.JSON(http.StatusOK, map[string]any{"following": true})
	})

	g.DELETE("/{key}/follow", func(e *core.RequestEvent) error {
		id, err := byKey(app, e.Request.PathValue("key"))
		if err != nil {
			return apis.NewNotFoundError("no such team", nil)
		}
		if f, err := app.FindFirstRecordByFilter(Follows, "user = {:u} && team = {:t}",
			map[string]any{"u": e.Auth.Id, "t": id.Id}); err == nil {
			if err := app.Delete(f); err != nil {
				return err
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"following": false})
	})

	se.Router.GET("/api/search", func(e *core.RequestEvent) error {
		q := strings.TrimSpace(e.Request.URL.Query().Get("q"))
		comps := []map[string]any{}
		teams := []map[string]any{}
		if len([]rune(q)) < 2 {
			return e.JSON(http.StatusOK, map[string]any{"competitions": comps, "teams": teams})
		}
		like := map[string]any{"q": q}
		if recs, err := app.FindRecordsByFilter("competitions",
			"name ~ {:q} || shortName ~ {:q} || country ~ {:q}", "name", 8, 0, like); err == nil {
			for _, c := range recs {
				if !hasVisibleSeason(app, c.Id) {
					continue
				}
				v := map[string]any{
					"key": c.GetString("key"), "name": c.GetString("name"),
					"shortName": c.GetString("shortName"), "country": c.GetString("country"), "logo": "",
				}
				if l := c.GetString("logo"); l != "" {
					v["logo"] = "competitions/" + c.Id + "/" + l
				}
				comps = append(comps, v)
			}
		}
		if recs, err := app.FindRecordsByFilter(Collection,
			"name ~ {:q} || fifaCode = {:q}", "name", 40, 0, like); err == nil {
			lq := strings.ToLower(q)
			// Names that start with the query first.
			sort.SliceStable(recs, func(i, j int) bool {
				a := strings.HasPrefix(strings.ToLower(recs[i].GetString("name")), lq)
				b := strings.HasPrefix(strings.ToLower(recs[j].GetString("name")), lq)
				return a && !b
			})
			for _, r := range recs {
				if len(teams) == 20 {
					break
				}
				teams = append(teams, view(app, r))
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"competitions": comps, "teams": teams})
	}).Bind(apis.RequireAuth())
}

func byKey(app core.App, key string) (*core.Record, error) {
	return app.FindFirstRecordByFilter(Collection, "key = {:k}", map[string]any{"k": key})
}

// view is what every endpoint shows of a team.
func view(app core.App, id *core.Record) map[string]any {
	return map[string]any{
		"id":          id.Id,
		"key":         id.GetString("key"),
		"name":        id.GetString("name"),
		"fifaCode":    id.GetString("fifaCode"),
		"iso2":        id.GetString("iso2"),
		"national":    id.GetBool("national"),
		"country":     id.GetString("country"),
		"countryName": id.GetString("countryName"),
		"crest":       Crest(app, id.Id),
	}
}

func hasVisibleSeason(app core.App, competitionID string) bool {
	n, _ := app.CountRecords("tournaments",
		dbx.HashExp{"competition": competitionID},
		dbx.Not(dbx.HashExp{"status": tournaments.StatusDraft}))
	return n > 0
}

// nextMatch is a followed team's next (or live) match, for Home's "Your
// teams" strip; nil when none is scheduled.
func nextMatch(app core.App, id *core.Record, now time.Time) map[string]any {
	rows := SeasonRowIDs(app, []string{id.Id})
	if len(rows) == 0 {
		return nil
	}
	in := make([]any, len(rows))
	mine := map[string]bool{}
	for i, r := range rows {
		in[i] = r
		mine[r] = true
	}
	var ms []*core.Record
	// Live matches started up to 3 hours ago still count as "next".
	from := now.Add(-3 * time.Hour).UTC().Format(types.DefaultDateLayout)
	if err := app.RecordQuery("matches").
		AndWhere(dbx.Or(dbx.In("homeTeam", in...), dbx.In("awayTeam", in...))).
		AndWhere(dbx.NewExp("kickoff >= {:from}", dbx.Params{"from": from})).
		AndWhere(dbx.NotIn("status", "finished")).
		OrderBy("kickoff ASC").Limit(8).All(&ms); err != nil {
		return nil
	}
	for _, m := range ms {
		t, err := app.FindRecordById("tournaments", m.GetString("tournament"))
		if err != nil || t.GetString("status") == tournaments.StatusDraft {
			continue
		}
		home := mine[m.GetString("homeTeam")]
		oppID, label := m.GetString("awayTeam"), m.GetString("awayLabel")
		if !home {
			oppID, label = m.GetString("homeTeam"), m.GetString("homeLabel")
		}
		opp := map[string]any{"teamId": oppID, "name": label, "fifaCode": "", "iso2": "", "logo": "", "key": ""}
		if o, err := app.FindRecordById("teams", oppID); err == nil {
			opp["name"] = o.GetString("name")
			opp["fifaCode"] = o.GetString("fifaCode")
			opp["iso2"] = o.GetString("iso2")
			opp["logo"] = o.GetString("logo")
			if oi, err := app.FindRecordById(Collection, o.GetString("identity")); err == nil {
				opp["key"] = oi.GetString("key")
			}
		}
		return map[string]any{
			"id":         m.Id,
			"kickoff":    m.GetString("kickoff"),
			"status":     m.GetString("status"),
			"home":       home,
			"opponent":   opp,
			"tournament": map[string]any{"id": t.Id, "name": t.GetString("name"), "shortName": t.GetString("shortName")},
		}
	}
	return nil
}

// seasons lists the seasons a team plays in (newest first, drafts left
// out), with its row in the league or group table where there is one.
func seasons(app core.App, id *core.Record) []map[string]any {
	rows, err := app.FindRecordsByFilter("teams", "identity = {:i}", "", 0, 0, map[string]any{"i": id.Id})
	if err != nil {
		return []map[string]any{}
	}
	out := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		t, err := app.FindRecordById("tournaments", row.GetString("tournament"))
		if err != nil || t.GetString("status") == tournaments.StatusDraft {
			continue
		}
		v := map[string]any{
			"teamId": row.Id,
			"tournament": map[string]any{
				"id": t.Id, "slug": t.GetString("slug"), "name": t.GetString("name"),
				"shortName": t.GetString("shortName"), "status": t.GetString("status"),
			},
			"table": nil,
		}
		if c, err := app.FindRecordById("competitions", t.GetString("competition")); err == nil {
			v["tournament"].(map[string]any)["competition"] = c.GetString("key")
		}
		if st, err := tournaments.StructureOf(t); err == nil {
			if tbl := tableRow(app, t.Id, st, row.Id); tbl != nil {
				v["table"] = tbl
			}
		}
		out = append(out, v)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a := out[i]["tournament"].(map[string]any)
		b := out[j]["tournament"].(map[string]any)
		ra, rb := statusRank(a["status"].(string)), statusRank(b["status"].(string))
		if ra != rb {
			return ra < rb
		}
		return a["name"].(string) > b["name"].(string)
	})
	return out
}

func statusRank(s string) int {
	switch s {
	case tournaments.StatusActive:
		return 0
	case tournaments.StatusUpcoming:
		return 1
	default:
		return 2
	}
}

// tableRow is a team's line in its league or group table from the finished
// matches; nil when the season has no table stage for it.
func tableRow(app core.App, tournamentID string, st *tournaments.Structure, teamID string) map[string]any {
	var ms []*core.Record
	if err := app.RecordQuery("matches").
		AndWhere(dbx.HashExp{"tournament": tournamentID}).
		All(&ms); err != nil {
		return nil
	}
	letter, found := "", false
	var stage []Result
	for _, m := range ms {
		if st.IsKnockout(m.GetString("stage")) {
			continue
		}
		if !found && (m.GetString("homeTeam") == teamID || m.GetString("awayTeam") == teamID) {
			letter, found = m.GetString("groupLetter"), true
		}
	}
	if !found {
		return nil
	}
	for _, m := range ms {
		if st.IsKnockout(m.GetString("stage")) || m.GetString("groupLetter") != letter {
			continue
		}
		stage = append(stage, Result{
			Home: m.GetString("homeTeam"), Away: m.GetString("awayTeam"),
			HomeGoals: m.GetInt("ftHome"), AwayGoals: m.GetInt("ftAway"),
			Played: m.GetString("status") == "finished" || m.GetString("finalizedAt") != "",
		})
	}
	tbl := Table(stage)
	for i, r := range tbl {
		if r.Team == teamID {
			return map[string]any{
				"group": letter, "rank": i + 1, "size": len(tbl),
				"played": r.Played, "won": r.Won, "drawn": r.Drawn, "lost": r.Lost,
				"gf": r.GF, "ga": r.GA, "pts": r.Pts,
			}
		}
	}
	return nil
}

// Result is one match of a table stage.
type Result struct {
	Home, Away           string
	HomeGoals, AwayGoals int
	Played               bool
}

// Row is one team's line of a table.
type Row struct {
	Team                                  string
	Played, Won, Drawn, Lost, GF, GA, Pts int
}

// Table ranks the teams of a stage by points, goal difference, goals
// scored, then id (a stable last resort, not a competition's real
// tiebreak). Every team in the fixtures gets a row, played or not.
func Table(results []Result) []Row {
	rows := map[string]*Row{}
	get := func(id string) *Row {
		if rows[id] == nil {
			rows[id] = &Row{Team: id}
		}
		return rows[id]
	}
	for _, r := range results {
		if r.Home == "" || r.Away == "" {
			continue
		}
		h, a := get(r.Home), get(r.Away)
		if !r.Played {
			continue
		}
		h.Played++
		a.Played++
		h.GF += r.HomeGoals
		h.GA += r.AwayGoals
		a.GF += r.AwayGoals
		a.GA += r.HomeGoals
		switch {
		case r.HomeGoals > r.AwayGoals:
			h.Won++
			a.Lost++
			h.Pts += 3
		case r.HomeGoals < r.AwayGoals:
			a.Won++
			h.Lost++
			a.Pts += 3
		default:
			h.Drawn++
			a.Drawn++
			h.Pts++
			a.Pts++
		}
	}
	out := make([]Row, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool {
		x, y := out[i], out[j]
		if x.Pts != y.Pts {
			return x.Pts > y.Pts
		}
		if gx, gy := x.GF-x.GA, y.GF-y.GA; gx != gy {
			return gx > gy
		}
		if x.GF != y.GF {
			return x.GF > y.GF
		}
		return x.Team < y.Team
	})
	return out
}
