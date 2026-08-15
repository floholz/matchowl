// Package football is a thin client for the API-Football (api-sports.io) free
// tier. One /fixtures call returns a whole season's fixtures for a league, so
// a periodic sync costs a single request and stays well within the 100/day
// free limit. League and season come from the tournament's sync config.
package football

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const baseURL = "https://v3.football.api-sports.io"

// DefaultLeague is API-Football's FIFA World Cup league id, used when a
// tournament's sync config omits one.
const DefaultLeague = 1

type Client struct {
	key    string
	league int
	season int
	http   *http.Client
}

func New(key string, league, season int) *Client {
	if league == 0 {
		league = DefaultLeague
	}
	return &Client{key: key, league: league, season: season, http: &http.Client{Timeout: 20 * time.Second}}
}

// Fixture is the subset of the API-Football fixture payload we use.
type Fixture struct {
	ID        int       // provider fixture id
	Date      time.Time // kickoff (UTC)
	Round     string    // e.g. "Group A - 1", "Round of 32"
	Status    string    // NS, 1H, HT, 2H, ET, BT, P, FT, AET, PEN, PST, CANC, ...
	HomeID    int       // provider team ids (0 when the slot is still TBD)
	AwayID    int
	HomeName  string
	AwayName  string
	HomeLogo  string
	AwayLogo  string
	HomeGoals *int // full 90' (nil if not played)
	AwayGoals *int
	FTHome    *int // regulation
	FTAway    *int
	ETHome    *int // after extra time (cumulative)
	ETAway    *int
	PenHome   *int // shootout
	PenAway   *int
}

// Finished reports whether the provider considers the match complete.
func (f Fixture) Finished() bool {
	switch f.Status {
	case "FT", "AET", "PEN", "WO":
		return true
	}
	return false
}

// Live reports whether the match is currently in progress.
func (f Fixture) Live() bool {
	switch f.Status {
	case "1H", "2H", "HT", "ET", "BT", "P", "LIVE", "INT":
		return true
	}
	return false
}

type apiResponse struct {
	Errors   json.RawMessage `json:"errors"`
	Results  int             `json:"results"`
	Response []struct {
		Fixture struct {
			ID     int       `json:"id"`
			Date   time.Time `json:"date"`
			Status struct {
				Short string `json:"short"`
			} `json:"status"`
		} `json:"fixture"`
		League struct {
			Round string `json:"round"`
		} `json:"league"`
		Teams struct {
			Home fixtureTeam `json:"home"`
			Away fixtureTeam `json:"away"`
		} `json:"teams"`
		Goals struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"goals"`
		Score struct {
			Fulltime  scorePair `json:"fulltime"`
			Extratime scorePair `json:"extratime"`
			Penalty   scorePair `json:"penalty"`
		} `json:"score"`
	} `json:"response"`
}

type scorePair struct {
	Home *int `json:"home"`
	Away *int `json:"away"`
}

type fixtureTeam struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// Fixtures returns every fixture of the configured league+season in a single
// request.
func (c *Client) Fixtures(ctx context.Context) ([]Fixture, error) {
	return c.FixturesForSeason(ctx, c.season)
}

// get performs an authenticated GET against the API and decodes the envelope
// into out (which must be an *apiEnvelope-compatible struct with Errors and
// Response fields). API-level errors (bad key, plan limits) arrive as HTTP 200
// with a non-empty `errors` object, so they are surfaced here.
func (c *Client) get(ctx context.Context, path string, out interface{ errors() json.RawMessage }) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-apisports-key", c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api-football: status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return err
	}
	if s := strings.TrimSpace(string(out.errors())); s != "" && s != "[]" && s != "{}" {
		return fmt.Errorf("api-football errors: %s", s)
	}
	return nil
}

func (a *apiResponse) errors() json.RawMessage { return a.Errors }

// FixturesForSeason fetches the configured league's fixtures for any season
// (used by the dev API diagnostic to replay a finished tournament, e.g. 2022).
func (c *Client) FixturesForSeason(ctx context.Context, yr int) ([]Fixture, error) {
	var ar apiResponse
	if err := c.get(ctx, fmt.Sprintf("/fixtures?league=%d&season=%d", c.league, yr), &ar); err != nil {
		return nil, err
	}

	out := make([]Fixture, 0, len(ar.Response))
	for _, r := range ar.Response {
		out = append(out, Fixture{
			ID:        r.Fixture.ID,
			Date:      r.Fixture.Date.UTC(),
			Round:     r.League.Round,
			Status:    r.Fixture.Status.Short,
			HomeID:    r.Teams.Home.ID,
			AwayID:    r.Teams.Away.ID,
			HomeName:  r.Teams.Home.Name,
			AwayName:  r.Teams.Away.Name,
			HomeLogo:  r.Teams.Home.Logo,
			AwayLogo:  r.Teams.Away.Logo,
			HomeGoals: r.Goals.Home,
			AwayGoals: r.Goals.Away,
			FTHome:    r.Score.Fulltime.Home,
			FTAway:    r.Score.Fulltime.Away,
			ETHome:    r.Score.Extratime.Home,
			ETAway:    r.Score.Extratime.Away,
			PenHome:   r.Score.Penalty.Home,
			PenAway:   r.Score.Penalty.Away,
		})
	}
	return out, nil
}

// Status reports the account/plan and request quota (api-football /status).
func (c *Client) Status(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/status", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api-football: status %d", resp.StatusCode)
	}
	var out struct {
		Response map[string]any `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Response, nil
}

// NormalizeName lowercases and strips non-alphanumerics so provider team names
// can be matched against the openfootball-seeded names despite spelling
// differences ("Korea Republic" vs "South Korea" still need an alias map).
func NormalizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// League is one entry of the API-Football league catalog.
type League struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Type    string   `json:"type"` // "League" | "Cup"
	Logo    string   `json:"logo"`
	Country string   `json:"country"`
	Flag    string   `json:"flag"`
	Seasons []Season `json:"seasons"`
}

