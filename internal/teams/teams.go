// Package teams keeps the team identity records (PLAN.md, "pools at the
// center", phase 2). A `teams` row exists once per season; a
// `team_identities` row is the club or nation behind those rows, so a team
// can be followed, searched and get a page (/teams/{key}).
//
// Every `teams` row is linked to its identity on create, and a sweep at boot
// links the rows that predate identities. Two rows are the same team when
// they share the provider id, else the club key, else the flag code of a
// national team, else the normalized name.
package teams

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const (
	// Collection holds one row per club or nation.
	Collection = "team_identities"
	// Follows holds who follows which identity.
	Follows = "team_follows"
)

// Register links every new `teams` row to its identity. Call it before the
// seed runs, so seeded rows get linked too.
func Register(app core.App) {
	app.OnRecordCreate("teams").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("identity") == "" {
			if id, err := Identify(e.App, e.Record); err != nil {
				log.Printf("[teams] identify %s: %v", e.Record.GetString("name"), err)
			} else {
				e.Record.Set("identity", id.Id)
			}
		}
		return e.Next()
	})
}

// LinkAll links every `teams` row that has no identity yet (the backfill for
// rows created before identities existed). Returns how many were linked.
func LinkAll(app core.App) (int, error) {
	rows, err := app.FindRecordsByFilter("teams", "identity = ''", "created", 0, 0)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, tm := range rows {
		id, err := Identify(app, tm)
		if err != nil {
			log.Printf("[teams] identify %s: %v", tm.GetString("name"), err)
			continue
		}
		tm.Set("identity", id.Id)
		if err := app.SaveNoValidate(tm); err != nil {
			log.Printf("[teams] link %s: %v", tm.GetString("name"), err)
			continue
		}
		n++
	}
	return n, nil
}

// National reports whether a season team row is a national team: it carries
// a flag code and no club key.
func National(tm *core.Record) bool {
	return tm.GetString("clubKey") == "" && tm.GetString("iso2") != ""
}

