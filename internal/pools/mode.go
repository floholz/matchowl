package pools

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/floholz/matchowl/internal/clock"
	"github.com/floholz/matchowl/internal/tournaments"
)

// Pool modes. Classic is the one points table for the whole season; h2h
// pairs members up every matchday (see PLAN.md, "head-to-head pools").
const (
	ModeClassic = "classic"
	ModeH2H     = "h2h"
)

// Modes in display order, reused by the migration's select field.
var Modes = []string{ModeClassic, ModeH2H}

// Season returns the one season a pool plays ("" for Global, which binds
// none). The field is still a relation list for history's sake; it holds at
// most one id since the h2h rework.
func Season(lg *core.Record) string {
	if ts := lg.GetStringSlice("tournaments"); len(ts) > 0 {
		return ts[0]
	}
	return ""
}

// isLeague reports whether a season is a plain round-robin league: one
// group stage, no knockout after it, every team meeting every other at
// least once. That rules out the UCL league phase (8 games among 36) and
// every cup. Those default to classic; leagues default to head-to-head.
func isLeague(st *tournaments.Structure) bool {
	if st == nil || len(st.Stages) != 1 || st.Stages[0].Kind != tournaments.KindGroup {
		return false
	}
	return st.GroupSize > 1 && st.GamesPerTeam >= st.GroupSize-1
}

// Defaults returns the mode and save-call allowance a new pool for the
// season gets when the creator picks nothing: h2h for leagues, classic
// otherwise; one save call when a matchday has up to 6 matches, else two.
func Defaults(t *core.Record) (mode string, saveCalls int, matchesPerRound int) {
	st, _ := tournaments.StructureOf(t)
	mode = ModeClassic
	if isLeague(st) {
		mode = ModeH2H
	}
	if st != nil {
		matchesPerRound = st.GroupSize / 2
	}
	saveCalls = 1
	if matchesPerRound > 6 {
		saveCalls = 2
	}
	return
}

// resolveSettings fills empty mode / save-call values from the season's
// defaults and validates the result.
func resolveSettings(t *core.Record, mode string, saveCalls int) (string, int, error) {
	dm, ds, _ := Defaults(t)
	if mode == "" {
		mode = dm
	}
	if mode != ModeClassic && mode != ModeH2H {
		return "", 0, fmt.Errorf("unknown mode %q", mode)
	}
	if saveCalls == 0 {
		saveCalls = ds
	}
	if saveCalls < 1 || saveCalls > 2 {
		return "", 0, fmt.Errorf("save calls: 1 or 2")
	}
	return mode, saveCalls, nil
}

// LockAt is when a pool's season, mode and save-call settings freeze: the
// first kick-off of the pool's first round, i.e. the first matchday that
// had not started when the pool was created. Zero when the pool binds no
// season; the creation time itself when no round is left to play.
func LockAt(app core.App, lg *core.Record) time.Time {
	tid := Season(lg)
	if tid == "" {
		return time.Time{}
	}
	created := lg.GetDateTime("created").Time()
	rounds, err := tournaments.Rounds(app, tid)
	if err != nil {
		return created
	}
	if r := tournaments.FirstRoundAfter(rounds, created); r != nil {
		return r.First
	}
	return created
}

// Locked reports whether the pool's settings are frozen.
func Locked(app core.App, lg *core.Record) bool {
	la := LockAt(app, lg)
	return !la.IsZero() && !clock.Now(app).Before(la)
}

// modeView is the settings block every pool payload carries.
func modeView(app core.App, lg *core.Record) map[string]any {
	mode := lg.GetString("mode")
	if mode == "" {
		mode = ModeClassic
	}
	la := LockAt(app, lg)
	lockAt := ""
	if !la.IsZero() {
		lockAt = la.UTC().Format(time.RFC3339)
	}
	return map[string]any{
		"mode":      mode,
		"saveCalls": lg.GetInt("saveCalls"),
		"lockAt":    lockAt,
		"locked":    !la.IsZero() && !clock.Now(app).Before(la),
	}
}
