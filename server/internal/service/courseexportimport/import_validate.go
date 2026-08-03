package courseexportimport

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	modelcourse "github.com/lextures/lextures/server/internal/models/course"
	"github.com/lextures/lextures/server/internal/models/courseexport"
	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursemoduleexternallinks"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
)

func validateExportPayload(ex *Bundle) error {
	if ex.FormatVersion != exportFormatVersion {
		return InvalidInput("Unsupported export formatVersion (expected 1).")
	}
	if strings.TrimSpace(ex.CourseCode) == "" {
		return InvalidInput("Export is missing courseCode.")
	}
	scaleOK := false
	for _, s := range modelcourse.GradingScales {
		if s == strings.TrimSpace(ex.Grading.GradingScale) {
			scaleOK = true
			break
		}
	}
	if !scaleOK {
		return InvalidInput("Invalid grading scale in export.")
	}
	for _, g := range ex.Grading.AssignmentGroups {
		if strings.TrimSpace(g.Name) == "" {
			return InvalidInput("Each assignment group in the export needs a name.")
		}
	}
	if err := validateSyllabusSections(toRepoSyllabus(ex.Syllabus)); err != nil {
		return err
	}
	if err := validateStructureExport(ex.Structure); err != nil {
		return err
	}
	for id, body := range ex.ContentPages {
		if len(body.Markdown) > maxModuleContentMarkdownLen {
			return InvalidInput(fmt.Sprintf("Content page %s markdown is too long.", id))
		}
	}
	for id, body := range ex.Assignments {
		if len(body.Markdown) > maxModuleContentMarkdownLen {
			return InvalidInput(fmt.Sprintf("Assignment %s markdown is too long.", id))
		}
	}
	for id, body := range ex.Quizzes {
		if len(body.Markdown) > maxModuleContentMarkdownLen {
			return InvalidInput(fmt.Sprintf("Quiz %s markdown is too long.", id))
		}
	}
	for _, it := range ex.Structure {
		if it.Kind != "external_link" {
			continue
		}
		if it.ExternalURL == nil {
			continue
		}
		t := strings.TrimSpace(*it.ExternalURL)
		if t == "" {
			continue
		}
		if _, err := coursemoduleexternallinks.ValidateExternalHTTPURL(t); err != nil {
			return InvalidInput(err.Error())
		}
	}
	if err := validateContentToolsExport(ex); err != nil {
		return err
	}
	if err := validateCourseFilesExport(ex); err != nil {
		return err
	}
	return validateExportEnrollments(ex.Enrollments)
}

