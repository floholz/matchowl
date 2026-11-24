package dev

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/floholz/matchowl/internal/clock"
)

// The sports snapshot behind the dev seed: competitions, seasons, teams and
// matches exported from a real database (a copy of the alpha one) into
// internal/dev/data/snapshot.json.gz, which is committed. It holds no
// people — no users, tips, pools or chat — since the repo is public. Crests
// are not stored: the seed fetches them from the provider's media CDN (free,
// no API quota) by provider id.

// snapshotCollections are exported and imported in this order, so every
// relation points at a row that already exists.
var snapshotCollections = []string{
	"competitions", "tournaments", "team_identities", "teams",
	"tournament_groups", "matches", "match_events",
}

// SnapshotPath is where the committed snapshot lives, relative to the repo.
const SnapshotPath = "internal/dev/data/snapshot.json.gz"

type snapshot struct {
	TakenAt     time.Time                    `json:"takenAt"`
	Collections map[string][]map[string]any  `json:"collections"`
	Logos       map[string]map[string]string `json:"logos"` // collection → record id → crest URL
	Counts      map[string]int               `json:"counts"`
}

// logoURL is the provider CDN crest of a row, or "".
func logoURL(col string, r *core.Record) string {
	switch col {
	case "teams":
		if r.GetString("provider") == "api-football" && r.GetInt("providerId") != 0 {
			return fmt.Sprintf("https://media.api-sports.io/football/teams/%d.png", r.GetInt("providerId"))
		}
	case "competitions":
		if id := r.GetInt("apiFootballLeague"); id != 0 {
			return fmt.Sprintf("https://media.api-sports.io/football/leagues/%d.png", id)
		}
	}
	return ""
}

// ExportSnapshot writes the sports data of app to path (gzip JSON). The
// snapshot's clock is the dev clock when one is pinned, else now.
func ExportSnapshot(app core.App, path string) (*snapshot, error) {
	takenAt := time.Now().UTC()
	if t, ok := clock.Sim(app); ok {
		takenAt = t
	}
	s := &snapshot{TakenAt: takenAt, Collections: map[string][]map[string]any{}, Logos: map[string]map[string]string{}, Counts: map[string]int{}}
	for _, name := range snapshotCollections {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			continue // older database without it
		}
		recs, err := app.FindAllRecords(col)
		if err != nil {
			return nil, err
		}
		rows := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			row := map[string]any{"id": r.Id}
			for _, f := range col.Fields {
				switch f.Type() {
				case core.FieldTypeAutodate, core.FieldTypeFile, core.FieldTypePassword:
					continue
				}
				if f.GetName() == "id" {
					continue
				}
				row[f.GetName()] = r.Get(f.GetName())
			}
			if u := logoURL(name, r); u != "" && r.GetString("logo") != "" {
				if s.Logos[name] == nil {
					s.Logos[name] = map[string]string{}
				}
				s.Logos[name][r.Id] = u
			}
			rows = append(rows, row)
		}
		s.Collections[name] = rows
		s.Counts[name] = len(rows)
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	enc := json.NewEncoder(gz)
	if err := enc.Encode(s); err != nil {
		return nil, err
	}
	return s, gz.Close()
}

func readSnapshot(path string) (*snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(gz)
	if err != nil {
		return nil, err
	}
	s := &snapshot{}
	return s, json.Unmarshal(b, s)
}

// importSnapshot loads a snapshot into an empty database, moving every date
// by shift. Unless keepSync, seasons are switched to manual sync: moved
// dates no longer match the provider's, and the sync would drag them back.
// Returns the number of rows written.
func importSnapshot(app core.App, s *snapshot, shift time.Duration, keepSync bool) (int, error) {
	// A fresh database already holds what the migrations seed (the World
	// Cup competition and season): the snapshot replaces it.
	for i := len(snapshotCollections) - 1; i >= 0; i-- {
		col, err := app.FindCollectionByNameOrId(snapshotCollections[i])
		if err != nil {
			continue
		}
		recs, _ := app.FindAllRecords(col)
		for _, r := range recs {
			if err := app.Delete(r); err != nil {
				return 0, fmt.Errorf("clear %s: %w", col.Name, err)
			}
		}
	}
	n := 0
	for _, name := range snapshotCollections {
		rows := s.Collections[name]
		if len(rows) == 0 {
			continue
		}
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			return n, err
		}
		dates := map[string]bool{}
		known := map[string]bool{}
		for _, f := range col.Fields {
			known[f.GetName()] = true
			if f.Type() == core.FieldTypeDate {
				dates[f.GetName()] = true
			}
		}
		err = app.RunInTransaction(func(tx core.App) error {
			for _, row := range rows {
				rec := core.NewRecord(col)
				rec.Id = fmt.Sprint(row["id"])
				for k, v := range row {
					if k == "id" || !known[k] {
						continue
					}
					if dates[k] {
						if str, _ := v.(string); str != "" {
							if dt, err := types.ParseDateTime(str); err == nil {
								v = dt.Time().Add(shift)
							}
						}
					}
					rec.Set(k, v)
				}
				if name == "tournaments" {
					if !keepSync {
						manualSync(rec)
					}
					// Scoring config ids are random per database.
					if _, err := tx.FindRecordById("scoring_configs", rec.GetString("scoringConfig")); err != nil {
						if def, err := tx.FindFirstRecordByFilter("scoring_configs", "isDefault = true"); err == nil {
							rec.Set("scoringConfig", def.Id)
						}
					}
				}
				if err := tx.SaveNoValidate(rec); err != nil {
					return fmt.Errorf("%s %s: %w", name, rec.Id, err)
				}
				n++
			}
			return nil
		})
		if err != nil {
			return n, err
		}
	}
	return n, nil
}

// manualSync switches a season's sync provider to manual.
func manualSync(t *core.Record) {
	cfg := map[string]any{}
	_ = t.UnmarshalJSONField("sync", &cfg)
	cfg["provider"] = "manual"
	t.Set("sync", cfg)
}

// attachLogos fetches the snapshot's crests from the provider CDN, a few at
// a time. Best-effort: offline or a missing crest just leaves the code chip.
func attachLogos(app core.App, s *snapshot) int {
	type job struct{ col, id, url string }
	var jobs []job
	for col, m := range s.Logos {
		for id, u := range m {
			jobs = append(jobs, job{col, id, u})
		}
	}
	ch := make(chan job)
	var mu sync.Mutex
	var wg sync.WaitGroup
	done := 0
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ch {
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				f, err := filesystem.NewFileFromURL(ctx, j.url)
				cancel()
				if err != nil {
					continue
				}
				mu.Lock()
				if rec, err := app.FindRecordById(j.col, j.id); err == nil {
					rec.Set("logo", f)
					if err := app.SaveNoValidate(rec); err == nil {
						done++
					} else {
						log.Printf("[devseed] logo %s: %v", j.id, err)
					}
				}
				mu.Unlock()
			}
		}()
	}
	for _, j := range jobs {
		ch <- j
	}
	close(ch)
	wg.Wait()
	return done
}
