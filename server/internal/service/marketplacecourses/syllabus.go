package marketplacecourses

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	courserepo "github.com/lextures/lextures/server/internal/repos/course"
)

// SyllabusFixture is the parsed course syllabus for a locale.
type SyllabusFixture struct {
	RequireAcceptance bool
	Sections          []courserepo.SyllabusSection
	Locale            string
	FilePath          string
}

// LoadSyllabus parses embedded syllabus.json for locale (falls back to English).
func LoadSyllabus(courseDirSlug, locale string) (*SyllabusFixture, error) {
	loc := strings.TrimSpace(locale)
	if loc == "" {
		loc = defaultLocale
	}
	filePath := path.Join("content", courseDirSlug, loc, "syllabus.json")
	fixture, err := loadSyllabusFile(filePath, loc)
	if err == nil {
		return fixture, nil
	}
	if loc != defaultLocale {
		return LoadSyllabus(courseDirSlug, defaultLocale)
	}
	return nil, fmt.Errorf("marketplace course %s syllabus: locale %q missing", courseDirSlug, loc)
}

func loadSyllabusFile(filePath, locale string) (*SyllabusFixture, error) {
	b, err := contentFS.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var raw struct {
		RequireAcceptance bool                         `json:"require_acceptance"`
		Sections          []courserepo.SyllabusSection `json:"sections"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", filePath, err)
	}
	return &SyllabusFixture{
		RequireAcceptance: raw.RequireAcceptance,
		Sections:          raw.Sections,
		Locale:            locale,
		FilePath:          filePath,
	}, nil
}

// ValidateSyllabus checks fixture structure and links.
func ValidateSyllabus(s *SyllabusFixture, internalSlugs map[string]struct{}) *ValidationError {
	var ve ValidationError
	if s == nil {
		ve.add("syllabus is nil")
		return &ve
	}
	if len(s.Sections) == 0 {
		ve.add(s.FilePath + ": at least one syllabus section is required")
	}
	seen := make(map[string]struct{})
	for i, sec := range s.Sections {
		id := strings.TrimSpace(sec.ID)
		if id == "" {
			ve.add(fmt.Sprintf("%s: section %d missing id", s.FilePath, i))
			continue
		}
		if _, dup := seen[id]; dup {
			ve.add(fmt.Sprintf("%s: duplicate section id %q", s.FilePath, id))
		}
		seen[id] = struct{}{}
		if strings.TrimSpace(sec.Heading) == "" {
			ve.add(fmt.Sprintf("%s: section %q missing heading", s.FilePath, id))
		}
		if strings.TrimSpace(sec.Markdown) == "" {
			ve.add(fmt.Sprintf("%s: section %q missing markdown", s.FilePath, id))
		}
		if scriptTagRE.MatchString(sec.Markdown) {
			ve.add(fmt.Sprintf("%s: section %q: script tags are not allowed", s.FilePath, id))
		}
		for _, href := range extractLinks(sec.Markdown) {
			if err := validateLink(href, internalSlugs); err != nil {
				ve.add(fmt.Sprintf("%s: section %q: %v", s.FilePath, id, err))
			}
		}
	}
	if len(ve.Errors) == 0 {
		return nil
	}
	return &ve
}

func syncSyllabus(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, fixture *SyllabusFixture) error {
	if fixture == nil {
		return fmt.Errorf("syllabus is nil")
	}
	return upsertSyllabusTx(ctx, tx, courseID, fixture.Sections, fixture.RequireAcceptance)
}

func syllabusUpToDate(ctx context.Context, tx pgx.Tx, courseID uuid.UUID, fixture *SyllabusFixture) (bool, error) {
	if fixture == nil {
		return false, fmt.Errorf("syllabus is nil")
	}
	var raw []byte
	var require bool
	err := tx.QueryRow(ctx, `
SELECT COALESCE(cs.sections, '[]'::jsonb), COALESCE(cs.require_syllabus_acceptance, false)
FROM course.course_syllabus cs
WHERE cs.course_id = $1
`, courseID).Scan(&raw, &require)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if require != fixture.RequireAcceptance {
		return false, nil
	}
	var stored []courserepo.SyllabusSection
	if err := json.Unmarshal(raw, &stored); err != nil {
		return false, nil
	}
	return syllabusSectionsEqual(stored, fixture.Sections), nil
}

func syllabusSectionsEqual(a, b []courserepo.SyllabusSection) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ID != b[i].ID || a[i].Heading != b[i].Heading || a[i].Markdown != b[i].Markdown {
			return false
		}
	}
	return true
}

func upsertSyllabusTx(
	ctx context.Context,
	tx pgx.Tx,
	courseID uuid.UUID,
	sections []courserepo.SyllabusSection,
	requireSyllabusAcceptance bool,
) error {
	if sections == nil {
		sections = []courserepo.SyllabusSection{}
	}
	raw, err := json.Marshal(sections)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO course.course_syllabus (course_id, sections, require_syllabus_acceptance, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (course_id) DO UPDATE SET
    sections = EXCLUDED.sections,
    require_syllabus_acceptance = EXCLUDED.require_syllabus_acceptance,
    updated_at = NOW()
`, courseID, raw, requireSyllabusAcceptance)
	return err
}
