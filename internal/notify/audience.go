package notify

import (
	"sort"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/users"
)

// Audience is a stored recipient filter for a mailing. Every criterion
// narrows; an unset one does not apply. The base set is every human
// account with a verified email (unverified accounts get no mail, ever).
type Audience struct {
	Roles        []string `json:"roles,omitempty"`        // member | admin | owner ("member" = no role)
	Playing      string   `json:"playing,omitempty"`      // tournament id: plays this season
	Pool         string   `json:"pool,omitempty"`         // pool id: member of this pool
	JoinedAfter  string   `json:"joinedAfter,omitempty"`  // YYYY-MM-DD
	JoinedBefore string   `json:"joinedBefore,omitempty"` // YYYY-MM-DD
	Emails       []string `json:"emails,omitempty"`       // explicit list: only these
	Exclude      []string `json:"exclude,omitempty"`      // never these
}

// Recipient is one resolved member of an audience.
type Recipient struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Resolve lists the users an audience reaches, in name order.
func (a Audience) Resolve(app core.App) ([]*core.Record, error) {
	all, err := app.FindRecordsByFilter("users", "id != ''", "name", 0, 0)
	if err != nil {
		return nil, err
	}
	set := func(list []string) map[string]bool {
		if len(list) == 0 {
			return nil
		}
		m := map[string]bool{}
		for _, v := range list {
			if v = strings.ToLower(strings.TrimSpace(v)); v != "" {
				m[v] = true
			}
		}
		return m
	}
	only, never, roles := set(a.Emails), set(a.Exclude), set(a.Roles)
	var playing, inPool map[string]bool
	if a.Playing != "" {
		playing = map[string]bool{}
		recs, _ := app.FindRecordsByFilter("tournament_players", "tournament = {:t}", "", 0, 0, map[string]any{"t": a.Playing})
		for _, r := range recs {
			playing[r.GetString("user")] = true
		}
	}
	if a.Pool != "" {
		inPool = map[string]bool{}
		recs, _ := app.FindRecordsByFilter("pool_members", "pool = {:p}", "", 0, 0, map[string]any{"p": a.Pool})
		for _, r := range recs {
			inPool[r.GetString("user")] = true
		}
	}
	var after, before time.Time
	if a.JoinedAfter != "" {
		after, _ = time.Parse("2006-01-02", a.JoinedAfter)
	}
	if a.JoinedBefore != "" {
		before, _ = time.Parse("2006-01-02", a.JoinedBefore)
	}
	out := make([]*core.Record, 0, len(all))
	for _, u := range all {
		if users.IsBot(u) || u.Email() == "" || !u.Verified() {
			continue
		}
		email := strings.ToLower(u.Email())
		if only != nil && !only[email] {
			continue
		}
		if never != nil && never[email] {
			continue
		}
		if roles != nil {
			role := u.GetString("role")
			if role == "" {
				role = "member"
			}
			if !roles[role] {
				continue
			}
		}
		if playing != nil && !playing[u.Id] {
			continue
		}
		if inPool != nil && !inPool[u.Id] {
			continue
		}
		created := u.GetDateTime("created").Time()
		if !after.IsZero() && created.Before(after) {
			continue
		}
		if !before.IsZero() && !created.Before(before.Add(24*time.Hour)) {
			continue
		}
		out = append(out, u)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return strings.ToLower(out[i].GetString("name")) < strings.ToLower(out[j].GetString("name"))
	})
	return out, nil
}
