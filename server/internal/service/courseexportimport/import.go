package courseexportimport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/models/coursesyllabus"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

// Import errors (HTTP maps these to 400/404).
var (
	ErrInvalidInput = errors.New("invalid import input")
)

// InvalidInput wraps a user-facing import validation message.
func InvalidInput(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidInput, msg)
}

// IsInvalidInput reports whether err is (or wraps) ErrInvalidInput.
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

// InvalidInputMessage extracts the message after ErrInvalidInput, or err.Error().
func InvalidInputMessage(err error) string {
	if err == nil {
		return ""
	}
	if !IsInvalidInput(err) {
		return err.Error()
	}
	msg := err.Error()
	prefix := ErrInvalidInput.Error() + ": "
	if strings.HasPrefix(msg, prefix) {
		return strings.TrimPrefix(msg, prefix)
	}
	return msg
}

const (
	maxExportEnrollments         = 5000
	maxEnrollmentEmailLen        = 320
	maxEnrollmentDisplayNameLen  = 256
	maxSyllabusSections          = 50
	maxSyllabusHeadingLen        = 512
	maxSyllabusMarkdownLen       = 200_000
	maxModuleContentMarkdownLen  = 200_000
	maxContentToolInstances      = 5000
	maxContentToolConfigBytes    = 262144 // matches content_tool_instances_config_size
)

