package coursechecklist

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lextures/lextures/server/internal/repos/course"
	"github.com/lextures/lextures/server/internal/repos/coursefeed"
	"github.com/lextures/lextures/server/internal/repos/coursegrading"
	"github.com/lextures/lextures/server/internal/repos/courseoutcomes"
	"github.com/lextures/lextures/server/internal/repos/coursesections"
	"github.com/lextures/lextures/server/internal/repos/coursestructure"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	"github.com/lextures/lextures/server/internal/repos/filemanager"
	"github.com/lextures/lextures/server/internal/repos/studentaccommodations"
	"github.com/lextures/lextures/server/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
)

// MaxSnapshotQueries is the hard budget for a full snapshot load (NFR / AC-9).
const MaxSnapshotQueries = 18

// queryCounter tallies logical SQL round-trips during LoadSnapshot (tests / AC-9).
type queryCounter struct {
	n atomic.Int64
}

// DataNeedsForItems returns the union of DataNeeds for the given registry items,
// always including DataNeedCourse.
func DataNeedsForItems(items []ItemDescriptor) []DataNeed {
	return unionDataNeeds(items)
}

// DataNeedsForEvaluate computes needs for an EvaluateOptions selection.
func DataNeedsForEvaluate(reg *Registry, opt EvaluateOptions) []DataNeed {
	if reg == nil {
		reg = MustDefault()
	}
	return unionDataNeeds(reg.ItemsForEvaluate(opt))
}

// LoadSnapshot loads a CourseSnapshot for courseCode using existing repos plus
// read-only batch helpers (FR-7, FR-15). Query count for a full load MUST stay ≤ 18.
func LoadSnapshot(ctx context.Context, pool *pgxpool.Pool, courseCode string, needs []DataNeed) (CourseSnapshot, error) {
	return loadSnapshot(ctx, pool, courseCode, needs, nil)
}

// LoadSnapshotCounted is LoadSnapshot with a query counter for tests/benchmarks.
func LoadSnapshotCounted(ctx context.Context, pool *pgxpool.Pool, courseCode string, needs []DataNeed, counter *int64) (CourseSnapshot, error) {
	qc := &queryCounter{}
	snap, err := loadSnapshot(ctx, pool, courseCode, needs, qc)
	if counter != nil {
		*counter = qc.n.Load()
	}
	snap.QueryCount = int(qc.n.Load())
	return snap, err
}

