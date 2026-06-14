package course

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var heroObjectPositionRe = regexp.MustCompile(`^\d+(?:\.\d+)?%\s+\d+(?:\.\d+)?%$`)

// ValidHeroObjectPosition reports whether pos is a CSS object-position percentage pair.
func ValidHeroObjectPosition(pos string) bool {
	return heroObjectPositionRe.MatchString(strings.TrimSpace(pos))
}

// HeroImagePatch describes optional hero image field updates.
type HeroImagePatch struct {
	ImageURL             *string
	ObjectPosition       *string
	UpdateImageURL       bool
	UpdateObjectPosition bool
}

// SetHeroImage applies partial hero image updates for the given course_code.
func SetHeroImage(ctx context.Context, pool *pgxpool.Pool, courseCode string, patch HeroImagePatch) (*CoursePublic, error) {
	if !patch.UpdateImageURL && !patch.UpdateObjectPosition {
		return GetPublicByCourseCode(ctx, pool, courseCode)
	}

	setClauses := make([]string, 0, 2)
	args := make([]any, 0, 3)
	argN := 1

	if patch.UpdateImageURL {
		setClauses = append(setClauses, "hero_image_url = $"+strconv.Itoa(argN))
		args = append(args, patch.ImageURL)
		argN++
	}
	if patch.UpdateObjectPosition {
		setClauses = append(setClauses, "hero_image_object_position = $"+strconv.Itoa(argN))
		args = append(args, patch.ObjectPosition)
		argN++
	}
	setClauses = append(setClauses, "updated_at = NOW()")

	args = append(args, courseCode)
	q := "UPDATE course.courses SET " + strings.Join(setClauses, ", ") + " WHERE course_code = $" + strconv.Itoa(argN)

	tag, err := pool.Exec(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, nil
	}
	return GetPublicByCourseCode(ctx, pool, courseCode)
}