// Identify finds the identity a season team row belongs to, creating it when
// there is none yet. It doesn't save the team row.
func Identify(app core.App, tm *core.Record) (*core.Record, error) {
	provider, pid := tm.GetString("provider"), tm.GetInt("providerId")
	if provider != "" && pid != 0 {
		if rec, err := app.FindFirstRecordByFilter(Collection,
			"provider = {:p} && providerId = {:id}", map[string]any{"p": provider, "id": pid}); err == nil {
			return rec, nil
		}
	}
	// Same team without a provider id on one side: another season's row with
	// the same club key, flag code or name that is already linked.
	if rec := siblingIdentity(app, tm); rec != nil {
		if provider != "" && pid != 0 && rec.GetInt("providerId") == 0 {
			rec.Set("provider", provider)
			rec.Set("providerId", pid)
			if err := app.Save(rec); err != nil {
				return nil, err
			}
		}
		return rec, nil
	}

	col, err := app.FindCollectionByNameOrId(Collection)
	if err != nil {
		return nil, err
	}
	rec := core.NewRecord(col)
	national := National(tm)
	base := tm.GetString("clubKey")
	if base == "" {
		base = Slug(tm.GetString("name"))
	}
	if base == "" {
		base = "team"
	}
	key, err := freeKey(app, base)
	if err != nil {
		return nil, err
	}
	rec.Set("key", key)
	rec.Set("name", tm.GetString("name"))
	rec.Set("fifaCode", tm.GetString("fifaCode"))
	rec.Set("iso2", tm.GetString("iso2"))
	rec.Set("national", national)
	if national {
		rec.Set("country", tm.GetString("iso2"))
		rec.Set("countryName", tm.GetString("name"))
	}
	rec.Set("provider", provider)
	rec.Set("providerId", pid)
	if err := app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// siblingIdentity returns the identity of another season's row of the same
// team, matched without the provider id.
func siblingIdentity(app core.App, tm *core.Record) *core.Record {
	var filter string
	params := map[string]any{"self": tm.Id}
	switch {
	case tm.GetString("clubKey") != "":
		filter = "clubKey = {:v}"
		params["v"] = tm.GetString("clubKey")
	case National(tm):
		filter = "clubKey = '' && iso2 = {:v}"
		params["v"] = tm.GetString("iso2")
	default:
		filter = "clubKey = '' && iso2 = '' && name = {:v}"
		params["v"] = tm.GetString("name")
	}
	rows, err := app.FindRecordsByFilter("teams", "identity != '' && id != {:self} && "+filter, "", 0, 0, params)
	if err != nil {
		return nil
	}
	pid := tm.GetInt("providerId")
	for _, r := range rows {
		// Two rows with different provider ids are different teams even if
		// the names agree.
		if o := r.GetInt("providerId"); pid != 0 && o != 0 && o != pid {
			continue
		}
		if rec, err := app.FindRecordById(Collection, r.GetString("identity")); err == nil {
			return rec
		}
	}
	return nil
}

// Relink re-checks a season row's identity after it got its provider id:
// when another identity already carries that id, the row moves there, and
// an identity left without rows is merged into it (follows move along).
// Otherwise the row's identity takes the provider id.
func Relink(app core.App, tm *core.Record) error {
	provider, pid := tm.GetString("provider"), tm.GetInt("providerId")
	cur := tm.GetString("identity")
	if provider == "" || pid == 0 || cur == "" {
		return nil
	}
	other, err := app.FindFirstRecordByFilter(Collection,
		"provider = {:p} && providerId = {:id} && id != {:cur}",
		map[string]any{"p": provider, "id": pid, "cur": cur})
	if err != nil {
		rec, err := app.FindRecordById(Collection, cur)
		if err != nil || rec.GetInt("providerId") != 0 {
			return nil
		}
		rec.Set("provider", provider)
		rec.Set("providerId", pid)
		return app.Save(rec)
	}
	tm.Set("identity", other.Id)
	if err := app.Save(tm); err != nil {
		return err
	}
	if n, _ := app.CountRecords("teams", dbx.HashExp{"identity": cur}); n > 0 {
		return nil
	}
	follows, _ := app.FindRecordsByFilter(Follows, "team = {:t}", "", 0, 0, map[string]any{"t": cur})
	for _, f := range follows {
		if _, err := app.FindFirstRecordByFilter(Follows, "user = {:u} && team = {:t}",
			map[string]any{"u": f.GetString("user"), "t": other.Id}); err == nil {
			continue // already follows the merged team; the cascade drops this one
		}
		f.Set("team", other.Id)
		if err := app.Save(f); err != nil {
			return err
		}
	}
	if old, err := app.FindRecordById(Collection, cur); err == nil {
		return app.Delete(old)
	}
	return nil
}

func freeKey(app core.App, base string) (string, error) {
	if len(base) > 60 {
		base = strings.Trim(base[:60], "-")
	}
	key := base
	if key == "mine" { // /api/teams/mine
		key = base + "-2"
	}
	for i := 2; i < 100; i++ {
		if _, err := app.FindFirstRecordByFilter(Collection, "key = {:k}", map[string]any{"k": key}); err != nil {
			return key, nil
		}
		key = fmt.Sprintf("%s-%d", base, i)
	}
	return "", fmt.Errorf("no free key for %q", base)
}

// SetOrigin records a team's country (as the provider names it) and whether
// it's a national team, keeping values an admin set already. iso is the
// country's ISO code ("" when the name isn't known). Returns whether
// anything changed.
func SetOrigin(app core.App, identityID, countryName, iso string, national bool) (bool, error) {
	rec, err := app.FindRecordById(Collection, identityID)
	if err != nil {
		return false, err
	}
	changed := false
	if national && !rec.GetBool("national") {
		rec.Set("national", true)
		changed = true
	}
	if countryName != "" && rec.GetString("countryName") == "" {
		rec.Set("countryName", countryName)
		changed = true
	}
	if iso != "" && rec.GetString("country") == "" {
		rec.Set("country", iso)
		changed = true
	}
	if !changed {
		return false, nil
	}
	return true, app.Save(rec)
}

// Crest returns the path of the team's newest crest under /api/files
// ("teams/<row>/<file>"), or "".
func Crest(app core.App, identityID string) string {
	rows, err := app.FindRecordsByFilter("teams",
		"identity = {:i} && logo != ''", "-created", 1, 0, map[string]any{"i": identityID})
	if err != nil || len(rows) == 0 {
		return ""
	}
	return "teams/" + rows[0].Id + "/" + rows[0].GetString("logo")
}

// SeasonRowIDs returns the ids of every season row of the given identities.
func SeasonRowIDs(app core.App, identityIDs []string) []string {
	if len(identityIDs) == 0 {
		return nil
	}
	var out []string
	for _, id := range identityIDs {
		rows, err := app.FindRecordsByFilter("teams", "identity = {:i}", "", 0, 0, map[string]any{"i": id})
		if err != nil {
			continue
		}
		for _, r := range rows {
			out = append(out, r.Id)
		}
	}
	return out
}

var nonSlugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Slug turns a name into a URL key ("FK Austria Wien" → "fk-austria-wien").
func Slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss", "é", "e", "è", "e", "á", "a", "í", "i", "ó", "o", "ú", "u", "ñ", "n", "ç", "c").Replace(s)
	s = nonSlugRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}
