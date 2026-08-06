package tournaments

import (
	"github.com/pocketbase/pocketbase/core"
)

// StructureCache lazily loads and memoizes tournament structures by
// tournament id, for call sites that process many records in one pass
// (recompute, sync, tips views).
type StructureCache struct {
	app core.App
	m   map[string]*Structure
}

func NewStructureCache(app core.App) *StructureCache {
	return &StructureCache{app: app, m: map[string]*Structure{}}
}

// For returns the structure of the given tournament id, or nil when the
// record or its structure can't be loaded.
func (c *StructureCache) For(tournamentID string) *Structure {
	if tournamentID == "" {
		return nil
	}
	if s, ok := c.m[tournamentID]; ok {
		return s
	}
	var s *Structure
	if rec, err := c.app.FindRecordById(collection, tournamentID); err == nil {
		if parsed, err := StructureOf(rec); err == nil {
			s = parsed
		}
	}
	c.m[tournamentID] = s
	return s
}

// IsKnockoutMatch reports whether a match record's stage is knockout per its
// tournament's structure (fallback for unloadable structures: anything not
// literally "group").
func (c *StructureCache) IsKnockoutMatch(match *core.Record) bool {
	if s := c.For(match.GetString("tournament")); s != nil {
		return s.IsKnockout(match.GetString("stage"))
	}
	return match.GetString("stage") != "group"
}
