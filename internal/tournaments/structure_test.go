package tournaments

import (
	"encoding/json"
	"testing"
)

// euro2028 is a 6-group Euro shape: 24 teams, top 2 advance plus the 4 best
// thirds, no third-place play-off, no official allocation table.
const euro2028 = `{
  "stages": [
    {"code": "group", "name": "Group stage", "kind": "group"},
    {"code": "R16",   "name": "Round of 16", "kind": "knockout"},
    {"code": "QF",    "name": "Quarter-finals", "kind": "knockout"},
    {"code": "SF",    "name": "Semi-finals", "kind": "knockout"},
    {"code": "FINAL", "name": "Final", "kind": "knockout"}
  ],
  "groupSize": 4, "gamesPerTeam": 3, "directQualifiers": 2,
  "extraQualifiers": {"fromPosition": 3, "count": 4}
}`

// koCup is a straight knockout cup: no groups, no qualifiers, a consolation
// third-place match before the final.
const koCup = `{
  "stages": [
    {"code": "QF",    "name": "Quarter-finals", "kind": "knockout"},
    {"code": "SF",    "name": "Semi-finals", "kind": "knockout"},
    {"code": "3RD",   "name": "Third place", "kind": "knockout", "consolation": true},
    {"code": "FINAL", "name": "Final", "kind": "knockout"}
  ]
}`

func parse(t *testing.T, raw string) *Structure {
	t.Helper()
	s := &Structure{}
	if err := json.Unmarshal([]byte(raw), s); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	s.Normalize()
	if err := s.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	return s
}

func TestWC2026StructureIsValid(t *testing.T) {
	s := parse(t, WC2026Structure)
	if got := s.ChampionStage().Code; got != "FINAL" {
		t.Errorf("champion stage = %q, want FINAL", got)
	}
	if !s.IsKnockout("3RD") || s.IsKnockout("group") {
		t.Error("stage kinds wrong")
	}
	if s.ExtraQualifiers == nil || s.ExtraQualifiers.Count != 8 || s.ExtraQualifiers.TableKey != "wc2026" {
		t.Errorf("extraQualifiers = %+v", s.ExtraQualifiers)
	}
}

func TestEuroShape(t *testing.T) {
	s := parse(t, euro2028)
	if s.PointsWin != 3 || s.PointsDraw != 1 {
		t.Errorf("Normalize points = %d/%d, want 3/1", s.PointsWin, s.PointsDraw)
	}
	if got := s.ChampionStage().Code; got != "FINAL" {
		t.Errorf("champion stage = %q, want FINAL", got)
	}
	if n := len(s.KnockoutStages()); n != 4 {
		t.Errorf("knockout stages = %d, want 4", n)
	}
	if s.GroupStage() == nil || s.GroupStage().Code != "group" {
		t.Error("group stage not found")
	}
	if s.ExtraQualifiers.Count != 4 || s.ExtraQualifiers.TableKey != "" {
		t.Errorf("extraQualifiers = %+v", s.ExtraQualifiers)
	}
}

func TestKnockoutCupShape(t *testing.T) {
	s := parse(t, koCup)
	if s.HasGroups() {
		t.Error("KO cup should have no group stage")
	}
	// Consolation must not crown the champion even though it precedes FINAL.
	if got := s.ChampionStage().Code; got != "FINAL" {
		t.Errorf("champion stage = %q, want FINAL", got)
	}
	if s.ExtraQualifiers != nil {
		t.Error("KO cup should have no extra qualifiers")
	}
}

func TestValidateRejections(t *testing.T) {
	bad := []string{
		`{"stages": []}`, // no stages
		`{"stages": [{"code":"a","name":"A","kind":"group"},{"code":"a","name":"A2","kind":"knockout"}], "groupSize":4,"gamesPerTeam":3}`, // dup code
		`{"stages": [{"code":"g","name":"G","kind":"group"}], "groupSize": 1, "gamesPerTeam": 3}`,                                         // groupSize < 2
		`{"stages": [{"code":"F","name":"Final","kind":"knockout"}], "extraQualifiers": {"fromPosition":3,"count":8}}`,                    // extras without groups
		`{"stages": [{"code":"g","name":"G","kind":"weird"}]}`,                                                                           // bad kind
		`{"stages": [{"code":"g","name":"G","kind":"group","consolation":true}], "groupSize":4,"gamesPerTeam":3}`,                         // consolation group
	}
	for i, raw := range bad {
		s := &Structure{}
		if err := json.Unmarshal([]byte(raw), s); err != nil {
			t.Fatalf("case %d unmarshal: %v", i, err)
		}
		if err := s.Validate(); err == nil {
			t.Errorf("case %d: expected validation error", i)
		}
	}
}
