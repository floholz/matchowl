// Package survey serves the end-of-tournament feedback survey. The questions
// live in the frontend (single survey, single run); this side only validates
// the submitted answer shape. One record per user (unique index on
// survey_responses.user), but re-submitting updates it in place — players may
// refine their answers after the final. Raw responses are read from the PB
// dashboard for analysis; the API only returns the caller's own answers.
package survey

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const collection = "survey_responses"

// Register wires the survey endpoints (any signed-in user).
func Register(app core.App, se *core.ServeEvent) {
	// GET /api/survey — the caller's own submission, for pre-filling the form.
	se.Router.GET("/api/survey", func(e *core.RequestEvent) error {
		rec, err := findByUser(app, e.Auth.Id)
		if err != nil {
			return err
		}
		if rec == nil {
			return e.JSON(http.StatusOK, map[string]any{"submitted": false})
		}
		return e.JSON(http.StatusOK, map[string]any{
			"submitted": true,
			"answers":   rec.Get("answers"),
		})
	}).Bind(apis.RequireAuth())

	// POST /api/survey — submit, or update an earlier submission in place (the
	// unique index keeps it at one record per user either way).
	se.Router.POST("/api/survey", func(e *core.RequestEvent) error {
		var body struct {
			Answers answers `json:"answers"`
		}
		if err := e.BindBody(&body); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		a := body.Answers
		if err := a.validate(); err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		rec, err := findByUser(app, e.Auth.Id)
		if err != nil {
			return err
		}
		if rec == nil {
			col, err := app.FindCollectionByNameOrId(collection)
			if err != nil {
				return err
			}
			rec = core.NewRecord(col)
			rec.Set("user", e.Auth.Id)
		}
		rec.Set("answers", a)
		if err := app.Save(rec); err != nil {
			// Two first-time submits racing land on the unique index.
			return apis.NewBadRequestError("please retry", nil)
		}
		return e.JSON(http.StatusOK, map[string]any{"submitted": true})
	}).Bind(apis.RequireAuth())
}

func findByUser(app core.App, userID string) (*core.Record, error) {
	recs, err := app.FindRecordsByFilter(collection, "user = {:u}", "", 1, 0,
		map[string]any{"u": userID})
	if err != nil || len(recs) == 0 {
		return nil, err
	}
	return recs[0], nil
}

// answers is the validated survey payload (v1). Enum values and free-text
// bounds mirror the form in frontend/src/routes/survey/+page.svelte.
type answers struct {
	V                 int      `json:"v"`
	Enjoy             int      `json:"enjoy"`                       // 1–5 overall rating
	PlayedOther       *bool    `json:"playedOther"`                 // played other prediction games before
	Comparison        string   `json:"comparison,omitempty"`        // better|same|worse (iff playedOther)
	Liked             []string `json:"liked"`                       // feature keys, see likedKeys
	LikedOther        string   `json:"likedOther,omitempty"`        // free text for the "other" pick
	Annoyed           string   `json:"annoyed,omitempty"`           // free text
	PlayAgain         string   `json:"playAgain"`                   // definitely|probably|no
	PlaySeason        string   `json:"playSeason"`                  // definitely|maybe|no
	Competitions      []string `json:"competitions,omitempty"`      // iff playSeason != no
	CompetitionsOther string   `json:"competitionsOther,omitempty"` // free text for the "other" pick
	FairPrice         string   `json:"fairPrice"`                   // donations|fee|nopay
	Comments          string   `json:"comments,omitempty"`          // free text
}

var (
	likedKeys       = enum("tips", "forecast", "leagues", "chat", "notifications", "design", "other")
	competitionKeys = enum("premierLeague", "bundesliga", "laLiga", "serieA", "ligue1", "championsLeague", "other")
	comparisons     = enum("better", "same", "worse")
	playAgains      = enum("definitely", "probably", "no")
	playSeasons     = enum("definitely", "maybe", "no")
	fairPrices      = enum("donations", "fee", "nopay")
)

func (a *answers) validate() error {
	a.V = 1
	if a.Enjoy < 1 || a.Enjoy > 5 {
		return fmt.Errorf("enjoy must be 1–5")
	}
	if a.PlayedOther == nil {
		return fmt.Errorf("playedOther is required")
	}
	if *a.PlayedOther {
		if !comparisons[a.Comparison] {
			return fmt.Errorf("comparison is required")
		}
	} else {
		a.Comparison = ""
	}
	if len(a.Liked) > len(likedKeys) {
		return fmt.Errorf("too many liked picks")
	}
	for _, k := range a.Liked {
		if !likedKeys[k] {
			return fmt.Errorf("unknown liked key %q", k)
		}
	}
	if !playAgains[a.PlayAgain] {
		return fmt.Errorf("playAgain is required")
	}
	if !playSeasons[a.PlaySeason] {
		return fmt.Errorf("playSeason is required")
	}
	if a.PlaySeason == "no" {
		a.Competitions = nil
		a.CompetitionsOther = ""
	}
	if len(a.Competitions) > len(competitionKeys) {
		return fmt.Errorf("too many competition picks")
	}
	for _, k := range a.Competitions {
		if !competitionKeys[k] {
			return fmt.Errorf("unknown competition key %q", k)
		}
	}
	if !fairPrices[a.FairPrice] {
		return fmt.Errorf("fairPrice is required")
	}
	a.LikedOther = clip(a.LikedOther, 300)
	a.CompetitionsOther = clip(a.CompetitionsOther, 300)
	a.Annoyed = clip(a.Annoyed, 2000)
	a.Comments = clip(a.Comments, 2000)
	// An "other" pick without text (or text without the pick) says nothing —
	// drop the orphaned half so exports stay clean.
	if !slices.Contains(a.Liked, "other") {
		a.LikedOther = ""
	}
	if a.LikedOther == "" {
		a.Liked = drop(a.Liked, "other")
	}
	if !slices.Contains(a.Competitions, "other") {
		a.CompetitionsOther = ""
	}
	if a.CompetitionsOther == "" {
		a.Competitions = drop(a.Competitions, "other")
	}
	return nil
}

func drop(list []string, key string) []string {
	return slices.DeleteFunc(list, func(k string) bool { return k == key })
}

func clip(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		s = s[:max]
	}
	return s
}

func enum(keys ...string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, k := range keys {
		m[k] = true
	}
	return m
}
