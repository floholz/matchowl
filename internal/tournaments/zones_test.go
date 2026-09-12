package tournaments

import "testing"

func TestNormalizeDefaultZonesForLeaguePhase(t *testing.T) {
	s := &Structure{
		Stages: []Stage{
			{Code: "group", Name: "Pool", Kind: KindGroup},
			{Code: "R32", Name: "Round of 32", Kind: KindKnockout},
			{Code: "R16", Name: "Round of 16", Kind: KindKnockout},
		},
		GroupSize:    36,
		GamesPerTeam: 8,
	}
	s.Normalize()
	if len(s.Zones) != 2 || s.Zones[0].To != 8 || s.Zones[1].From != 9 || s.Zones[1].To != 24 {
		t.Fatalf("zones = %+v", s.Zones)
	}
	if s.Zones[1].Name != "Round of 32" {
		t.Fatalf("play-off zone named %q", s.Zones[1].Name)
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func TestNormalizeLeavesOtherShapesWithoutZones(t *testing.T) {
	league := &Structure{Stages: []Stage{{Code: "group", Name: "Pool", Kind: KindGroup}}, GroupSize: 20}
	league.Normalize()
	if len(league.Zones) != 0 {
		t.Fatalf("plain pool got zones %+v", league.Zones)
	}
	phaseOnly := &Structure{Stages: []Stage{{Code: "group", Name: "Pool", Kind: KindGroup}}, GroupSize: 36}
	phaseOnly.Normalize()
	if len(phaseOnly.Zones) != 2 || phaseOnly.Zones[1].Name != "Knockout play-offs" {
		t.Fatalf("pool phase before the draw should get the fixed zones, got %+v", phaseOnly.Zones)
	}
	custom := &Structure{
		Stages:    []Stage{{Code: "group", Name: "Pool", Kind: KindGroup}, {Code: "R16", Name: "Round of 16", Kind: KindKnockout}},
		GroupSize: 36,
		Zones:     []Zone{{Key: "x", Name: "Custom", From: 1, To: 4}},
	}
	custom.Normalize()
	if len(custom.Zones) != 1 || custom.Zones[0].Key != "x" {
		t.Fatalf("admin zones overwritten: %+v", custom.Zones)
	}
}
