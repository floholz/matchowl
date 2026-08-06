// Package tournaments owns the tournament root entity introduced by the
// Matchowl rework. Each tournament record carries its competition structure
// (stages, group shape, qualifier rules) and results-sync config as JSON, so
// the rest of the app reads tournament shape as data instead of hardcoding
// one event's format.
package tournaments

import (
	"fmt"
	"regexp"
)

// Stage kinds.
const (
	KindGroup    = "group"
	KindKnockout = "knockout"
)

// Tournament statuses. draft tournaments are invisible to non-admins;
// archived ones are read-only history.
const (
	StatusDraft    = "draft"
	StatusUpcoming = "upcoming"
	StatusActive   = "active"
	StatusFinished = "finished"
	StatusArchived = "archived"
)

// Statuses in lifecycle order, reused by the migration's select field.
var Statuses = []string{StatusDraft, StatusUpcoming, StatusActive, StatusFinished, StatusArchived}

// Stage is one phase of a tournament, in play order. Consolation marks
// knockout stages whose winner is not on the path to the title (e.g. a
// third-place play-off): they never crown the champion.
type Stage struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Consolation bool   `json:"consolation,omitempty"`
}

// ExtraQualifiers configures group-stage qualification beyond the top
// DirectQualifiers of each group (WC2026: the 8 best third-placed teams).
// TableKey optionally names an official slot-allocation table registered in
// internal/bracket; without one the greedy fallback allocates slots.
type ExtraQualifiers struct {
	FromPosition int    `json:"fromPosition"`
	Count        int    `json:"count"`
	TableKey     string `json:"tableKey,omitempty"`
}

// Zone is a named position range of the final table (league shapes): "1-4
// Champions League", "18-20 relegated". Zones color the standings UI and are
// the targets of forecast teamset calls.
type Zone struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	From int    `json:"from"`
	To   int    `json:"to"`
}

// Structure is the parsed `structure` JSON of a tournament record. Group
// shape fields are only meaningful when a group-kind stage exists.
type Structure struct {
	Stages           []Stage          `json:"stages"`
	GroupSize        int              `json:"groupSize,omitempty"`
	GamesPerTeam     int              `json:"gamesPerTeam,omitempty"`
	DirectQualifiers int              `json:"directQualifiers,omitempty"`
	ExtraQualifiers  *ExtraQualifiers `json:"extraQualifiers,omitempty"`
	Zones            []Zone           `json:"zones,omitempty"`
	PointsWin        int              `json:"pointsWin,omitempty"`
	PointsDraw       int              `json:"pointsDraw,omitempty"`
}

// Zone returns the zone with the given key, or nil.
func (s *Structure) Zone(key string) *Zone {
	for i := range s.Zones {
		if s.Zones[i].Key == key {
			return &s.Zones[i]
		}
	}
	return nil
}

var stageCodeRe = regexp.MustCompile(`^[A-Za-z0-9_-]{1,12}$`)

// Validate checks internal consistency; it does not apply defaults (see
// Normalize).
func (s *Structure) Validate() error {
	if len(s.Stages) == 0 {
		return fmt.Errorf("structure: at least one stage required")
	}
	seen := map[string]bool{}
	groups := 0
	for i, st := range s.Stages {
		if !stageCodeRe.MatchString(st.Code) {
			return fmt.Errorf("structure: stage %d has invalid code %q", i, st.Code)
		}
		if seen[st.Code] {
			return fmt.Errorf("structure: duplicate stage code %q", st.Code)
		}
		seen[st.Code] = true
		if st.Name == "" || len(st.Name) > 64 {
			return fmt.Errorf("structure: stage %q needs a name (max 64 chars)", st.Code)
		}
		switch st.Kind {
		case KindGroup:
			groups++
		case KindKnockout:
		default:
			return fmt.Errorf("structure: stage %q has unknown kind %q", st.Code, st.Kind)
		}
		if st.Consolation && st.Kind != KindKnockout {
			return fmt.Errorf("structure: stage %q: only knockout stages can be consolation", st.Code)
		}
	}
	if groups > 1 {
		return fmt.Errorf("structure: at most one group stage is supported")
	}
	if groups == 1 {
		// 24 covers big league seasons (Serie A = 20); WC-style groups are 4.
		if s.GroupSize < 2 || s.GroupSize > 24 {
			return fmt.Errorf("structure: groupSize must be 2..24")
		}
		if s.GamesPerTeam < 1 {
			return fmt.Errorf("structure: gamesPerTeam must be >= 1")
		}
		if s.DirectQualifiers < 0 || s.DirectQualifiers > s.GroupSize {
			return fmt.Errorf("structure: directQualifiers must be 0..groupSize")
		}
		if eq := s.ExtraQualifiers; eq != nil {
			if eq.FromPosition < 1 || eq.FromPosition > s.GroupSize {
				return fmt.Errorf("structure: extraQualifiers.fromPosition must be 1..groupSize")
			}
			if eq.Count < 1 {
				return fmt.Errorf("structure: extraQualifiers.count must be >= 1")
			}
		}
	} else if s.ExtraQualifiers != nil {
		return fmt.Errorf("structure: extraQualifiers requires a group stage")
	}
	if len(s.Zones) > 0 && groups == 0 {
		return fmt.Errorf("structure: zones require a group stage")
	}
	seenZone := map[string]bool{}
	for _, z := range s.Zones {
		if !stageCodeRe.MatchString(z.Key) {
			return fmt.Errorf("structure: zone key %q invalid", z.Key)
		}
		if seenZone[z.Key] {
			return fmt.Errorf("structure: duplicate zone %q", z.Key)
		}
		seenZone[z.Key] = true
		if z.Name == "" || len(z.Name) > 64 {
			return fmt.Errorf("structure: zone %q needs a name (max 64 chars)", z.Key)
		}
		if z.From < 1 || z.To < z.From || z.To > s.GroupSize {
			return fmt.Errorf("structure: zone %q range %d-%d invalid for groupSize %d", z.Key, z.From, z.To, s.GroupSize)
		}
	}
	if s.PointsWin < 0 || s.PointsDraw < 0 || (s.PointsWin > 0 && s.PointsDraw > s.PointsWin) {
		return fmt.Errorf("structure: invalid pointsWin/pointsDraw")
	}
	return nil
}