func validateCourseFilesExport(ex *Bundle) error {
	if len(ex.CourseFiles) > maxExportCourseFiles {
		return InvalidInput(fmt.Sprintf("Too many course files in export (max %d).", maxExportCourseFiles))
	}
	if len(ex.FileFolders) > maxExportFileFolders {
		return InvalidInput(fmt.Sprintf("Too many file folders in export (max %d).", maxExportFileFolders))
	}
	if len(ex.FileItems) > maxExportFileItems {
		return InvalidInput(fmt.Sprintf("Too many file items in export (max %d).", maxExportFileItems))
	}
	seenFiles := map[uuid.UUID]struct{}{}
	var totalApprox int64
	for _, f := range ex.CourseFiles {
		if f.ID == uuid.Nil {
			return InvalidInput("Each course file needs a valid id.")
		}
		if _, ok := seenFiles[f.ID]; ok {
			return InvalidInput("Duplicate course file id.")
		}
		seenFiles[f.ID] = struct{}{}
		if strings.TrimSpace(f.OriginalFilename) == "" {
			return InvalidInput(fmt.Sprintf("Course file %s needs originalFilename.", f.ID))
		}
		if f.ByteSize < 0 {
			return InvalidInput(fmt.Sprintf("Course file %s has invalid byteSize.", f.ID))
		}
		if b64 := strings.TrimSpace(f.ContentBase64); b64 != "" {
			approx := (int64(len(b64)) * 3) / 4
			if approx > maxExportSingleFileBytes+1024 {
				return InvalidInput(fmt.Sprintf("Course file %s content is too large.", f.ID))
			}
			totalApprox += approx
		}
	}
	seenFolders := map[uuid.UUID]struct{}{}
	for _, f := range ex.FileFolders {
		if f.ID == uuid.Nil {
			return InvalidInput("Each file folder needs a valid id.")
		}
		if _, ok := seenFolders[f.ID]; ok {
			return InvalidInput("Duplicate file folder id.")
		}
		seenFolders[f.ID] = struct{}{}
		if strings.TrimSpace(f.Name) == "" {
			return InvalidInput(fmt.Sprintf("File folder %s needs a name.", f.ID))
		}
		if len(f.Name) > 255 {
			return InvalidInput(fmt.Sprintf("File folder %s name is too long.", f.ID))
		}
	}
	seenItems := map[uuid.UUID]struct{}{}
	for _, it := range ex.FileItems {
		if it.ID == uuid.Nil {
			return InvalidInput("Each file item needs a valid id.")
		}
		if _, ok := seenItems[it.ID]; ok {
			return InvalidInput("Duplicate file item id.")
		}
		seenItems[it.ID] = struct{}{}
		if strings.TrimSpace(it.OriginalFilename) == "" && strings.TrimSpace(it.DisplayName) == "" {
			return InvalidInput(fmt.Sprintf("File item %s needs a name.", it.ID))
		}
		if it.ByteSize < 0 {
			return InvalidInput(fmt.Sprintf("File item %s has invalid byteSize.", it.ID))
		}
		// Unknown folder parents (not listed in this export) are nulled on import;
		// they may already exist on the target course.
		if b64 := strings.TrimSpace(it.ContentBase64); b64 != "" {
			approx := (int64(len(b64)) * 3) / 4
			if approx > maxExportSingleFileBytes+1024 {
				return InvalidInput(fmt.Sprintf("File item %s content is too large.", it.ID))
			}
			totalApprox += approx
		}
	}
	if totalApprox > maxExportTotalFileBytes {
		return InvalidInput(fmt.Sprintf("Total file content in export exceeds size limit (%d bytes).", maxExportTotalFileBytes))
	}
	return nil
}

func validateContentToolsExport(ex *Bundle) error {
	if ex.ContentToolSettings != nil {
		s := ex.ContentToolSettings
		if s.MaxInstancesPerItem != 0 && (s.MaxInstancesPerItem < 1 || s.MaxInstancesPerItem > 200) {
			return InvalidInput("contentToolSettings.maxInstancesPerItem must be between 1 and 200.")
		}
		mode := strings.TrimSpace(s.LinkIngestionMode)
		if mode != "" && mode != "public" && mode != "allowlist" && mode != "disabled" {
			return InvalidInput("contentToolSettings.linkIngestionMode is invalid.")
		}
	}
	if len(ex.ContentToolInstances) > maxContentToolInstances {
		return InvalidInput(fmt.Sprintf("Too many content tool instances in export (max %d).", maxContentToolInstances))
	}
	seen := map[uuid.UUID]struct{}{}
	allowedHost := map[string]struct{}{
		"content_page": {}, "assignment": {}, "quiz": {}, "syllabus": {}, "portfolio_artifact": {},
	}
	for _, in := range ex.ContentToolInstances {
		if in.ID == uuid.Nil {
			return InvalidInput("Each content tool instance needs a valid id.")
		}
		if _, ok := seen[in.ID]; ok {
			return InvalidInput("Duplicate content tool instance id.")
		}
		seen[in.ID] = struct{}{}
		if strings.TrimSpace(in.ToolID) == "" {
			return InvalidInput("Each content tool instance needs a toolId.")
		}
		if strings.TrimSpace(in.ToolVersion) == "" {
			return InvalidInput("Each content tool instance needs a toolVersion.")
		}
		host := strings.TrimSpace(in.HostKind)
		if _, ok := allowedHost[host]; !ok {
			return InvalidInput(fmt.Sprintf("Unsupported content tool hostKind: %s.", host))
		}
		if host == "syllabus" {
			if in.StructureItemID != nil {
				return InvalidInput("Syllabus content tool instances must not have structureItemId.")
			}
		} else if in.StructureItemID == nil {
			return InvalidInput("Non-syllabus content tool instances need structureItemId.")
		}
		status := strings.TrimSpace(in.Status)
		if status != "" && status != "active" && status != "archived" {
			return InvalidInput("content tool instance status must be active or archived.")
		}
		if len(in.ConfigJSON) > maxContentToolConfigBytes {
			return InvalidInput(fmt.Sprintf("Content tool instance %s config is too large.", in.ID))
		}
		if len(in.ConfigJSON) > 0 && !json.Valid(in.ConfigJSON) {
			return InvalidInput(fmt.Sprintf("Content tool instance %s configJson is not valid JSON.", in.ID))
		}
	}
	return nil
}

