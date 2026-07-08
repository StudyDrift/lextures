package httpserver

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/lextures/lextures/server/internal/models/coursemodulequiz"
	icrepo "github.com/lextures/lextures/server/internal/repos/introcourse"
	ctrepo "github.com/lextures/lextures/server/internal/repos/coursetranslation"
	"github.com/lextures/lextures/server/internal/repos/enrollment"
	introcourseservice "github.com/lextures/lextures/server/internal/service/introcourse"
)

func (d Deps) resolveIntroCourseLocale(ctx context.Context, courseID, viewer uuid.UUID, r *http.Request) string {
	if d.Pool != nil && courseID != uuid.Nil && viewer != uuid.Nil {
		if loc, err := ctrepo.GetEnrollmentContentLocale(ctx, d.Pool, courseID, viewer); err == nil && loc != nil {
			if tag, err := normalizeLocaleInput(*loc); err == nil {
				return tag
			}
		}
		var userLocale string
		if err := d.Pool.QueryRow(ctx, `SELECT COALESCE(NULLIF(locale, ''), 'en') FROM "user".users WHERE id = $1`, viewer).Scan(&userLocale); err == nil {
			if tag, err := normalizeLocaleInput(userLocale); err == nil {
				return tag
			}
		}
	}
	if r != nil {
		return detectBrowserLocale(r.Header.Get("Accept-Language"))
	}
	return "en"
}

func (d Deps) enrichIntroCourseContentPage(
	r *http.Request,
	courseCode string,
	courseID, itemID, viewer uuid.UUID,
	canEdit bool,
	title, markdown *string,
) {
	if canEdit || courseCode != introcourseservice.CourseCode || d.Pool == nil || title == nil || markdown == nil {
		return
	}
	slug, err := icrepo.SlugByStructureItemID(r.Context(), d.Pool, itemID)
	if err != nil || slug == "" {
		return
	}
	locale := d.resolveIntroCourseLocale(r.Context(), courseID, viewer, r)
	*title, *markdown = introcourseservice.LocalizePage(slug, *title, *markdown, locale)
}

func (d Deps) localizeIntroCourseSyllabus(
	r *http.Request,
	courseCode string,
	courseID, viewer uuid.UUID,
	out syllabusResponse,
) syllabusResponse {
	if courseCode != introcourseservice.CourseCode || d.Pool == nil || len(out.Sections) == 0 {
		return out
	}
	isStaff, err := enrollment.UserIsCourseStaff(r.Context(), d.Pool, courseCode, viewer)
	if err != nil || isStaff {
		return out
	}
	locale := d.resolveIntroCourseLocale(r.Context(), courseID, viewer, r)
	out.Sections = introcourseservice.LocalizeSyllabus(out.Sections, locale)
	return out
}

func (d Deps) enrichIntroCourseQuiz(
	r *http.Request,
	courseCode string,
	courseID, itemID, viewer uuid.UUID,
	canEdit bool,
	title, markdown *string,
	questions *[]coursemodulequiz.QuizQuestion,
) {
	if canEdit || courseCode != introcourseservice.CourseCode || d.Pool == nil {
		return
	}
	slug, err := icrepo.SlugByStructureItemID(r.Context(), d.Pool, itemID)
	if err != nil || slug == "" || title == nil || markdown == nil || questions == nil {
		return
	}
	locale := d.resolveIntroCourseLocale(r.Context(), courseID, viewer, r)
	*title, *markdown, *questions = introcourseservice.LocalizeQuiz(slug, *title, *markdown, *questions, locale)
}