package sync

import (
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/football"
)

// eventKeys derives a stable key for each provider event. Events carry no
// id, so the key is the event's content (type, detail, minute, team,
// player) plus how often that same content came before it — two identical
// yellow cards stay two rows, and a VAR-cancelled goal disappears as its
// key does.
func eventKeys(evs []football.Event) []string {
	seen := map[string]int{}
	out := make([]string, len(evs))
	for i, e := range evs {
		base := fmt.Sprintf("%s|%s|%d+%d|%d|%d", strings.ToLower(e.Type), strings.ToLower(e.Detail), e.Elapsed, e.Extra, e.TeamID, e.PlayerID)
		out[i] = fmt.Sprintf("%s#%d", base, seen[base])
		seen[base]++
	}
	return out
}

// applyEvents mirrors a fixture's events into match_events for one match:
// new keys are created, known ones updated when their text or order moved,
// and keys the provider no longer lists are deleted. Returns how many rows
// changed.
func applyEvents(app core.App, rec *core.Record, f football.Fixture) int {
	if !f.HasEvents {
		return 0
	}
	col, err := app.FindCollectionByNameOrId("match_events")
	if err != nil {
		return 0
	}
	existing, _ := app.FindRecordsByFilter("match_events", "match = {:m}", "", 0, 0, map[string]any{"m": rec.Id})
	byKey := map[string]*core.Record{}
	for _, r := range existing {
		byKey[r.GetString("key")] = r
	}
	side := sideResolver(app, rec, f)
	keys := eventKeys(f.Events)
	keep := map[string]bool{}
	changed := 0
	for i, e := range f.Events {
		k := keys[i]
		keep[k] = true
		row := byKey[k]
		if row == nil {
			row = core.NewRecord(col)
			row.Set("match", rec.Id)
			row.Set("key", k)
		}
		s, team := side(e)
		vals := map[string]any{
			"seq": i, "minute": e.Elapsed, "extra": e.Extra, "type": e.Type, "detail": e.Detail,
			"side": s, "team": team, "player": e.Player, "assist": e.Assist, "comments": e.Comments,
		}
		dirty := row.IsNew()
		for f, v := range vals {
			if fmt.Sprint(row.Get(f)) != fmt.Sprint(v) {
				row.Set(f, v)
				dirty = true
			}
		}
		if dirty && app.Save(row) == nil {
			changed++
		}
	}
	for k, r := range byKey {
		if !keep[k] && app.Delete(r) == nil {
			changed++
		}
	}
	return changed
}

// sideResolver tells which side an event belongs to, by the provider team id
// on the match's team rows, else by name.
func sideResolver(app core.App, rec *core.Record, f football.Fixture) func(football.Event) (string, string) {
	type side struct {
		name, team string
		pid        int
		label      string
	}
	sides := []side{{name: "home", team: rec.GetString("homeTeam"), label: f.HomeName}, {name: "away", team: rec.GetString("awayTeam"), label: f.AwayName}}
	for i := range sides {
		if t, err := app.FindRecordById("teams", sides[i].team); err == nil {
			sides[i].pid = t.GetInt("providerId")
		}
	}
	if f.HomeID != 0 {
		sides[0].pid = f.HomeID
	}
	if f.AwayID != 0 {
		sides[1].pid = f.AwayID
	}
	return func(e football.Event) (string, string) {
		for _, s := range sides {
			if (s.pid != 0 && s.pid == e.TeamID) || (e.TeamID == 0 && football.NormalizeName(e.Team) == football.NormalizeName(s.label)) {
				return s.name, s.team
			}
		}
		return "", ""
	}
}
