package survey

import (
	"slices"
	"testing"
)

// A minimal valid submission to mutate per case.
func base() answers {
	return answers{
		Enjoy:       4,
		PlayedOther: new(false),
		Liked:       []string{"tips"},
		PlayAgain:   "probably",
		PlaySeason:  "maybe",
		FairPrice:   "fee",
	}
}

func TestValidateDropsTextlessOther(t *testing.T) {
	a := base()
	a.Liked = []string{"other", "design"}
	a.Competitions = []string{"laLiga", "other"}
	a.CompetitionsOther = "  " // whitespace only = no text
	if err := a.validate(); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(a.Liked, "other") || slices.Contains(a.Competitions, "other") {
		t.Fatalf("textless other picks survived: liked=%v competitions=%v", a.Liked, a.Competitions)
	}
	if !slices.Contains(a.Liked, "design") || !slices.Contains(a.Competitions, "laLiga") {
		t.Fatalf("real picks were dropped: liked=%v competitions=%v", a.Liked, a.Competitions)
	}
}

func TestValidateClearsOrphanedOtherText(t *testing.T) {
	a := base()
	a.LikedOther = "the bots"
	a.Competitions = []string{"bundesliga"}
	a.CompetitionsOther = "my sunday league"
	if err := a.validate(); err != nil {
		t.Fatal(err)
	}
	if a.LikedOther != "" || a.CompetitionsOther != "" {
		t.Fatalf("text without the other pick survived: %q / %q", a.LikedOther, a.CompetitionsOther)
	}
}

func TestValidateKeepsOtherWithText(t *testing.T) {
	a := base()
	a.Liked = []string{"other"}
	a.LikedOther = "the confetti"
	a.Competitions = []string{"other"}
	a.CompetitionsOther = "Eredivisie"
	if err := a.validate(); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(a.Liked, "other") || a.LikedOther != "the confetti" {
		t.Fatalf("liked other+text was dropped: %v %q", a.Liked, a.LikedOther)
	}
	if !slices.Contains(a.Competitions, "other") || a.CompetitionsOther != "Eredivisie" {
		t.Fatalf("competitions other+text was dropped: %v %q", a.Competitions, a.CompetitionsOther)
	}
}

func TestValidateNoSeasonClearsCompetitions(t *testing.T) {
	a := base()
	a.PlaySeason = "no"
	a.Competitions = []string{"laLiga", "other"}
	a.CompetitionsOther = "Allsvenskan"
	if err := a.validate(); err != nil {
		t.Fatal(err)
	}
	if len(a.Competitions) != 0 || a.CompetitionsOther != "" {
		t.Fatalf("playSeason=no left competitions: %v %q", a.Competitions, a.CompetitionsOther)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(*answers){
		"enjoy out of range":  func(a *answers) { a.Enjoy = 6 },
		"missing playedOther": func(a *answers) { a.PlayedOther = nil },
		"comparison required": func(a *answers) { a.PlayedOther = new(true); a.Comparison = "" },
		"bad liked key":       func(a *answers) { a.Liked = []string{"hax"} },
		"bad competition key": func(a *answers) { a.Competitions = []string{"hax"} },
		"bad fairPrice":       func(a *answers) { a.FairPrice = "gold" },
	}
	for name, mut := range cases {
		a := base()
		mut(&a)
		if err := a.validate(); err == nil {
			t.Errorf("%s: expected error, got none", name)
		}
	}
}
