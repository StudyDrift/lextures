import { Suspense, useEffect } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { RequireAuth } from './auth/require-auth'
import { ApiErrorBoundary } from './components/api-error-boundary'
import { AppShell } from './components/layout/app-shell'
import { RouteFallback } from './components/route-fallback'
import * as Pages from './lazy-pages'
import { applyDocumentScrollMode, isStandalonePublicRoute } from './lib/standalone-public-routes'

export default function App() {
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    applyDocumentScrollMode(location.pathname)
  }, [location.pathname])

  useEffect(() => {
    function onAuthRequired() {
      const from = `${location.pathname}${location.search}${location.hash}`
      if (isStandalonePublicRoute(location.pathname)) {
        return
      }
      navigate('/login', { replace: true, state: { from } })
    }
    window.addEventListener('studydrift-auth-required', onAuthRequired)
    return () => {
      window.removeEventListener('studydrift-auth-required', onAuthRequired)
    }
  }, [location.hash, location.pathname, location.search, navigate])

  return (
    <Suspense fallback={<RouteFallback />}>
      <Routes>
        <Route path="/login" element={<Pages.Login />} />
        <Route path="/login/magic-link" element={<Pages.MagicLinkPage />} />
        <Route path="/login/mfa" element={<Pages.MfaLogin />} />
        <Route path="/saml-callback" element={<Pages.SamlCallback />} />
        <Route path="/sso-error" element={<Pages.SsoError />} />
        <Route path="/signup" element={<Pages.Signup />} />
        <Route path="/forgot-password" element={<Pages.ForgotPassword />} />
        <Route path="/reset-password" element={<Pages.ResetPassword />} />
        <Route path="/ai-disclosure" element={<Pages.AiDisclosurePage />} />
        <Route path="/trust" element={<Pages.TrustCenterPage />} />
        <Route path="/p/:slug/content/:aid" element={<Pages.PublicPortfolioContentPage />} />
        <Route path="/p/:slug" element={<Pages.PublicPortfolioPage />} />
        <Route path="/verify/:token" element={<Pages.CcrVerifyPage />} />
        <Route element={<RequireAuth />}>
          <Route path="/cli-auth" element={<Pages.CliAuthPage />} />
          <Route
            element={
              <ApiErrorBoundary>
                <AppShell />
              </ApiErrorBoundary>
            }
          >
            <Route path="/" element={<Pages.Dashboard />} />
            <Route path="/privacy-centre" element={<Pages.PrivacyCentrePage />} />
            <Route path="/me/study-insights" element={<Pages.StudyInsightsPage />} />
            <Route path="/me/ccr" element={<Pages.MyCCR />} />
            <Route path="/me/ce-transcript" element={<Pages.CeTranscript />} />
            <Route path="/transcripts" element={<Pages.TranscriptsPage />} />
            <Route path="/advising-notes" element={<Pages.AdvisingNotesPage />} />
            <Route path="/me/research-studies" element={<Pages.ResearchStudiesPage />} />
            <Route path="/me/accommodations" element={<Pages.MyAccommodationsPage />} />
            <Route path="/parent" element={<Pages.ParentDashboard />} />
            <Route path="/parent/conferences" element={<Pages.ConferenceBooking />} />
            <Route path="/conferences/availability" element={<Pages.ConferenceAvailabilitySetup />} />
            <Route path="/ai" element={<Pages.AskAiPage />} />
            <Route path="/review" element={<Pages.ReviewSessionPage />} />
            <Route path="/courses" element={<Pages.Courses />} />
            <Route path="/notebooks/global" element={<Pages.GlobalNotebookPage />} />
            <Route path="/notebooks" element={<Pages.MyNotebooksPage />} />
            <Route path="/courses/create" element={<Pages.CourseCreate />} />
            <Route path="/courses/:courseCode" element={<Pages.CourseLayout />}>
              <Route path="settings/*" element={<Pages.CourseSettings />} />
              <Route path="feed" element={<Pages.CourseFeedPage />} />
              <Route path="discussions" element={<Pages.CourseDiscussionsPage />} />
              <Route path="collab-docs/:docId?" element={<Pages.CourseCollabDocsPage />} />
              <Route path="files" element={<Pages.CourseFilesPage />} />
              <Route path="groups" element={<Pages.CourseGroupsPage />} />
              <Route path="syllabus" element={<Pages.CourseSyllabus />} />
              <Route path="modules/content/:itemId" element={<Pages.CourseModuleContentPage />} />
              <Route path="modules/assignment/:itemId" element={<Pages.CourseModuleAssignmentPage />} />
              <Route
                path="modules/assignment/:itemId/moderation"
                element={<Pages.ModerationDashboard />}
              />
              <Route path="modules/quiz/:itemId/attempt" element={<Pages.CourseModuleQuizAttemptPage />} />
              <Route path="modules/quiz/:itemId" element={<Pages.CourseModuleQuizPage />} />
              <Route path="diagnostic" element={<Pages.CourseDiagnosticPage />} />
              <Route path="modules/external-link/:itemId" element={<Pages.CourseModuleExternalLinkPage />} />
              <Route path="modules/h5p/:itemId" element={<Pages.CourseModuleH5PPage />} />
              <Route path="modules/lti/:itemId" element={<Pages.CourseModuleLtiPage />} />
              <Route path="modules/vibe-activity/:itemId" element={<Pages.CourseModuleVibeActivityPage />} />
              <Route path="modules/textbook-resource/:itemId" element={<Pages.CourseModuleTextbookResourcePage />} />
              <Route path="questions" element={<Pages.CourseQuestionBankPage />} />
              <Route path="misconception-report" element={<Pages.CourseMisconceptionReportPage />} />
              <Route path="modules" element={<Pages.CourseModules />} />
              <Route path="live" element={<Pages.CourseLivePage />} />
              <Route path="office-hours" element={<Pages.CourseOfficeHoursPage />} />
              <Route path="notebook" element={<Pages.CourseNotebookPage />} />
              <Route path="calendar" element={<Pages.CourseCalendarPage />} />
              <Route path="my-grades" element={<Pages.CourseMyGrades />} />
              <Route path="gradebook" element={<Pages.CourseGradebook />} />
              <Route path="reports" element={<Pages.CourseStudentReportsPage />} />
              <Route path="at-risk" element={<Pages.CourseAtRiskPage />} />
              <Route path="event-log" element={<Pages.CourseEventLogPage />} />
              <Route path="students/:enrollmentId/progress" element={<Pages.StudentProgressPage />} />
              <Route path="standards-gradebook" element={<Pages.CourseStandardsGradebook />} />
              <Route path="standards-coverage" element={<Pages.CourseStandardsCoveragePage />} />
              <Route path="mastery-heatmap" element={<Pages.CourseMasteryHeatmap />} />
              <Route path="outcomes-report" element={<Pages.CourseOutcomesReport />} />
              <Route path="whats-working" element={<Pages.CourseWhatsWorking />} />
              <Route path="enrollments" element={<Pages.CourseEnrollments />} />
              <Route path="attendance" element={<Pages.CourseAttendance />} />
              <Route path="behavior" element={<Pages.CourseBehavior />} />
              <Route path="report-cards" element={<Pages.CourseReportCards />} />
              <Route path="reading-dashboard" element={<Pages.ReadingDashboardPage />} />
              <Route path="whiteboard" element={<Pages.CourseWhiteboardPage />} />
              <Route path="whiteboard/:boardId" element={<Pages.CourseWhiteboardPage />} />
              <Route path="final-grades" element={<Pages.FinalGradeSubmission />} />
              <Route path="evaluation" element={<Pages.CourseEvaluation />} />
              <Route path="evaluation-results" element={<Pages.CourseEvaluationResults />} />
              <Route index element={<Pages.CourseDetail />} />
            </Route>
            <Route path="/calendar" element={<Pages.Calendar />} />
            <Route path="/admin/accommodations" element={<Pages.AdminAccommodationsPage />} />
            <Route path="/admin/accommodations/audit" element={<Pages.AccommodationAuditPage />} />
            <Route path="/admin/ccr/achievements" element={<Pages.AdminCCRAchievementsPage />} />
            <Route path="/admin/quarantine" element={<Pages.AdminQuarantinePage />} />
            <Route path="/catalog" element={<Pages.CourseCatalogPage />} />
            <Route path="/portfolios" element={<Pages.MyPortfoliosPage />} />
            <Route path="/portfolios/:pid/content/:aid" element={<Pages.PortfolioArtifactContentPage />} />
            <Route path="/portfolios/:pid" element={<Pages.PortfolioEditorPage />} />
            <Route path="/library/:orgId" element={<Pages.LibraryCatalogPage />} />
            <Route path="/reading-log" element={<Pages.ReadingLogPage />} />
            <Route path="/admin/compliance/iso" element={<Pages.IsoComplianceAdminPage />} />
            <Route path="/admin/compliance/security-reports" element={<Pages.SecurityDisclosureAdminPage />} />
            <Route path="/admin/compliance/backup" element={<Pages.BackupOpsAdminPage />} />
            <Route path="/admin/caption-compliance" element={<Pages.CaptionComplianceReportPage />} />
            <Route path="/admin/attendance/dashboard" element={<Pages.AttendanceDashboard />} />
            <Route path="/admin/attendance/export" element={<Pages.AttendanceExport />} />
            <Route path="/admin/behavior/dashboard" element={<Pages.BehaviorDashboard />} />
            <Route path="/admin/broadcasts/:orgId" element={<Pages.BroadcastComposer />} />
            <Route path="/admin/conferences/schedule" element={<Pages.ConferenceScheduleGrid />} />
            <Route path="/admin/demographics/student" element={<Pages.StudentDemographicsPage />} />
            <Route path="/admin/demographics/title1" element={<Pages.Title1ReportPage />} />
            <Route path="/admin/content-filter" element={<Pages.ContentFilterSettingsPage />} />
            <Route path="/admin/sis" element={<Pages.SisIntegrationPage />} />
            <Route path="/admin/bookstore" element={<Pages.BookstoreIntegrationPage />} />
            <Route path="/admin/final-grades/status" element={<Pages.GradeSubmissionStatus />} />
            <Route path="/admin/incompletes" element={<Pages.IncompletesAdminPage />} />
            <Route path="/admin/academic-calendar" element={<Pages.AcademicCalendarAdminPage />} />
            <Route path="/admin/consent-studies" element={<Pages.ConsentStudiesAdminPage />} />
            <Route path="/admin/accessibility" element={<Pages.AccessibilityServicesPage />} />
            <Route path="/admin/evaluations/templates" element={<Pages.EvaluationTemplates />} />
            <Route path="/admin/evaluations/report" element={<Pages.EvaluationReport />} />
            <Route path="/reports" element={<Pages.Reports />} />
            <Route path="/inbox" element={<Pages.Inbox />} />
            <Route path="/settings" element={<Navigate to="/settings/account" replace />} />
            <Route path="/settings/ai" element={<Navigate to="/settings/ai/models" replace />} />
            <Route path="/settings/ai/:aiSection" element={<Pages.Settings />} />
            <Route path="/settings/:tab" element={<Pages.Settings />} />
          </Route>
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  )
}