// Normalize applies defaults for omitted optional values (3-1-0 points).
func (s *Structure) Normalize() {
	if s.HasGroups() {
		if s.PointsWin == 0 {
			s.PointsWin = 3
		}
		if s.PointsDraw == 0 && s.PointsWin != 1 {
			s.PointsDraw = 1
		}
	}
}

// HasGroups reports whether the tournament has a group stage.
func (s *Structure) HasGroups() bool { return s.GroupStage() != nil }

// GroupStage returns the group-kind stage, or nil.
func (s *Structure) GroupStage() *Stage {
	for i := range s.Stages {
		if s.Stages[i].Kind == KindGroup {
			return &s.Stages[i]
		}
	}
	return nil
}

// KnockoutStages returns the knockout stages in play order.
func (s *Structure) KnockoutStages() []Stage {
	out := make([]Stage, 0, len(s.Stages))
	for _, st := range s.Stages {
		if st.Kind == KindKnockout {
			out = append(out, st)
		}
	}
	return out
}

// ChampionStage returns the last non-consolation knockout stage — the match
// whose winner is the tournament champion — or nil for group-only formats.
func (s *Structure) ChampionStage() *Stage {
	for i := len(s.Stages) - 1; i >= 0; i-- {
		if s.Stages[i].Kind == KindKnockout && !s.Stages[i].Consolation {
			return &s.Stages[i]
		}
	}
	return nil
}

// Stage returns the stage with the given code, or nil.
func (s *Structure) Stage(code string) *Stage {
	for i := range s.Stages {
		if s.Stages[i].Code == code {
			return &s.Stages[i]
		}
	}
	return nil
}

// IsKnockout reports whether the given stage code is a knockout stage.
// Unknown codes count as knockout only if a group stage exists and the code
// isn't its code — callers should prefer validated data.
func (s *Structure) IsKnockout(code string) bool {
	if st := s.Stage(code); st != nil {
		return st.Kind == KindKnockout
	}
	return false
}

// StageCodes returns all stage codes in play order.
func (s *Structure) StageCodes() []string {
	out := make([]string, len(s.Stages))
	for i, st := range s.Stages {
		out[i] = st.Code
	}
	return out
}

// Sync providers.
const (
	ProviderAuto         = "auto" // API-Football when a key reaches the season, else openfootball
	ProviderAPIFootball  = "api-football"
	ProviderOpenfootball = "openfootball"
	ProviderManual       = "manual"
)

// Sync is the parsed `sync` JSON of a tournament record.
type Sync struct {
	Provider          string `json:"provider"`
	APIFootballLeague int    `json:"apiFootballLeague,omitempty"`
	Season            int    `json:"season,omitempty"`
	OpenfootballURL   string `json:"openfootballURL,omitempty"`
}

// Validate checks the sync config.
func (s *Sync) Validate() error {
	switch s.Provider {
	case ProviderAuto, ProviderAPIFootball, ProviderOpenfootball, ProviderManual, "":
	default:
		return fmt.Errorf("sync: unknown provider %q", s.Provider)
	}
	return nil
}
