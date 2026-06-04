import { useEffect } from 'react'
import { Navigate, Route, Routes, useLocation, useNavigate } from 'react-router-dom'
import { RequireAuth } from './auth/require-auth'
import { ApiErrorBoundary } from './components/api-error-boundary'
import { AppShell } from './components/layout/app-shell'
import Calendar from './pages/lms/calendar'
import CourseCalendarPage from './pages/lms/course-calendar-page'
import CourseEnrollments from './pages/lms/course-enrollments'
import CourseFeedPage from './pages/lms/course-feed-page'
import CourseDiscussionsPage from './pages/lms/course-discussions-page'
import CourseCollabDocsPage from './pages/lms/course-collab-docs-page'
import CourseFilesPage from './pages/lms/course-files-page'
import CourseGroupsPage from './pages/lms/course-groups-page'
import CourseGradebook from './pages/lms/course-gradebook'
import CourseAtRiskPage from './pages/lms/course-at-risk'
import StudentProgressPage from './pages/lms/student-progress-page'
import CourseMyGrades from './pages/lms/course-my-grades'
import AdminAccommodationsPage from './pages/lms/admin-accommodations-page'
import AccommodationAuditPage from './pages/lms/accommodation-audit-page'
import AdminQuarantinePage from './pages/lms/admin-quarantine-page'
import CourseCreate from './pages/lms/course-create'
import CourseDetail from './pages/lms/course-detail'
import CourseLayout from './pages/lms/course-layout'
import CourseModuleAssignmentPage from './pages/lms/course-module-assignment-page'
import ModerationDashboard from './pages/lms/moderation-dashboard'
import CourseModuleContentPage from './pages/lms/course-module-content-page'
import CourseModuleExternalLinkPage from './pages/lms/course-module-external-link-page'
import CourseModuleH5PPage from './pages/lms/course-module-h5p-page'
import CourseModuleLtiPage from './pages/lms/course-module-lti-page'
import CourseModuleVibeActivityPage from './pages/lms/course-module-vibe-activity-page'
import CourseModuleQuizPage from './pages/lms/course-module-quiz-page'
import CourseDiagnosticPage from './pages/lms/course-diagnostic-page'
import { CourseQuestionBankPage } from './pages/lms/course-question-bank-page'
import CourseMisconceptionReportPage from './pages/lms/course-misconception-report-page'
import CourseModules from './pages/lms/course-modules'
import CourseLivePage from './pages/lms/course-live-page'
import CourseOfficeHoursPage from './pages/lms/course-office-hours-page'
import CourseNotebookPage from './pages/lms/course-notebook-page'
import CourseSettings from './pages/lms/course-settings'
import CourseStandardsCoveragePage from './pages/lms/course-standards-coverage-page'
import CourseStandardsGradebook from './pages/lms/course-standards-gradebook'
import CourseSyllabus from './pages/lms/course-syllabus'
import Courses from './pages/lms/courses'
import Dashboard from './pages/lms/dashboard'
import StudyInsightsPage from './pages/lms/study-insights-page'
import AskAiPage from './pages/lms/ask-ai-page'
import ReviewSessionPage from './pages/lms/review-session-page'
import Inbox from './pages/lms/inbox'
import GlobalNotebookPage from './pages/lms/global-notebook-page'
import MyNotebooksPage from './pages/lms/my-notebooks-page'
import Reports from './pages/lms/reports'
import CourseEventLogPage from './pages/lms/course-event-log'
import CourseMasteryHeatmap from './pages/lms/course-mastery-heatmap'
import CourseOutcomesReport from './pages/lms/course-outcomes-report'
import CourseWhatsWorking from './pages/lms/course-whats-working'
import Settings from './pages/lms/settings'
import ForgotPassword from './pages/forgot-password'
import Login from './pages/login'
import MfaLogin from './pages/mfa-login'
import SamlCallback from './pages/saml-callback'
import SsoError from './pages/sso-error'
import AiDisclosurePage from './pages/ai-disclosure-page'
import MagicLinkPage from './pages/magic-link'
import ResetPassword from './pages/reset-password'
import Signup from './pages/signup'
import ParentDashboard from './pages/lms/parent/parent-dashboard'
import TrustCenterPage from './pages/trust-center-page'
import IsoComplianceAdminPage from './pages/iso-compliance-admin-page'
import SecurityDisclosureAdminPage from './pages/security-disclosure-admin-page'
import BackupOpsAdminPage from './pages/backup-ops-admin-page'
import CaptionComplianceReportPage from './pages/admin/caption-compliance-report'
import AttendanceDashboard from './pages/admin/AttendanceDashboard'
import AttendanceExport from './pages/admin/AttendanceExport'
import BehaviorDashboard from './pages/admin/BehaviorDashboard'
import CourseAttendance from './pages/lms/CourseAttendance'
import CourseBehavior from './pages/lms/CourseBehavior'
import CourseReportCards from './pages/lms/CourseReportCards'
import LibraryCatalogPage from './pages/lms/library-catalog-page'
import ReadingLogPage from './pages/lms/reading-log-page'
import ReadingDashboardPage from './pages/lms/reading-dashboard-page'
import PrivacyCentrePage from './pages/privacy-centre-page'
import CliAuthPage from './pages/cli-auth'
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
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route path="/login/magic-link" element={<MagicLinkPage />} />
      <Route path="/login/mfa" element={<MfaLogin />} />
      <Route path="/saml-callback" element={<SamlCallback />} />
      <Route path="/sso-error" element={<SsoError />} />
      <Route path="/signup" element={<Signup />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/reset-password" element={<ResetPassword />} />
      <Route path="/ai-disclosure" element={<AiDisclosurePage />} />
      <Route path="/trust" element={<TrustCenterPage />} />
      <Route element={<RequireAuth />}>
        <Route path="/cli-auth" element={<CliAuthPage />} />
        <Route
          element={
            <ApiErrorBoundary>
              <AppShell />
            </ApiErrorBoundary>
          }
        >
          <Route path="/" element={<Dashboard />} />
          <Route path="/privacy-centre" element={<PrivacyCentrePage />} />
          <Route path="/me/study-insights" element={<StudyInsightsPage />} />
          <Route path="/parent" element={<ParentDashboard />} />
          <Route path="/ai" element={<AskAiPage />} />
          <Route path="/review" element={<ReviewSessionPage />} />
          <Route path="/courses" element={<Courses />} />
          <Route path="/notebooks/global" element={<GlobalNotebookPage />} />
          <Route path="/notebooks" element={<MyNotebooksPage />} />
          <Route path="/courses/create" element={<CourseCreate />} />
          <Route path="/courses/:courseCode" element={<CourseLayout />}>
            <Route path="settings/*" element={<CourseSettings />} />
            <Route path="feed" element={<CourseFeedPage />} />
            <Route path="discussions" element={<CourseDiscussionsPage />} />
            <Route path="collab-docs" element={<CourseCollabDocsPage />} />
            <Route path="collab-docs/:docId" element={<CourseCollabDocsPage />} />
            <Route path="files" element={<CourseFilesPage />} />
            <Route path="groups" element={<CourseGroupsPage />} />
            <Route path="syllabus" element={<CourseSyllabus />} />
            <Route path="modules/content/:itemId" element={<CourseModuleContentPage />} />
            <Route path="modules/assignment/:itemId" element={<CourseModuleAssignmentPage />} />
            <Route
              path="modules/assignment/:itemId/moderation"
              element={<ModerationDashboard />}
            />
            <Route path="modules/quiz/:itemId" element={<CourseModuleQuizPage />} />
            <Route path="diagnostic" element={<CourseDiagnosticPage />} />
            <Route path="modules/external-link/:itemId" element={<CourseModuleExternalLinkPage />} />
            <Route path="modules/h5p/:itemId" element={<CourseModuleH5PPage />} />
            <Route path="modules/lti/:itemId" element={<CourseModuleLtiPage />} />
            <Route path="modules/vibe-activity/:itemId" element={<CourseModuleVibeActivityPage />} />
            <Route path="questions" element={<CourseQuestionBankPage />} />
            <Route path="misconception-report" element={<CourseMisconceptionReportPage />} />
            <Route path="modules" element={<CourseModules />} />
            <Route path="live" element={<CourseLivePage />} />
            <Route path="office-hours" element={<CourseOfficeHoursPage />} />
            <Route path="notebook" element={<CourseNotebookPage />} />
            <Route path="calendar" element={<CourseCalendarPage />} />
            <Route path="my-grades" element={<CourseMyGrades />} />
            <Route path="gradebook" element={<CourseGradebook />} />
            <Route path="at-risk" element={<CourseAtRiskPage />} />
            <Route path="event-log" element={<CourseEventLogPage />} />
            <Route path="students/:enrollmentId/progress" element={<StudentProgressPage />} />
            <Route path="standards-gradebook" element={<CourseStandardsGradebook />} />
            <Route path="standards-coverage" element={<CourseStandardsCoveragePage />} />
            <Route path="mastery-heatmap" element={<CourseMasteryHeatmap />} />
            <Route path="outcomes-report" element={<CourseOutcomesReport />} />
            <Route path="whats-working" element={<CourseWhatsWorking />} />
            <Route path="enrollments" element={<CourseEnrollments />} />
            <Route path="attendance" element={<CourseAttendance />} />
            <Route path="behavior" element={<CourseBehavior />} />
            <Route path="report-cards" element={<CourseReportCards />} />
            <Route path="reading-dashboard" element={<ReadingDashboardPage />} />
            <Route index element={<CourseDetail />} />
          </Route>
          <Route path="/calendar" element={<Calendar />} />
          <Route path="/admin/accommodations" element={<AdminAccommodationsPage />} />
          <Route path="/admin/accommodations/audit" element={<AccommodationAuditPage />} />
          <Route path="/admin/quarantine" element={<AdminQuarantinePage />} />
          <Route path="/library/:orgId" element={<LibraryCatalogPage />} />
          <Route path="/reading-log" element={<ReadingLogPage />} />
          <Route path="/admin/compliance/iso" element={<IsoComplianceAdminPage />} />
          <Route path="/admin/compliance/security-reports" element={<SecurityDisclosureAdminPage />} />
          <Route path="/admin/compliance/backup" element={<BackupOpsAdminPage />} />
          <Route path="/admin/caption-compliance" element={<CaptionComplianceReportPage />} />
          <Route path="/admin/attendance/dashboard" element={<AttendanceDashboard />} />
          <Route path="/admin/attendance/export" element={<AttendanceExport />} />
          <Route path="/admin/behavior/dashboard" element={<BehaviorDashboard />} />
          <Route path="/reports" element={<Reports />} />
          <Route path="/inbox" element={<Inbox />} />
          <Route path="/settings" element={<Navigate to="/settings/account" replace />} />
          <Route path="/settings/ai" element={<Navigate to="/settings/ai/models" replace />} />
          <Route path="/settings/ai/:aiSection" element={<Settings />} />
          <Route path="/settings/:tab" element={<Settings />} />
        </Route>
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
