package migrations

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Competition hub: every tournament is a season of a competition.
//
//   - competitions.description (text ≤ 500): admin-written blurb shown on
//     the hub; empty → the client derives one from the season's structure.
//   - every tournament without a competition gets one created from its own
//     name (key from the slug, teamKind guessed from its teams' flags), so
//     the relation can become required.
//   - tournaments.competition → required.
func init() {
	m.Register(func(app core.App) error {
		comps, err := app.FindCollectionByNameOrId(nCompetitions)
		if err != nil {
			return err
		}
		if comps.Fields.GetByName("description") == nil {
			comps.Fields.Add(&core.TextField{Name: "description", Max: 500})
			if err := app.Save(comps); err != nil {
				return err
			}
		}

		// Backfill orphans.
		orphans, err := app.FindRecordsByFilter(nTournaments, "competition = ''", "startsAt", 0, 0)
		if err != nil {
			return err
		}
		for _, t := range orphans {
			c := core.NewRecord(comps)
			c.Set("key", uniqueCompKey(app, compKeyFrom(t.GetString("slug"), t.GetString("name"))))
			// The imported shortName is the bare league name ("Serie A");
			// else strip the season off the tournament name.
			name := t.GetString("shortName")
			if name == "" {
				name = compNameFrom(t.GetString("name"))
			}
			c.Set("name", name)
			c.Set("teamKind", guessTeamKind(app, t.Id))
			// Carry the provider league id over so logo fetches and later
			// imports of the same league find this competition.
			var sync struct {
				APIFootballLeague int `json:"apiFootballLeague"`
			}
			if err := t.UnmarshalJSONField("sync", &sync); err == nil && sync.APIFootballLeague != 0 {
				if _, err := app.FindFirstRecordByFilter(nCompetitions, "apiFootballLeague = {:l}",
					map[string]any{"l": sync.APIFootballLeague}); err != nil {
					c.Set("apiFootballLeague", sync.APIFootballLeague)
				}
			}
			if err := app.Save(c); err != nil {
				return err
			}
			t.Set("competition", c.Id)
			if err := app.Save(t); err != nil {
				return err
			}
		}

		tcol, err := app.FindCollectionByNameOrId(nTournaments)
		if err != nil {
			return err
		}
		if f, ok := tcol.Fields.GetByName("competition").(*core.RelationField); ok && !f.Required {
			f.Required = true
			return app.Save(tcol)
		}
		return nil
	}, func(app core.App) error {
		tcol, err := app.FindCollectionByNameOrId(nTournaments)
		if err != nil {
			return err
		}
		if f, ok := tcol.Fields.GetByName("competition").(*core.RelationField); ok && f.Required {
			f.Required = false
			if err := app.Save(tcol); err != nil {
				return err
			}
		}
		comps, err := app.FindCollectionByNameOrId(nCompetitions)
		if err != nil {
			return err
		}
		if comps.Fields.GetByName("description") != nil {
			comps.Fields.RemoveByName("description")
			return app.Save(comps)
		}
		return nil
	})
}

var compKeyJunk = regexp.MustCompile(`[^a-z0-9]+`)

// compKeyFrom turns "bundesliga-2025-26" / "wc2026" into a competition key:
// the slug with trailing season digits stripped, else the slugified name.
func compKeyFrom(slug, name string) string {
	k := regexp.MustCompile(`[-_]?(19|20)\d{2}([-/](19|20)?\d{2})?$`).ReplaceAllString(slug, "")
	k = regexp.MustCompile(`(19|20)\d{2}$`).ReplaceAllString(k, "")
	k = strings.Trim(k, "-")
	if len(k) < 2 {
		k = strings.Trim(compKeyJunk.ReplaceAllString(strings.ToLower(name), "-"), "-")
	}
	if len(k) < 2 {
		k = slug
	}
	if len(k) > 32 {
		k = strings.Trim(k[:32], "-")
	}
	return k
}

var seasonSuffix = regexp.MustCompile(`\s+(19|20)\d{2}([-/](19|20)?\d{2})?$`)

// compNameFrom strips a trailing season from a tournament name:
// "Serie A 2026/27" → "Serie A", "FIFA World Cup 2026" → "FIFA World Cup".
func compNameFrom(name string) string {
	n := strings.TrimSpace(seasonSuffix.ReplaceAllString(name, ""))
	if n == "" {
		return name
	}
	return n
}

func uniqueCompKey(app core.App, base string) string {
	key := base
	for i := 2; ; i++ {
		if _, err := app.FindFirstRecordByFilter(nCompetitions, "key = {:k}", map[string]any{"k": key}); err != nil {
			return key
		}
		suffix := fmt.Sprintf("-%d", i)
		key = base
		if len(key)+len(suffix) > 32 {
			key = key[:32-len(suffix)]
		}
		key += suffix
	}
}

// guessTeamKind: a tournament whose teams mostly carry a country flag is a
// national-team event; clubs don't have iso2 set.
func guessTeamKind(app core.App, tournamentID string) string {
	teams, err := app.FindRecordsByFilter(nTeams, "tournament = {:t}", "", 0, 0, map[string]any{"t": tournamentID})
	if err != nil || len(teams) == 0 {
		return "national"
	}
	n := 0
	for _, t := range teams {
		if t.GetString("iso2") != "" {
			n++
		}
	}
	if n*2 > len(teams) {
		return "national"
	}
	return "club"
}
