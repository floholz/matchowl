// Package bracket holds official "extra qualifier" slot-allocation tables,
// keyed by the tournament structure's extraQualifiers.tableKey. The "wc2026"
// entry is FIFA's 2026 best-third → Round-of-32 table (Annex C of the
// tournament regulations): for each of the 495 combinations of 8 qualifying
// third-placed groups, which group's third faces each group winner.
// Tournaments without an official table fall back to the deterministic
// greedy allocation in internal/sync.
package bracket

import (
	"embed"
	"encoding/json"
	"path"
	"sort"
	"strings"
)

//go:embed tables
var tablesFS embed.FS

// registry[tableKey][sortedQualifierLetters] = { winnerGroupLetter: thirdGroupLetter }
var registry = map[string]map[string]map[string]string{}

func init() {
	entries, err := tablesFS.ReadDir("tables")
	if err != nil {
		panic("bracket: " + err.Error())
	}
	for _, e := range entries {
		key := strings.TrimSuffix(e.Name(), ".json")
		raw, err := tablesFS.ReadFile(path.Join("tables", e.Name()))
		if err != nil {
			panic("bracket: " + err.Error())
		}
		var t map[string]map[string]string
		if err := json.Unmarshal(raw, &t); err != nil {
			panic("bracket: bad table " + e.Name() + ": " + err.Error())
		}
		registry[key] = t
	}
}

// Key normalises a set of qualifying extra-qualifier group letters to the
// table key (sorted, upper-case, deduped).
func Key(groups []string) string {
	seen := map[string]bool{}
	out := make([]string, 0, len(groups))
	for _, g := range groups {
		g = strings.ToUpper(strings.TrimSpace(g))
		if g != "" && !seen[g] {
			seen[g] = true
			out = append(out, g)
		}
	}
	sort.Strings(out)
	return strings.Join(out, "")
}

// ThirdFor returns the group letter whose extra-qualifier team faces the
// given group winner, per the named official table. ok is false when the
// table doesn't exist or the qualifier combination isn't in it (callers fall
// back to a deterministic matching).
func ThirdFor(tableKey string, qualifiers []string, winner string) (string, bool) {
	m, ok := Lookup(tableKey, qualifiers)
	if !ok {
		return "", false
	}
	g, ok := m[strings.ToUpper(winner)]
	return g, ok
}

// Lookup returns the full winner→thirdGroup map for a qualifier combination
// in the named table.
func Lookup(tableKey string, qualifiers []string) (map[string]string, bool) {
	t, ok := registry[tableKey]
	if !ok {
		return nil, false
	}
	m, ok := t[Key(qualifiers)]
	return m, ok
}

// WinnerLetter returns the group-winner letter ("1X" -> "X") of a knockout
// match's two labels, i.e. the side an extra-qualifier team is drawn against.
func WinnerLetter(homeLabel, awayLabel string) (string, bool) {
	for _, l := range []string{homeLabel, awayLabel} {
		if len(l) == 2 && l[0] == '1' && l[1] >= 'A' && l[1] <= 'Z' {
			return string(l[1]), true
		}
	}
	return "", false
}

// Table exposes a whole official table (served to the frontend so its
// Forecast bracket uses the identical allocation). nil when the key has no
// registered table.
func Table(tableKey string) map[string]map[string]string { return registry[tableKey] }
