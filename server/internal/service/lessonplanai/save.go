package lessonplanai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/coursemoduleassignments"
	"github.com/lextures/lextures/server/internal/repos/coursemodulecontent"
	"github.com/lextures/lextures/server/internal/repos/coursemodulequizzes"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// SaveAcceptedOptions configures persisting accepted components to a draft module (FR-5).
type SaveAcceptedOptions struct {
	CourseID            uuid.UUID
	ModuleTitle         string
	AcceptedKeys        []string
	Components          []ComponentSlot
	ComponentEdits      map[string]json.RawMessage // optional instructor edits keyed by component
}

// SaveResult is returned after save-to-course.
type SaveResult struct {
	ModuleID uuid.UUID `json:"module_id"`
}

// SaveToCourse creates a draft module with accepted assets (FR-5, AC-4).
func SaveToCourse(ctx context.Context, pool *pgxpool.Pool, opts SaveAcceptedOptions) (SaveResult, error) {
	if pool == nil {
		return SaveResult{}, errors.New("lessonplanai: nil pool")
	}
	if len(opts.AcceptedKeys) == 0 {
		return SaveResult{}, fmt.Errorf("accepted_components is required")
	}
	title := strings.TrimSpace(opts.ModuleTitle)
	if title == "" {
		title = "AI Lesson (draft)"
	}

	mod, err := coursestructure.CreateDraftModule(ctx, pool, opts.CourseID, title)
	if err != nil {
		return SaveResult{}, err
	}

	compByKey := map[string]ComponentSlot{}
	for _, c := range opts.Components {
		compByKey[c.Key] = c
	}

	for _, key := range opts.AcceptedKeys {
		slot, ok := compByKey[key]
		if !ok || slot.Status != StatusCompleted {
			continue
		}
		content := slot.Content
		if edited, ok := opts.ComponentEdits[key]; ok && len(edited) > 0 {
			content = edited
		}
		provJSON, _ := json.Marshal(slot.Provenance)
		if err := saveComponent(ctx, pool, opts.CourseID, mod.ID, key, content, provJSON); err != nil {
			return SaveResult{}, err
		}
	}

	return SaveResult{ModuleID: mod.ID}, nil
}

func saveComponent(ctx context.Context, pool *pgxpool.Pool, courseID, moduleID uuid.UUID, key string, content, provenance json.RawMessage) error {
	switch {
	case key == ComponentLessonPlan:
		var c LessonPlanContent
		if err := json.Unmarshal(content, &c); err != nil {
			return err
		}
		row, err := coursestructure.InsertContentPageUnderModule(ctx, pool, courseID, moduleID, "Lesson Plan")
		if err != nil {
			return err
		}
		if err := coursestructure.SetItemPublished(ctx, pool, courseID, row.ID, false); err != nil {
			return err
		}
		if len(provenance) > 0 {
			if err := coursestructure.SetItemProvenance(ctx, pool, courseID, row.ID, provenance); err != nil {
				return err
			}
		}
		_, err = coursemodulecontent.PatchContentPage(ctx, pool, courseID, row.ID, c.Markdown, false, nil)
		return err

	case key == ComponentQuiz:
		var c QuizContent
		if err := json.Unmarshal(content, &c); err != nil {
			return err
		}
		row, err := coursestructure.InsertQuizUnderModule(ctx, pool, courseID, moduleID, "Formative Quiz")
		if err != nil {
			return err
		}
		if err := coursestructure.SetItemPublished(ctx, pool, courseID, row.ID, false); err != nil {
			return err
		}
		if len(provenance) > 0 {
			if err := coursestructure.SetQuizProvenance(ctx, pool, courseID, row.ID, provenance); err != nil {
				return err
			}
		}
		_, err = coursemodulequizzes.PatchForCourseItem(ctx, pool, courseID, row.ID, coursemodulequizzes.PatchWrite{
			Questions: &c.Questions,
		})
		return err

	case key == ComponentRubric:
		var c RubricContent
		if err := json.Unmarshal(content, &c); err != nil {
			return err
		}
		row, err := coursestructure.InsertAssignmentUnderModule(ctx, pool, courseID, moduleID, "Open-Ended Task")
		if err != nil {
			return err
		}
		if err := coursestructure.SetItemPublished(ctx, pool, courseID, row.ID, false); err != nil {
			return err
		}
		if len(provenance) > 0 {
			if err := coursestructure.SetItemProvenance(ctx, pool, courseID, row.ID, provenance); err != nil {
				return err
			}
		}
		rubricJSON, err := json.Marshal(c.Rubric)
		if err != nil {
			return err
		}
		raw := json.RawMessage(rubricJSON)
		_, err = coursemoduleassignments.PatchForCourseItem(ctx, pool, courseID, row.ID, coursemoduleassignments.PatchWrite{
			RubricJSON: &raw,
		})
		return err

	case strings.HasPrefix(key, ComponentActivityPrefix):
		var c ActivityContent
		if err := json.Unmarshal(content, &c); err != nil {
			return err
		}
		label := activityTitle(c.Level)
		row, err := coursestructure.InsertContentPageUnderModule(ctx, pool, courseID, moduleID, label)
		if err != nil {
			return err
		}
		if err := coursestructure.SetItemPublished(ctx, pool, courseID, row.ID, false); err != nil {
			return err
		}
		if len(provenance) > 0 {
			if err := coursestructure.SetItemProvenance(ctx, pool, courseID, row.ID, provenance); err != nil {
				return err
			}
		}
		_, err = coursemodulecontent.PatchContentPage(ctx, pool, courseID, row.ID, c.Markdown, false, nil)
		return err

	default:
		return fmt.Errorf("unknown component: %s", key)
	}
}

func activityTitle(level string) string {
	switch level {
	case "below_grade":
		return "Activity (Below Grade)"
	case "on_grade":
		return "Activity (On Grade)"
	case "advanced":
		return "Activity (Advanced)"
	case "ell":
		return "Activity (ELL)"
	case "iep":
		return "Activity (IEP)"
	default:
		return "Activity"
	}
}
