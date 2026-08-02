package courseexportimport

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/repos/contenttools"
)

// applyContentTools restores CT settings and instances after structure/bodies are applied.
// Instance IDs are preserved so ```lex-tool fences in imported markdown resolve.
// Learner state is never imported.
// onlyStructureIDs, when non-nil (mergeAdd), limits structure-hosted instances to newly inserted items.
func applyContentTools(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	mode courseexport.CourseImportMode,
	ex *Bundle,
	onlyStructureIDs map[uuid.UUID]struct{},
) error {
	if ex.ContentToolSettings == nil && len(ex.ContentToolInstances) == 0 {
		return nil
	}

	if ex.ContentToolSettings != nil {
		s := contenttools.SettingsRow{
			CourseID:             courseID,
			AllowedToolIDs:       ex.ContentToolSettings.AllowedToolIDs,
			StudentResetAllowed:  ex.ContentToolSettings.StudentResetAllowed,
			MaxInstancesPerItem:  ex.ContentToolSettings.MaxInstancesPerItem,
			MonthlyAITokenBudget: ex.ContentToolSettings.MonthlyAITokenBudget,
			DailyAICallsPerUser:  ex.ContentToolSettings.DailyAICallsPerUser,
			LinkIngestionMode:    ex.ContentToolSettings.LinkIngestionMode,
			LinkHostAllowlist:    ex.ContentToolSettings.LinkHostAllowlist,
			GradeLinksAllowed:    ex.ContentToolSettings.GradeLinksAllowed,
		}
		if _, err := contenttools.UpsertSettings(ctx, pool, courseID, s, uuid.Nil); err != nil {
			return err
		}
	}

	if len(ex.ContentToolInstances) == 0 {
		if mode == courseexport.CourseImportModeErase || mode == courseexport.CourseImportModeOverwrite {
			// Empty export list with erase/overwrite: clear leftover syllabus instances
			// (structure-hosted ones already cascade when outline items are deleted).
			return contenttools.DeleteAllInstancesForCourse(ctx, pool, courseID)
		}
		return nil
	}

	switch mode {
	case courseexport.CourseImportModeErase:
		if err := contenttools.DeleteAllInstancesForCourse(ctx, pool, courseID); err != nil {
			return err
		}
		for i := range ex.ContentToolInstances {
			if err := upsertOneContentToolInstance(ctx, pool, courseID, &ex.ContentToolInstances[i], false); err != nil {
				return err
			}
		}
	case courseexport.CourseImportModeOverwrite:
		keep := make([]uuid.UUID, 0, len(ex.ContentToolInstances))
		for i := range ex.ContentToolInstances {
			in := &ex.ContentToolInstances[i]
			if err := upsertOneContentToolInstance(ctx, pool, courseID, in, false); err != nil {
				return err
			}
			keep = append(keep, in.ID)
		}
		if err := contenttools.DeleteInstancesNotInExport(ctx, pool, courseID, keep); err != nil {
			return err
		}
	case courseexport.CourseImportModeMergeAdd:
		for i := range ex.ContentToolInstances {
			in := &ex.ContentToolInstances[i]
			if in.StructureItemID != nil {
				if onlyStructureIDs != nil {
					if _, ok := onlyStructureIDs[*in.StructureItemID]; !ok {
						continue
					}
				}
			}
			if err := upsertOneContentToolInstance(ctx, pool, courseID, in, true); err != nil {
				return err
			}
		}
	}
	return nil
}

func upsertOneContentToolInstance(
	ctx context.Context,
	pool *pgxpool.Pool,
	courseID uuid.UUID,
	in *courseexport.ExportedContentToolInstance,
	onlyInsert bool,
) error {
	// Skip instances whose host structure item is missing (e.g. partial merge).
	if in.StructureItemID != nil {
		ok, err := contenttools.StructureItemInCourse(ctx, pool, courseID, *in.StructureItemID)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	} else if strings.TrimSpace(in.HostKind) != "syllabus" {
		return nil
	}

	row := contenttools.InstanceRow{
		ID:                  in.ID,
		CourseID:            courseID,
		StructureItemID:     in.StructureItemID,
		HostKind:            strings.TrimSpace(in.HostKind),
		SectionKey:          in.SectionKey,
		ToolID:              strings.TrimSpace(in.ToolID),
		ToolVersion:         strings.TrimSpace(in.ToolVersion),
		Title:               in.Title,
		ConfigJSON:          in.ConfigJSON,
		ConfigSchemaVersion: in.ConfigSchemaVersion,
		Status:              strings.TrimSpace(in.Status),
	}
	if onlyInsert {
		_, err := contenttools.InsertInstanceIfMissing(ctx, pool, row)
		return err
	}
	return contenttools.UpsertInstanceForImport(ctx, pool, row)
}
