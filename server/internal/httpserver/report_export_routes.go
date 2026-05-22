package httpserver

import "github.com/go-chi/chi/v5"

func (d Deps) registerReportExportRoutes(r chi.Router) {
	// Course-scoped PDF export: GET /api/v1/courses/{course_code}/reports/{report_type}/export.pdf
	r.Get("/api/v1/courses/{course_code}/reports/{report_type}/export.pdf", d.handleExportCoursePDF())

	// Platform learning activity PDF export
	r.Get("/api/v1/reports/learning-activity/export.pdf", d.handleExportLearningActivityPDF())

	// Report schedule management
	r.Get("/api/v1/reports/schedules", d.handleListReportSchedules())
	r.Post("/api/v1/reports/schedules", d.handleCreateReportSchedule())
	r.Put("/api/v1/reports/schedules/{id}", d.handleUpdateReportSchedule())
	r.Delete("/api/v1/reports/schedules/{id}", d.handleDeleteReportSchedule())
}