func validateSyllabusSections(sections []course.SyllabusSection) error {
	if len(sections) > maxSyllabusSections {
		return InvalidInput(fmt.Sprintf("Too many sections (max %d).", maxSyllabusSections))
	}
	for _, s := range sections {
		if strings.TrimSpace(s.ID) == "" {
			return InvalidInput("Each section needs an id.")
		}
		if len(s.Heading) > maxSyllabusHeadingLen {
			return InvalidInput("Section heading is too long.")
		}
		if len(s.Markdown) > maxSyllabusMarkdownLen {
			return InvalidInput("Section content is too long.")
		}
	}
	return nil
}

func validateStructureExport(items []coursestructure.ItemResponse) error {
	allowed := map[string]struct{}{
		"module": {}, "heading": {}, "content_page": {}, "assignment": {}, "quiz": {}, "external_link": {},
	}
	seen := map[uuid.UUID]struct{}{}
	for _, it := range items {
		if _, ok := allowed[it.Kind]; !ok {
			return InvalidInput(fmt.Sprintf("Unsupported structure kind: %s.", it.Kind))
		}
		id, err := uuid.Parse(it.ID)
		if err != nil {
			return InvalidInput("Invalid structure item id.")
		}
		if it.ParentID != nil && *it.ParentID != "" {
			pid, err := uuid.Parse(*it.ParentID)
			if err != nil {
				return InvalidInput("Invalid structure parent id.")
			}
			if _, ok := seen[pid]; !ok {
				return InvalidInput("Structure items must be ordered so each parent appears before its children.")
			}
		} else if it.Kind != "module" {
			return InvalidInput("Only modules may have a null parent.")
		}
		if _, ok := seen[id]; ok {
			return InvalidInput("Duplicate structure item id.")
		}
		seen[id] = struct{}{}
	}
	return nil
}

func validateExportEnrollments(rows []courseexport.ExportedCourseEnrollment) error {
	if len(rows) > maxExportEnrollments {
		return InvalidInput(fmt.Sprintf("Too many enrollments in export (max %d).", maxExportEnrollments))
	}
	for _, row := range rows {
		e := strings.ToLower(strings.TrimSpace(row.Email))
		if e == "" || !strings.Contains(e, "@") || len(e) > maxEnrollmentEmailLen {
			return InvalidInput("Each enrollment needs a valid email address.")
		}
		role := strings.TrimSpace(row.Role)
		if role != "student" && role != "instructor" && role != "teacher" && role != "ta" &&
			role != "designer" && role != "observer" && role != "auditor" && role != "librarian" && role != "owner" {
			return InvalidInput(fmt.Sprintf("Invalid enrollment role `%s`.", role))
		}
		if row.InstructorGrantRole != nil {
			g := strings.TrimSpace(*row.InstructorGrantRole)
			if g != "" && g != "Teacher" && g != "TA" {
				return InvalidInput("instructorGrantRole must be Teacher or TA when set.")
			}
			if g != "" && role != "instructor" {
				return InvalidInput("instructorGrantRole may only be set when role is instructor.")
			}
		}
		if row.DisplayName != nil && len(*row.DisplayName) > maxEnrollmentDisplayNameLen {
			return InvalidInput(fmt.Sprintf("Enrollment display name is too long (max %d).", maxEnrollmentDisplayNameLen))
		}
	}
	return nil
}