func loadSnapshot(ctx context.Context, pool *pgxpool.Pool, courseCode string, needs []DataNeed, qc *queryCounter) (CourseSnapshot, error) {
	if pool == nil {
		return CourseSnapshot{}, fmt.Errorf("coursechecklist: nil pool")
	}
	if strings.TrimSpace(courseCode) == "" {
		return CourseSnapshot{}, fmt.Errorf("coursechecklist: empty courseCode")
	}
	if len(needs) == 0 {
		needs = AllDataNeeds
	}
	// Always load course.
	if !hasDataNeed(needs, DataNeedCourse) {
		needs = append([]DataNeed{DataNeedCourse}, needs...)
	}

	start := time.Now()
	ctx, span := telemetry.Tracer("coursechecklist").Start(ctx, "coursechecklist.LoadSnapshot")
	defer span.End()
	span.SetAttributes(attribute.String("course_code", courseCode))

	count := func(n int) {
		if qc != nil {
			qc.n.Add(int64(n))
		}
	}

	pub, err := course.GetPublicByCourseCode(ctx, pool, courseCode)
	count(1)
	if err != nil {
		return CourseSnapshot{}, err
	}
	if pub == nil {
		return CourseSnapshot{}, fmt.Errorf("coursechecklist: course %q not found", courseCode)
	}
	courseID, err := uuid.Parse(pub.ID)
	if err != nil {
		return CourseSnapshot{}, fmt.Errorf("coursechecklist: parse course id: %w", err)
	}

	var homeContentID *string
	if pub.CourseHomeContentItemID != nil {
		homeContentID = pub.CourseHomeContentItemID
	}

	snap := CourseSnapshot{
		CourseCode:              courseCode,
		CourseID:                courseID,
		Title:                   pub.Title,
		Description:             pub.Description,
		Published:               pub.Published,
		StartsAt:                pub.StartsAt,
		EndsAt:                  pub.EndsAt,
		VisibleFrom:             pub.VisibleFrom,
		HiddenAt:                pub.HiddenAt,
		CourseTimezone:          pub.CourseTimezone,
		ScheduleMode:            pub.ScheduleMode,
		SectionsEnabled:         pub.SectionsEnabled,
		FeedEnabled:             pub.FeedEnabled,
		FilesEnabled:            pub.FilesEnabled,
		SbgEnabled:              pub.SbgEnabled,
		StandardsEnabled:        pub.StandardsAlignmentEnabled,
		CourseType:              pub.CourseType,
		CourseMode:              pub.CourseMode,
		HeroImageURL:            pub.HeroImageURL,
		CourseHomeLanding:       pub.CourseHomeLanding,
		CourseHomeContentItemID: homeContentID,
		CreatedAt:               pub.CreatedAt,
		HomeschoolMode:          pub.OrgID == nil,
		OrgIsK12:                len(pub.GradeLevels) > 0,
		// ParentPortalEnabled left false by default (guardian rule N/A unless tests set it).
		GradingScale: pub.GradingScale,
		Features: CourseFeatures{
			NotebookEnabled:           pub.NotebookEnabled,
			FeedEnabled:               pub.FeedEnabled,
			CalendarEnabled:           pub.CalendarEnabled,
			DiscussionsEnabled:        pub.DiscussionsEnabled,
			FilesEnabled:              pub.FilesEnabled,
			AttendanceEnabled:         pub.AttendanceEnabled,
			StandardsAlignmentEnabled: pub.StandardsAlignmentEnabled,
			AdaptivePathsEnabled:      pub.AdaptivePathsEnabled,
			ContentToolsEnabled:       pub.ContentToolsEnabled,
			InteractiveQuizzesEnabled: pub.InteractiveQuizzesEnabled,
			RequireCaptions:           pub.RequireCaptions,
			GroupSpacesEnabled:        pub.GroupSpacesEnabled,
			VisualBoardsEnabled:       pub.VisualBoardsEnabled,
			AiTutorEnabled:            pub.AiTutorEnabled,
			ModulesAiAssistantEnabled: pub.ModulesAiAssistantEnabled,
		},
		Lazy: make(map[LazyLoaderID]any),
	}

	// Extra course markers not on CoursePublic (features_reviewed_at, grading_scheme_id,
	// catalog_language, created_by_user_id) — one query (CC.3).
	{
		var featuresReviewedAt *time.Time
		var gradingSchemeID *uuid.UUID
		var catalogLanguage *string
		var creatorID *uuid.UUID
		err := pool.QueryRow(ctx, `
SELECT features_reviewed_at, grading_scheme_id, NULLIF(TRIM(catalog_language), ''), created_by_user_id
FROM course.courses
WHERE id = $1
`, courseID).Scan(&featuresReviewedAt, &gradingSchemeID, &catalogLanguage, &creatorID)
		count(1)
		if err != nil {
			return CourseSnapshot{}, err
		}
		snap.FeaturesReviewedAt = featuresReviewedAt
		snap.GradingSchemeID = gradingSchemeID
		snap.CreatorUserID = creatorID
		if catalogLanguage != nil {
			snap.CatalogLanguage = *catalogLanguage
		}
	}

	if hasDataNeed(needs, DataNeedStructure) {
		items, err := coursestructure.ListForCourse(ctx, pool, courseID)
		count(1)
		if err != nil {
			return CourseSnapshot{}, err
		}
		snap.StructureItems = make([]StructureItem, 0, len(items))
		for _, it := range items {
			snap.StructureItems = append(snap.StructureItems, StructureItem{
				ID:                it.ID,
				Kind:              it.Kind,
				Title:             it.Title,
				ParentID:          it.ParentID,
				Published:         it.Published,
				DueAt:             it.DueAt,
				AssignmentGroupID: it.AssignmentGroupID,
				Archived:          it.Archived,
				SortOrder:         it.SortOrder,
			})
		}
	}

	if hasDataNeed(needs, DataNeedItemMeta) {
		meta, err := coursestructure.ListChecklistItemMeta(ctx, pool, courseID)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) {
				return CourseSnapshot{}, err
			}
			meta = map[uuid.UUID]coursestructure.ChecklistItemMeta{}
		}
		snap.ItemMeta = make(map[uuid.UUID]ItemMeta, len(meta))
		for id, m := range meta {
			snap.ItemMeta[id] = ItemMeta{
				Kind:                 m.Kind,
				HasBody:              m.HasBody,
				PointsWorth:          m.PointsWorth,
				ExternalURL:          m.ExternalURL,
				QuestionCount:        m.QuestionCount,
				LateSubmissionPolicy: m.LateSubmissionPolicy,
			}
		}
	}

	if hasDataNeed(needs, DataNeedSyllabus) {
		syl, err := course.GetSyllabusByCourseCode(ctx, pool, courseCode)
		count(1)
		if err != nil {
			return CourseSnapshot{}, err
		}
		if syl != nil {
			snap.SyllabusMalformed = syl.Malformed
			snap.AcceptanceDecidedAt = syl.AcceptanceDecidedAt
			snap.RequireSyllabusAcceptance = syl.RequireSyllabusAcceptance
			totalBytes := 0
			for _, s := range syl.Sections {
				md := s.Markdown
				chunk := len(s.Heading) + len(md) + 2
				if totalBytes+chunk > MaxSyllabusScanBytes {
					remain := MaxSyllabusScanBytes - totalBytes
					if remain < 0 {
						remain = 0
					}
					if len(md) > remain {
						md = md[:remain]
					}
					snap.SyllabusCheckedTruncated = true
				}
				totalBytes += len(s.Heading) + len(md) + 2
				snap.SyllabusSections = append(snap.SyllabusSections, SyllabusSectionSnap{
					Key:      s.ID,
					Title:    s.Heading,
					HasBody:  strings.TrimSpace(s.Markdown) != "",
					Markdown: md,
				})
				if snap.SyllabusCheckedTruncated {
					break
				}
			}
		}
	}

	if hasDataNeed(needs, DataNeedOutcomes) {
		outcomes, err := courseoutcomes.ListOutcomes(ctx, pool, courseID)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) {
				return CourseSnapshot{}, err
			}
		} else {
			for _, o := range outcomes {
				snap.Outcomes = append(snap.Outcomes, OutcomeSnap{ID: o.ID, Title: o.Title})
			}
		}
		links, err := courseoutcomes.ListLinksForCourse(ctx, pool, courseID)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) {
				return CourseSnapshot{}, err
			}
		} else {
			for _, l := range links {
				snap.OutcomeLinks = append(snap.OutcomeLinks, OutcomeLinkSnap{
					OutcomeID: l.OutcomeID,
					ItemID:    l.StructureItemID,
				})
			}
		}
	}

	if hasDataNeed(needs, DataNeedGrading) {
		settings, err := coursegrading.GetSettingsForCourseCode(ctx, pool, courseCode)
		count(2) // settings query + groups query inside helper
		if err != nil {
			return CourseSnapshot{}, err
		}
		if settings != nil {
			snap.GradingScale = settings.GradingScale
			for _, g := range settings.AssignmentGroups {
				w := g.WeightPercent
				snap.AssignmentGroups = append(snap.AssignmentGroups, AssignmentGroupSnap{
					ID:     g.ID,
					Name:   g.Name,
					Weight: &w,
				})
			}
		}
	}

	if hasDataNeed(needs, DataNeedEnrollments) {
		counts, err := enrollment.CountByRoleForCourse(ctx, pool, courseID)
		count(1)
		if err != nil {
			return CourseSnapshot{}, err
		}
		snap.EnrollmentCounts = counts
		people, err := enrollment.ListChecklistPeopleForCourse(ctx, pool, courseID)
		count(1)
		if err != nil {
			return CourseSnapshot{}, err
		}
		now := time.Now().UTC()
		for _, p := range people {
			enrolledAt := p.CreatedAt
			snap.People = append(snap.People, PersonSnap{
				UserID:            p.UserID,
				DisplayName:       p.DisplayName,
				Role:              p.Role,
				InvitationPending: p.InvitationPending,
				EnrolledAt:        &enrolledAt,
				SectionID:         p.SectionID,
				Active:            p.Active,
			})
			if p.InvitationPending {
				days := int(now.Sub(p.CreatedAt).Hours() / 24)
				if days < 0 {
					days = 0
				}
				snap.PendingInvitations = append(snap.PendingInvitations, PendingInviteSnap{
					DisplayName: p.DisplayName,
					UserID:      p.UserID,
					CreatedAt:   p.CreatedAt,
					DaysPending: days,
				})
			}
		}
	}

	if hasDataNeed(needs, DataNeedFeed) {
		channels, err := coursefeed.ListChecklistFeedChannels(ctx, pool, courseID)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) {
				return CourseSnapshot{}, err
			}
		} else {
			for _, ch := range channels {
				snap.FeedChannels = append(snap.FeedChannels, FeedChannelSnap{
					ID:          ch.ID,
					Name:        ch.Name,
					LatestAt:    ch.LatestAt,
					LatestTitle: ch.LatestTitle,
				})
				if strings.EqualFold(ch.Name, "announcements") && ch.StaffWelcome != nil {
					snap.AnnouncementsWelcome = &WelcomeMessageSnap{
						BodyLen:       ch.StaffWelcome.BodyLen,
						AuthorIsStaff: ch.StaffWelcome.AuthorIsStaff,
						PostedAt:      ch.StaffWelcome.PostedAt,
					}
				}
			}
		}
	}

	if hasDataNeed(needs, DataNeedFiles) {
		files, err := filemanager.ListFileMetaForCourse(ctx, pool, courseID)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) {
				return CourseSnapshot{}, err
			}
		} else {
			for _, f := range files {
				snap.Files = append(snap.Files, FileSnap{
					ID:          f.ID,
					DisplayName: f.DisplayName,
					ContentType: f.ContentType,
					ByteSize:    f.ByteSize,
				})
			}
		}
	}

	if hasDataNeed(needs, DataNeedSections) {
		sections, err := coursesections.ListForCourse(ctx, pool, courseID)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) {
				return CourseSnapshot{}, err
			}
		} else {
			for _, s := range sections {
				name := ""
				if s.Name != nil {
					name = *s.Name
				}
				snap.Sections = append(snap.Sections, SectionSnap{
					ID:          s.ID,
					SectionCode: s.SectionCode,
					Name:        name,
					Status:      s.Status,
				})
			}
		}
	}

	if hasDataNeed(needs, DataNeedAccommodations) {
		n, err := studentaccommodations.CountActiveForCourse(ctx, pool, courseID)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) {
				return CourseSnapshot{}, err
			}
		} else {
			snap.AccommodationCount = n
		}
	}

	if hasDataNeed(needs, DataNeedStandards) {
		var n int
		err := pool.QueryRow(ctx, `
SELECT COUNT(*)::int FROM course.course_standards WHERE course_id = $1
`, courseID).Scan(&n)
		count(1)
		if err != nil {
			if !isUndefinedTable(err) {
				return CourseSnapshot{}, err
			}
		} else {
			snap.StandardsCount = n
		}
	}

	if qc != nil {
		snap.QueryCount = int(qc.n.Load())
	}
	observeSnapshotQueryDuration(time.Since(start).Seconds())
	span.SetAttributes(attribute.Int("query_count", snap.QueryCount))
	return snap, nil
}

func isUndefinedTable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "does not exist") || strings.Contains(msg, "undefined_table")
}
