package migrations

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Leagues are pools now (PLAN.md §7): rename the collections, their `league`
// relation fields, indexes and rules, and the notification event keys that
// carried the old name ("league_lead", "league_chat") wherever they are
// stored (the notifications log and each user's notifyPrefs).
func init() {
	renames := []struct{ from, to string }{
		{"leagues", "pools"},
		{"league_members", "pool_members"},
		{"league_messages", "pool_messages"},
		{"league_reads", "pool_reads"},
	}
	rewrite := func(s string) string {
		r := strings.NewReplacer(
			"league_members_via_league", "pool_members_via_pool",
			"league_members", "pool_members",
			"league_messages", "pool_messages",
			"league_reads", "pool_reads",
			"idx_leagues_", "idx_pools_",
			"idx_lm_league_user", "idx_pm_pool_user",
			"idx_msg_league_created", "idx_msg_pool_created",
			"idx_reads_league_user", "idx_reads_pool_user",
			"`leagues`", "`pools`",
			"league.", "pool.",
			"(league,", "(pool,",
			"league_lead", "pool_lead",
			"league_chat", "pool_chat",
		)
		return r.Replace(s)
	}
	renameAll := func(app core.App, forward bool) error {
		for _, rn := range renames {
			from, to := rn.from, rn.to
			if !forward {
				from, to = to, from
			}
			col, err := app.FindCollectionByNameOrId(from)
			if err != nil {
				continue
			}
			col.Name = to
			// The relation to the pool.
			oldField, newField := "league", "pool"
			if !forward {
				oldField, newField = "pool", "league"
			}
			if f := col.Fields.GetByName(oldField); f != nil {
				f.SetName(newField)
			}
			fix := func(s string) string {
				if forward {
					return rewrite(s)
				}
				r := strings.NewReplacer(
					"pool_members_via_pool", "league_members_via_league",
					"pool_members", "league_members", "pool_messages", "league_messages", "pool_reads", "league_reads",
					"idx_pools_", "idx_leagues_", "idx_pm_pool_user", "idx_lm_league_user",
					"idx_msg_pool_created", "idx_msg_league_created", "idx_reads_pool_user", "idx_reads_league_user",
					"`pools`", "`leagues`", "pool.", "league.", "(pool,", "(league,",
				)
				return r.Replace(s)
			}
			for i, idx := range col.Indexes {
				col.Indexes[i] = fix(idx)
			}
			for _, rule := range []**string{&col.ListRule, &col.ViewRule, &col.CreateRule, &col.UpdateRule, &col.DeleteRule} {
				if *rule != nil {
					v := fix(**rule)
					*rule = &v
				}
			}
			if err := app.Save(col); err != nil {
				return err
			}
		}
		return nil
	}
	rekey := func(app core.App, from, to string) error {
		if recs, err := app.FindRecordsByFilter("notifications", "event = {:e} || dedupKey ~ {:k}", "", 0, 0,
			map[string]any{"e": from, "k": from + ":"}); err == nil {
			for _, r := range recs {
				if r.GetString("event") == from {
					r.Set("event", to)
				}
				r.Set("dedupKey", strings.Replace(r.GetString("dedupKey"), from+":", to+":", 1))
				if err := app.Save(r); err != nil {
					return err
				}
			}
		}
		users, err := app.FindRecordsByFilter("users", "notifyPrefs ~ {:e}", "", 0, 0, map[string]any{"e": from})
		if err != nil {
			return nil
		}
		for _, u := range users {
			u.Set("notifyPrefs", strings.ReplaceAll(u.GetString("notifyPrefs"), from, to))
			if err := app.Save(u); err != nil {
				return err
			}
		}
		return nil
	}
	m.Register(func(app core.App) error {
		if err := renameAll(app, true); err != nil {
			return err
		}
		for _, k := range [][2]string{{"league_lead", "pool_lead"}, {"league_chat", "pool_chat"}} {
			if err := rekey(app, k[0], k[1]); err != nil {
				return err
			}
		}
		return nil
	}, func(app core.App) error {
		if err := renameAll(app, false); err != nil {
			return err
		}
		for _, k := range [][2]string{{"pool_lead", "league_lead"}, {"pool_chat", "league_chat"}} {
			if err := rekey(app, k[0], k[1]); err != nil {
				return err
			}
		}
		return nil
	})
}
