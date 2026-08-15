package tournaments

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
)

// Forecast modes (the tournament's `forecastSpec` JSON). The pre-tournament
// Hail-Mary stays, but its SHAPE is an admin decision per tournament:
// the full ceremonial builder for a WC, a short list of headline calls for a
// league season, or none at all (tips-only).
const (
	ForecastFull  = "full"
	ForecastCalls = "calls"
	ForecastNone  = "none"
)

// Call types.
const (
	CallTeam    = "team"    // one pick: the champion (knockout winner, or table position 1)
	CallTeamset = "teamset" // an unordered set: a table zone or a reached knockout stage
)

// Call is one headline pick of a calls-mode forecast, e.g. "Champion",
// "Champions League spots", "Relegated", "Final four".
type Call struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Points int    `json:"points"`
	// teamset targets — exactly one of:
	Zone  string `json:"zone,omitempty"`  // structure zone key (final-table range)
	Stage string `json:"stage,omitempty"` // knockout stage code ("reached the SF")
	// teamset size; derived from the zone when zone-targeted.
	Count int `json:"count,omitempty"`
}

// ForecastSpec is the parsed `forecastSpec` JSON of a tournament record.
// A missing/empty spec means "full" (the pre-rework behavior).
type ForecastSpec struct {
	Mode  string `json:"mode"`
	Calls []Call `json:"calls,omitempty"`
}

// ForecastSpecOf parses a record's forecastSpec (missing → full mode).
func ForecastSpecOf(rec *core.Record) (*ForecastSpec, error) {
	f := &ForecastSpec{}
	if err := rec.UnmarshalJSONField("forecastSpec", f); err != nil || f.Mode == "" {
		return &ForecastSpec{Mode: ForecastFull}, nil
	}
	return f, nil
}

// Validate checks the spec against the tournament's structure (zone/stage
// references must exist; zone-targeted counts are filled in).
func (f *ForecastSpec) Validate(st *Structure) error {
	switch f.Mode {
	case ForecastFull, ForecastNone:
		if len(f.Calls) > 0 {
			return fmt.Errorf("forecastSpec: calls are only valid in calls mode")
		}
		return nil
	case ForecastCalls:
	default:
		return fmt.Errorf("forecastSpec: unknown mode %q", f.Mode)
	}
	if len(f.Calls) == 0 || len(f.Calls) > 12 {
		return fmt.Errorf("forecastSpec: calls mode needs 1..12 calls")
	}
	seen := map[string]bool{}
	for i := range f.Calls {
		c := &f.Calls[i]
		if !stageCodeRe.MatchString(c.Key) {
			return fmt.Errorf("forecastSpec: call %d has invalid key %q", i, c.Key)
		}
		if seen[c.Key] {
			return fmt.Errorf("forecastSpec: duplicate call %q", c.Key)
		}
		seen[c.Key] = true
		if c.Name == "" || len(c.Name) > 64 {
			return fmt.Errorf("forecastSpec: call %q needs a name (max 64 chars)", c.Key)
		}
		if c.Points < 1 {
			return fmt.Errorf("forecastSpec: call %q needs points >= 1", c.Key)
		}
		switch c.Type {
		case CallTeam:
			if c.Zone != "" || c.Stage != "" || c.Count != 0 {
				return fmt.Errorf("forecastSpec: team call %q takes no zone/stage/count", c.Key)
			}
		case CallTeamset:
			switch {
			case c.Zone != "" && c.Stage != "":
				return fmt.Errorf("forecastSpec: teamset %q: zone or stage, not both", c.Key)
			case c.Zone != "":
				z := st.Zone(c.Zone)
				if z == nil {
					return fmt.Errorf("forecastSpec: teamset %q references unknown zone %q", c.Key, c.Zone)
				}
				c.Count = z.To - z.From + 1
			case c.Stage != "":
				sg := st.Stage(c.Stage)
				if sg == nil || sg.Kind != KindKnockout {
					return fmt.Errorf("forecastSpec: teamset %q references unknown knockout stage %q", c.Key, c.Stage)
				}
				if c.Count < 1 || c.Count > 32 {
					return fmt.Errorf("forecastSpec: stage teamset %q needs count 1..32", c.Key)
				}
			default:
				return fmt.Errorf("forecastSpec: teamset %q needs a zone or a stage", c.Key)
			}
		default:
			return fmt.Errorf("forecastSpec: call %q has unknown type %q", c.Key, c.Type)
		}
	}
	return nil
}

// Relations derives the logical links between calls from the structure, so
// the UI can auto-fill / lock implied picks and reject impossible ones, and
// the backend can validate the same way:
//
//   - implies[a] lists calls b such that every member of a is necessarily a
//     member of b: the champion (position 1) is inside any zone that covers
//     position 1; a zone nested in a wider zone; a later knockout stage inside
//     an earlier one ("reached the SF" ⇒ "reached the QF"); the champion
//     reaches every stage.
//   - exclusive[a] lists calls b whose zone ranges don't overlap with a's, so
//     a team can't be picked in both.
//
// Only positional (zone) and stage-based calls relate; a team call in a
// group-less knockout cup relates to stage teamsets, in a league shape to
// zones. Maps are symmetric for exclusive, directed for implies.
func (f *ForecastSpec) Relations(st *Structure) (implies map[string][]string, exclusive map[string][]string) {
	implies = map[string][]string{}
	exclusive = map[string][]string{}
	if f.Mode != ForecastCalls {
		return
	}
	// Position range per call ([1,1] for the champion of a table shape;
	// zone range for zone teamsets). Stage depth per call (index in play
	// order; champion = beyond the last stage).
	type rng struct{ from, to int }
	pos := map[string]rng{}
	depth := map[string]int{}
	ko := st.KnockoutStages()
	stageIdx := map[string]int{}
	for i, s := range ko {
		stageIdx[s.Code] = i
	}
	for _, c := range f.Calls {
		switch {
		case c.Type == CallTeam:
			if len(ko) > 0 {
				depth[c.Key] = len(ko) // champion: reached every stage
			} else if st.HasGroups() {
				pos[c.Key] = rng{1, 1}
			}
		case c.Zone != "":
			if z := st.Zone(c.Zone); z != nil {
				pos[c.Key] = rng{z.From, z.To}
			}
		case c.Stage != "":
			if i, ok := stageIdx[c.Stage]; ok {
				depth[c.Key] = i
			}
		}
	}
	for _, a := range f.Calls {
		for _, b := range f.Calls {
			if a.Key == b.Key {
				continue
			}
			if ra, ok := pos[a.Key]; ok {
				if rb, ok := pos[b.Key]; ok {
					switch {
					case ra.from >= rb.from && ra.to <= rb.to && (ra != rb || a.Type == CallTeam):
						implies[a.Key] = append(implies[a.Key], b.Key)
					case ra.to < rb.from || rb.to < ra.from:
						exclusive[a.Key] = append(exclusive[a.Key], b.Key)
					}
				}
			}
			if da, ok := depth[a.Key]; ok {
				if db, ok := depth[b.Key]; ok && da > db {
					// a is deeper (later stage / champion) → a ⊂ b. Consolation
					// stages don't lie on the path to the title, so the
					// champion doesn't imply "reached the 3rd-place match".
					if b.Stage != "" {
						if s := st.Stage(b.Stage); s != nil && s.Consolation {
							continue
						}
					}
					implies[a.Key] = append(implies[a.Key], b.Key)
				}
			}
		}
	}
	return
}
