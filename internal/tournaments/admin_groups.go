package tournaments

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

// The admin group editor (2026-09-28, plan 06's lesson): move teams
// between groups, add or remove groups, and have every group-stage match
// follow — the one operation the Nations League repair needed by hand.

type groupTeamView struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FifaCode string `json:"fifaCode"`
	ISO2     string `json:"iso2"`
	Logo     string `json:"logo"`
}

type groupView struct {
	ID     string          `json:"id"`
	Letter string          `json:"letter"`
	Teams  []groupTeamView `json:"teams"`
}

// groupsView is what the editor loads: the groups with their teams, the
// teams in no group, and the structure numbers the groups imply.
func groupsView(app core.App, t *core.Record) (map[string]any, error) {
	teams, err := app.FindRecordsByFilter("teams", "tournament = {:t}", "name", 0, 0, map[string]any{"t": t.Id})
	if err != nil {
		return nil, err
	}
	tv := map[string]groupTeamView{}
	for _, tm := range teams {
		tv[tm.Id] = groupTeamView{
			ID: tm.Id, Name: tm.GetString("name"), FifaCode: tm.GetString("fifaCode"),
			ISO2: tm.GetString("iso2"), Logo: tm.GetString("logo"),
		}
	}
	groups, err := app.FindRecordsByFilter("tournament_groups", "tournament = {:t}", "letter", 0, 0, map[string]any{"t": t.Id})
	if err != nil {
		return nil, err
	}
	placed := map[string]bool{}
	out := make([]groupView, 0, len(groups))
	for _, g := range groups {
		gv := groupView{ID: g.Id, Letter: g.GetString("letter"), Teams: []groupTeamView{}}
		for _, id := range g.GetStringSlice("teams") {
			if v, ok := tv[id]; ok {
				gv.Teams = append(gv.Teams, v)
				placed[id] = true
			}
		}
		out = append(out, gv)
	}
	unassigned := []groupTeamView{}
	for _, tm := range teams {
		if !placed[tm.Id] {
			unassigned = append(unassigned, tv[tm.Id])
		}
	}
	st, _ := StructureOf(t)
	var structure map[string]any
	if st != nil {
		structure = map[string]any{"groupSize": st.GroupSize, "gamesPerTeam": st.GamesPerTeam, "hasGroups": st.HasGroups()}
	}
	nMatches, _ := app.CountRecords("matches", dbx.HashExp{"tournament": t.Id, "stage": "group"})
	return map[string]any{
		"groups": out, "unassigned": unassigned, "structure": structure, "groupMatches": nMatches,
	}, nil
}

// saveGroups replaces the tournament's groups with the given assignment
// and rewrites every group-stage match's letter from its home team (away
// team as fallback). Validates letters and membership first.
func saveGroups(app core.App, t *core.Record, in []struct {
	Letter string   `json:"letter"`
	Teams  []string `json:"teams"`
}) error {
	teams, err := app.FindRecordsByFilter("teams", "tournament = {:t}", "", 0, 0, map[string]any{"t": t.Id})
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, tm := range teams {
		known[tm.Id] = true
	}
	letters := map[string]bool{}
	letterOf := map[string]string{}
	for _, g := range in {
		l := strings.ToUpper(strings.TrimSpace(g.Letter))
		if l == "" || len(l) > 2 {
			return apis.NewBadRequestError(fmt.Sprintf("group letter %q must be 1–2 characters", g.Letter), nil)
		}
		if letters[l] {
			return apis.NewBadRequestError(fmt.Sprintf("group %s listed twice", l), nil)
		}
		letters[l] = true
		for _, id := range g.Teams {
			if !known[id] {
				return apis.NewBadRequestError("a team is not in this tournament", nil)
			}
			if prev, dup := letterOf[id]; dup {
				return apis.NewBadRequestError(fmt.Sprintf("a team sits in groups %s and %s", prev, l), nil)
			}
			letterOf[id] = l
		}
	}

	return app.RunInTransaction(func(tx core.App) error {
		existing, err := tx.FindRecordsByFilter("tournament_groups", "tournament = {:t}", "", 0, 0, map[string]any{"t": t.Id})
		if err != nil {
			return err
		}
		byLetter := map[string]*core.Record{}
		for _, g := range existing {
			byLetter[g.GetString("letter")] = g
		}
		col, err := tx.FindCollectionByNameOrId("tournament_groups")
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		largest := 0
		for _, g := range in {
			l := strings.ToUpper(strings.TrimSpace(g.Letter))
			seen[l] = true
			rec := byLetter[l]
			if rec == nil {
				rec = core.NewRecord(col)
				rec.Set("tournament", t.Id)
				rec.Set("letter", l)
			}
			rec.Set("teams", g.Teams)
			if err := tx.Save(rec); err != nil {
				return fmt.Errorf("save group %s: %w", l, err)
			}
			if len(g.Teams) > largest {
				largest = len(g.Teams)
			}
		}
		for l, rec := range byLetter {
			if !seen[l] {
				if err := tx.Delete(rec); err != nil {
					return fmt.Errorf("delete group %s: %w", l, err)
				}
			}
		}
		// Every group-stage match follows its teams.
		matches, err := tx.FindRecordsByFilter("matches", "tournament = {:t} && stage = 'group'", "", 0, 0, map[string]any{"t": t.Id})
		if err != nil {
			return err
		}
		for _, m := range matches {
			l := letterOf[m.GetString("homeTeam")]
			if l == "" {
				l = letterOf[m.GetString("awayTeam")]
			}
			if m.GetString("groupLetter") != l {
				m.Set("groupLetter", l)
				if err := tx.Save(m); err != nil {
					return fmt.Errorf("match %s: %w", m.Id, err)
				}
			}
		}
		// The structure's group size follows the largest group.
		if st, err := StructureOf(t); err == nil && largest > 0 && st.GroupSize != largest {
			st.GroupSize = largest
			b, _ := json.Marshal(st)
			t.Set("structure", string(b))
			if err := tx.Save(t); err != nil {
				return fmt.Errorf("structure: %w", err)
			}
		}
		return nil
	})
}

// registerGroups wires the editor's endpoints under the admin tournaments
// group (auth + admin already bound).
func registerGroups(app core.App, g *router.RouterGroup[*core.RequestEvent]) {
	// GET /api/admin/tournaments/{id}/groups
	g.GET("/{id}/groups", func(e *core.RequestEvent) error {
		t, err := app.FindRecordById(collection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		v, err := groupsView(app, t)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, v)
	})
	// PUT /api/admin/tournaments/{id}/groups { groups: [{letter, teams:[ids]}] }
	g.PUT("/{id}/groups", func(e *core.RequestEvent) error {
		t, err := app.FindRecordById(collection, e.Request.PathValue("id"))
		if err != nil {
			return apis.NewNotFoundError("no such tournament", nil)
		}
		var body struct {
			Groups []struct {
				Letter string   `json:"letter"`
				Teams  []string `json:"teams"`
			} `json:"groups"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		sort.SliceStable(body.Groups, func(i, j int) bool { return body.Groups[i].Letter < body.Groups[j].Letter })
		if err := saveGroups(app, t, body.Groups); err != nil {
			return err
		}
		t, _ = app.FindRecordById(collection, t.Id)
		v, err := groupsView(app, t)
		if err != nil {
			return err
		}
		return e.JSON(http.StatusOK, v)
	})
}
