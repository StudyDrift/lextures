package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/assignmentrubric"
)

func canvasRubricIDFromAssignment(obj map[string]any) int64 {
	if obj == nil {
		return 0
	}
	settings, _ := obj["rubric_settings"].(map[string]any)
	if settings == nil {
		return 0
	}
	return int64At(settings, "id")
}

func canvasOptionalRubricJSONFromAssignment(obj map[string]any) ([]byte, error) {
	if obj == nil {
		return nil, nil
	}
	criteria := arrAt(obj, "rubric")
	title := optionalTrimmedString(strAt(obj, "name", ""))
	if custom := optionalTrimmedString(strAt(obj, "rubric_title", "")); custom != nil {
		title = custom
	}
	if len(criteria) == 0 {
		return nil, nil
	}
	def, err := canvasRubricDefinitionFromCriteria(criteria, title)
	if err != nil || def == nil {
		return nil, err
	}
	return json.Marshal(def)
}

func canvasRubricDefinitionFromCriteria(criteria []map[string]any, title *string) (*assignmentrubric.RubricDefinition, error) {
	out := make([]assignmentrubric.RubricCriterion, 0, len(criteria))
	for _, crit := range criteria {
		if crit == nil {
			continue
		}
		if boolAt(crit, "ignore_for_scoring", false) {
			continue
		}
		criterionTitle := strings.TrimSpace(strAt(crit, "description", ""))
		if criterionTitle == "" {
			criterionTitle = strings.TrimSpace(strAt(crit, "title", ""))
		}
		if criterionTitle == "" {
			continue
		}
		var description *string
		if long := optionalTrimmedString(strAt(crit, "long_description", "")); long != nil {
			description = long
		}
		levels := canvasRubricLevelsFromRatings(arrAt(crit, "ratings"))
		if len(levels) == 0 {
			continue
		}
		out = append(out, assignmentrubric.RubricCriterion{
			ID:          uuid.New(),
			Title:       criterionTitle,
			Description: description,
			Levels:      levels,
		})
	}
	if len(out) == 0 {
		return nil, nil
	}
	def := &assignmentrubric.RubricDefinition{Criteria: out}
	if title != nil {
		def.Title = title
	}
	if err := assignmentrubric.ValidateRubricDefinition(def); err != nil {
		return nil, err
	}
	return def, nil
}

func canvasRubricLevelsFromRatings(ratings []map[string]any) []assignmentrubric.RubricLevel {
	type levelSort struct {
		level assignmentrubric.RubricLevel
		idx   int
	}
	sorted := make([]levelSort, 0, len(ratings))
	for i, rating := range ratings {
		if rating == nil {
			continue
		}
		pts, ok := coerceCanvasJSONNumber(rating["points"])
		if !ok || pts < 0 {
			continue
		}
		label := strings.TrimSpace(strAt(rating, "description", ""))
		if label == "" {
			label = strings.TrimSpace(strAt(rating, "title", ""))
		}
		if label == "" {
			label = formatCanvasRubricPointsLabel(pts)
		}
		var desc *string
		if long := optionalTrimmedString(strAt(rating, "long_description", "")); long != nil {
			desc = long
		}
		sorted = append(sorted, levelSort{
			idx: i,
			level: assignmentrubric.RubricLevel{
				Label:       label,
				Points:      pts,
				Description: desc,
			},
		})
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].level.Points != sorted[j].level.Points {
			return sorted[i].level.Points > sorted[j].level.Points
		}
		return sorted[i].idx < sorted[j].idx
	})
	out := make([]assignmentrubric.RubricLevel, 0, len(sorted))
	for _, row := range sorted {
		out = append(out, row.level)
	}
	return out
}

func formatCanvasRubricPointsLabel(points float64) string {
	if points == float64(int64(points)) {
		return fmt.Sprintf("%d pts", int64(points))
	}
	return fmt.Sprintf("%g pts", points)
}

func optionalTrimmedString(s string) *string {
	t := strings.TrimSpace(s)
	if t == "" {
		return nil
	}
	return &t
}

func nullableJSONBytes(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return b
}

func canvasEnrichAssignmentWithRubric(
	ctx context.Context,
	client *http.Client,
	canvasBase, accessToken string,
	canvasCourseID int64,
	obj map[string]any,
) error {
	if obj == nil || len(arrAt(obj, "rubric")) > 0 {
		return nil
	}
	rubricID := canvasRubricIDFromAssignment(obj)
	if rubricID <= 0 {
		return nil
	}
	rubricObj, err := canvasGetObject(ctx, client, canvasBase, accessToken,
		fmt.Sprintf("courses/%d/rubrics/%d", canvasCourseID, rubricID), nil)
	if err != nil || rubricObj == nil {
		return err
	}
	if data := arrAt(rubricObj, "data"); len(data) > 0 {
		obj["rubric"] = data
	}
	if title := strAt(rubricObj, "title", ""); title != "" {
		obj["rubric_title"] = title
	}
	return nil
}
