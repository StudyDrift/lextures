package sis

import (
	"context"

	"github.com/google/uuid"

	repoCatalog "github.com/lextures/lextures/server/internal/repos/catalog"
	repoSIS "github.com/lextures/lextures/server/internal/repos/sis"
)

// CatalogSection is one section pulled from a SIS catalog feed.
type CatalogSection struct {
	TermID         uuid.UUID
	SISCourseID    string
	SISSectionID   string
	CRN            *string
	Subject        string
	CourseNumber   string
	SectionNumber  *string
	Title          string
	Credits        *float64
	MeetingPattern *repoCatalog.MeetingPattern
	Room           *string
	Department     *string
	Prerequisites  []repoCatalog.Prerequisite
	InstructorName *string
	Status         string
}

// CatalogAdapter extends HE adapters with catalog pull (plan 14.2).
type CatalogAdapter interface {
	Adapter
	SyncCatalog(ctx context.Context, cfg ConnectionConfig, termID uuid.UUID, termName string) ([]CatalogSection, []repoSIS.SyncError, error)
}

// SyncCatalog pulls catalog sections from an HE vendor adapter.
func SyncCatalog(ctx context.Context, adapter Adapter, cfg ConnectionConfig, termID uuid.UUID, termName string) ([]CatalogSection, []repoSIS.SyncError, error) {
	if ca, ok := adapter.(CatalogAdapter); ok {
		return ca.SyncCatalog(ctx, cfg, termID, termName)
	}
	return nil, nil, nil
}

func stubCatalogSections(termID uuid.UUID, termName string) []CatalogSection {
	crn1 := "12345"
	crn2 := "12346"
	sec2 := "002"
	dept := "CS"
	room1 := "SCI 201"
	room2 := "SCI 105"
	credits3 := 3.0
	credits4 := 4.0
	instructor1 := "Dr. Alice Chen"
	instructor2 := "Prof. Bob Martinez"
	return []CatalogSection{
		{
			TermID:       termID,
			SISCourseID:  "CS-201",
			SISSectionID: "CS-201-001-SPRING",
			CRN:          &crn1,
			Subject:      "CS",
			CourseNumber: "201",
			SectionNumber: strPtr("001"),
			Title:        "Data Structures",
			Credits:      &credits3,
			MeetingPattern: &repoCatalog.MeetingPattern{
				Days: "MWF", StartTime: "10:00", EndTime: "10:50", Instructor: instructor1,
			},
			Room:           &room1,
			Department:     &dept,
			InstructorName: &instructor1,
			Prerequisites: []repoCatalog.Prerequisite{
				{Code: "CS 101", Title: "Intro to Computer Science"},
			},
			Status: repoCatalog.StatusActive,
		},
		{
			TermID:       termID,
			SISCourseID:  "CS-301",
			SISSectionID: "CS-301-002-SPRING",
			CRN:          &crn2,
			Subject:      "CS",
			CourseNumber: "301",
			SectionNumber: &sec2,
			Title:        "Algorithms",
			Credits:      &credits4,
			MeetingPattern: &repoCatalog.MeetingPattern{
				Days: "TR", StartTime: "14:00", EndTime: "15:15", Instructor: instructor2,
			},
			Room:           &room2,
			Department:     &dept,
			InstructorName: &instructor2,
			Prerequisites: []repoCatalog.Prerequisite{
				{Code: "CS 201", Title: "Data Structures"},
			},
			Status: repoCatalog.StatusActive,
		},
		{
			TermID:       termID,
			SISCourseID:  "MATH-150",
			SISSectionID: "MATH-150-001-SPRING",
			Subject:      "MATH",
			CourseNumber: "150",
			SectionNumber: strPtr("001"),
			Title:        "Calculus I",
			Credits:      &credits4,
			MeetingPattern: &repoCatalog.MeetingPattern{
				Days: "MWF", StartTime: "09:00", EndTime: "09:50",
			},
			Department: strPtr("MATH"),
			Status:     repoCatalog.StatusActive,
		},
	}
}

func strPtr(s string) *string { return &s }