// ApplyImport applies a course JSON export bundle to targetCourseCode (Rust apply_import).
// canvasInclude may be nil for pure JSON imports (all categories applied).
func ApplyImport(
	ctx context.Context,
	pool *pgxpool.Pool,
	targetCourseCode string,
	mode courseexport.CourseImportMode,
	ex *Bundle,
	canvasInclude *courseexport.CanvasImportInclude,
	blobOpts BlobOptions,
) error {
	if pool == nil {
		return errors.New("db pool is nil")
	}
	if ex == nil {
		return InvalidInput("Export body is required.")
	}
	targetCourseCode = strings.TrimSpace(targetCourseCode)
	if targetCourseCode == "" {
		return ErrNotFound
	}
	if mode != courseexport.CourseImportModeErase &&
		mode != courseexport.CourseImportModeMergeAdd &&
		mode != courseexport.CourseImportModeOverwrite {
		return InvalidInput("Invalid import mode.")
	}
	if err := validateExportPayload(ex); err != nil {
		return err
	}

	// Rewrite embedded file URLs when importing into a different course code.
	rewriteCourseFileURLsInBundle(ex, ex.CourseCode, targetCourseCode)

	courseIDPtr, err := course.GetIDByCourseCode(ctx, pool, targetCourseCode)
	if err != nil {
		return err
	}
	if courseIDPtr == nil {
		return ErrNotFound
	}
	courseID := *courseIDPtr

	applyGrades := true
	applySettings := true
	applyEnrollments := true
	applyFiles := true
	if canvasInclude != nil {
		applyGrades = canvasInclude.Grades
		applySettings = canvasInclude.Settings
		applyEnrollments = canvasInclude.Enrollments
		applyFiles = canvasInclude.Files
	}
	// JSON imports erase the outline even when the bundle has no structure rows.
	// Canvas partial imports skip that when every content category was unchecked.
	eraseOutlineBeforeApply := len(ex.Structure) > 0 || canvasInclude == nil
	overwritePruneStructure := len(ex.Structure) > 0 || canvasInclude == nil

	var mergeInserted map[uuid.UUID]struct{}

	switch mode {
	case courseexport.CourseImportModeErase:
		if eraseOutlineBeforeApply {
			if err := coursestructure.DeleteAllItemsForCourse(ctx, pool, courseID); err != nil {
				return err
			}
		}
		if applyGrades {
			if err := applyGradingFromExport(ctx, pool, targetCourseCode, &ex.Grading); err != nil {
				return err
			}
		}
		if applySettings {
			if err := applyCourseSnapshot(ctx, pool, targetCourseCode, &ex.Course); err != nil {
				return err
			}
			if _, err := course.UpsertSyllabus(ctx, pool, courseID, toRepoSyllabus(ex.Syllabus), ex.RequireSyllabusAcceptance); err != nil {
				return err
			}
		}
		for _, it := range ex.Structure {
			if _, err := coursestructure.ImportUpsertStructureItem(ctx, pool, courseID, it, false); err != nil {
				return err
			}
		}
		if err := applyModuleBodies(ctx, pool, courseID, ex, nil); err != nil {
			return err
		}

	case courseexport.CourseImportModeMergeAdd:
		if applyGrades {
			if err := mergeAddGradingGroups(ctx, pool, courseID, &ex.Grading); err != nil {
				return err
			}
		}
		if applySettings {
			if err := mergeSyllabusSections(ctx, pool, targetCourseCode, courseID, ex.Syllabus); err != nil {
				return err
			}
		}
		inserted := map[uuid.UUID]struct{}{}
		for _, it := range ex.Structure {
			ok, err := coursestructure.ImportUpsertStructureItem(ctx, pool, courseID, it, true)
			if err != nil {
				return err
			}
			if ok {
				if id, perr := uuid.Parse(it.ID); perr == nil {
					inserted[id] = struct{}{}
				}
			}
		}
		mergeInserted = inserted
		if err := applyModuleBodies(ctx, pool, courseID, ex, inserted); err != nil {
			return err
		}

	case courseexport.CourseImportModeOverwrite:
		if applyGrades {
			if err := applyGradingFromExport(ctx, pool, targetCourseCode, &ex.Grading); err != nil {
				return err
			}
		}
		if applySettings {
			if err := applyCourseSnapshot(ctx, pool, targetCourseCode, &ex.Course); err != nil {
				return err
			}
			if _, err := course.UpsertSyllabus(ctx, pool, courseID, toRepoSyllabus(ex.Syllabus), ex.RequireSyllabusAcceptance); err != nil {
				return err
			}
		}
		if overwritePruneStructure {
			keep := make([]uuid.UUID, 0, len(ex.Structure))
			for _, it := range ex.Structure {
				if id, perr := uuid.Parse(it.ID); perr == nil {
					keep = append(keep, id)
				}
			}
			if err := coursestructure.DeleteStructureNotInExport(ctx, pool, courseID, keep); err != nil {
				return err
			}
			for _, it := range ex.Structure {
				if _, err := coursestructure.ImportUpsertStructureItem(ctx, pool, courseID, it, false); err != nil {
					return err
				}
			}
			if err := applyModuleBodies(ctx, pool, courseID, ex, nil); err != nil {
				return err
			}
		}
	}

	// Content tools after structure + bodies so FKs and fence ids line up.
	// Canvas partial imports that skipped outline still apply CT when present in the bundle.
	if err := applyContentTools(ctx, pool, courseID, mode, ex, mergeInserted); err != nil {
		return err
	}

	// Course files + file manager (base64 bodies) after bodies so hero/markdown
	// path rewrites are already applied and ids stay stable for content URLs.
	if applyFiles {
		if err := applyCourseFiles(ctx, pool, courseID, targetCourseCode, mode, ex, blobOpts); err != nil {
			return err
		}
	}

	if applyEnrollments {
		if err := applyEnrollmentsFromExport(ctx, pool, targetCourseCode, courseID, mode, ex.Enrollments); err != nil {
			return err
		}
	}
	return nil
}

func toRepoSyllabus(sections []coursesyllabus.SyllabusSection) []course.SyllabusSection {
	if sections == nil {
		return []course.SyllabusSection{}
	}
	out := make([]course.SyllabusSection, 0, len(sections))
	for _, s := range sections {
		out = append(out, course.SyllabusSection{
			ID:       s.ID,
			Heading:  s.Heading,
			Markdown: s.Markdown,
		})
	}
	return out
}