// Season is one playable season of a League.
type Season struct {
	Year    int    `json:"year"`
	Start   string `json:"start"` // YYYY-MM-DD
	End     string `json:"end"`
	Current bool   `json:"current"`
}

type leaguesResponse struct {
	Errors   json.RawMessage `json:"errors"`
	Response []struct {
		League struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
			Logo string `json:"logo"`
		} `json:"league"`
		Country struct {
			Name string `json:"name"`
			Flag string `json:"flag"`
		} `json:"country"`
		Seasons []struct {
			Year    int    `json:"year"`
			Start   string `json:"start"`
			End     string `json:"end"`
			Current bool   `json:"current"`
		} `json:"seasons"`
	} `json:"response"`
}

func (l *leaguesResponse) errors() json.RawMessage { return l.Errors }

// Leagues searches the league catalog (/leagues?search=) — the provider
// needs at least 3 characters. Pass id > 0 to fetch a single league instead.
func (c *Client) Leagues(ctx context.Context, search string, id int) ([]League, error) {
	q := "/leagues?search=" + url.QueryEscape(search)
	if id > 0 {
		q = fmt.Sprintf("/leagues?id=%d", id)
	}
	var lr leaguesResponse
	if err := c.get(ctx, q, &lr); err != nil {
		return nil, err
	}
	out := make([]League, 0, len(lr.Response))
	for _, r := range lr.Response {
		l := League{ID: r.League.ID, Name: r.League.Name, Type: r.League.Type, Logo: r.League.Logo,
			Country: r.Country.Name, Flag: r.Country.Flag}
		for _, s := range r.Seasons {
			l.Seasons = append(l.Seasons, Season{Year: s.Year, Start: s.Start, End: s.End, Current: s.Current})
		}
		// Newest season first — that's what an admin adding a tournament wants.
		sort.Slice(l.Seasons, func(i, j int) bool { return l.Seasons[i].Year > l.Seasons[j].Year })
		out = append(out, l)
	}
	return out, nil
}

// Team is one entry of /teams for a league+season.
type Team struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Code     string `json:"code"` // 3-letter, may be empty
	Country  string `json:"country"`
	National bool   `json:"national"`
	Logo     string `json:"logo"`
}

type teamsResponse struct {
	Errors   json.RawMessage `json:"errors"`
	Response []struct {
		Team Team `json:"team"`
	} `json:"response"`
}

func (t *teamsResponse) errors() json.RawMessage { return t.Errors }

// Teams lists the teams of the configured league for a season.
func (c *Client) Teams(ctx context.Context, yr int) ([]Team, error) {
	var tr teamsResponse
	if err := c.get(ctx, fmt.Sprintf("/teams?league=%d&season=%d", c.league, yr), &tr); err != nil {
		return nil, err
	}
	out := make([]Team, 0, len(tr.Response))
	for _, r := range tr.Response {
		out = append(out, r.Team)
	}
	return out, nil
}

// StandingGroup is one table of /standings: a league season has one, a
// group-stage cup has one per group ("Group A", ...).
type StandingGroup struct {
	Name    string `json:"name"`
	TeamIDs []int  `json:"teamIds"`
}

type standingsResponse struct {
	Errors   json.RawMessage `json:"errors"`
	Response []struct {
		League struct {
			Standings [][]struct {
				Group string `json:"group"`
				Team  struct {
					ID int `json:"id"`
				} `json:"team"`
			} `json:"standings"`
		} `json:"league"`
	} `json:"response"`
}

func (s *standingsResponse) errors() json.RawMessage { return s.Errors }

// Standings returns the season's tables — the authoritative group membership
// for cups whose round labels don't carry the group letter ("Group Stage - 1").
func (c *Client) Standings(ctx context.Context, yr int) ([]StandingGroup, error) {
	var sr standingsResponse
	if err := c.get(ctx, fmt.Sprintf("/standings?league=%d&season=%d", c.league, yr), &sr); err != nil {
		return nil, err
	}
	var out []StandingGroup
	for _, r := range sr.Response {
		for _, tbl := range r.League.Standings {
			if len(tbl) == 0 {
				continue
			}
			g := StandingGroup{Name: tbl[0].Group}
			for _, row := range tbl {
				g.TeamIDs = append(g.TeamIDs, row.Team.ID)
			}
			out = append(out, g)
		}
	}
	return out, nil
}
