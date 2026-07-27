package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// survey_responses holds the one-time end-of-tournament feedback survey: one
// record per user with the validated answers as a JSON blob. The questions are
// hard-coded in the frontend (single survey, single run) and the answer shape
// is validated by the Go endpoint in internal/survey.
//
// No collection rules at all — raw responses are superuser-only (exported for
// analysis from the PB dashboard); users write through the validated endpoint
// and never read anyone's answers back. The unique index on user is the
// one-shot guard: a second submit fails at the DB layer.
const nSurveyResponses = "survey_responses"

func init() {
	m.Register(func(app core.App) error {
		if _, err := app.FindCollectionByNameOrId(nSurveyResponses); err == nil {
			return nil
		}
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection(nSurveyResponses)
		c.Fields.Add(&core.RelationField{
			Name:          "user",
			Required:      true,
			CollectionId:  users.Id,
			MaxSelect:     1,
			CascadeDelete: true,
		})
		c.Fields.Add(&core.JSONField{Name: "answers", Required: true, MaxSize: 8000})
		c.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		c.AddIndex("idx_survey_responses_user", true, "user", "")
		return app.Save(c)
	}, func(app core.App) error {
		if c, err := app.FindCollectionByNameOrId(nSurveyResponses); err == nil {
			return app.Delete(c)
		}
		return nil
	})
}
