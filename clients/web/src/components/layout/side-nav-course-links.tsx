import { useLocation } from 'react-router-dom'
import {
  ArrowLeft,
  Award,
  BarChart3,
  BookMarked,
  Calendar,
  ClipboardCheck,
  ClipboardList,
  Clock,
  FileText,
  FolderOpen,
  Layers,
  LayoutDashboard,
  Lightbulb,
  ListChecks,
  Library,
  MessageSquare,
  MessagesSquare,
  NotebookPen,
  Gamepad2,
  LayoutGrid,
  PenLine,
  Pencil,
  Send,
  Settings,
  Star,
  Target,
  TrendingUp,
  Users,
  UsersRound,
  Video,
  AlertTriangle,
  Activity,
} from 'lucide-react'
import { atRiskFeatureEnabled, atRiskI18n } from '../../lib/at-risk-i18n'
import {
  outcomesReportFeatureEnabled,
  studentProgressFeatureEnabled,
  xapiEmissionFeatureEnabled,
} from '../../lib/platform-features'
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { usePermissions } from '../../context/use-permissions'
import {
  courseEnrollmentsReadPermission,
  courseGradebookViewPermission,
  courseItemCreatePermission,
  courseItemsCreatePermission,
  viewerIsCourseStaffEnrollment,
  viewerShouldHideCourseEnrollmentsNav,
  viewerShouldShowMyGradesNav,
} from '../../lib/courses-api'
import { useCourseViewAs } from '../../lib/course-view-as'
import { useViewerEnrollmentRoles } from '../../lib/use-viewer-enrollment-roles'
import { sideNavActiveClass } from './side-nav-styles'
import { SideNavLink } from './side-nav-link'
import { SideNavSectionLabel } from './side-nav-section-label'

type SideNavCourseLinksProps = {
  courseCode: string
}

export function SideNavCourseLinks({ courseCode }: SideNavCourseLinksProps) {
  const location = useLocation()
  const {
    notebookEnabled,
    feedEnabled,
    calendarEnabled,
    questionBankEnabled,
    standardsAlignmentEnabled,
    discussionsEnabled,
    collabDocsEnabled,
    sbgEnabled,
    liveSessionsEnabled,
    groupSpacesEnabled,
    officeHoursEnabled,
    filesEnabled,
    attendanceEnabled,
    whiteboardEnabled,
    reportCardsEnabled,
    visualBoardsEnabled,
    interactiveQuizzesEnabled,
    screenShareEnabled,
  } = useCourseNavFeatures()
  const { allows, loading: permLoading } = usePermissions()
  const {
    instructorInsightsEnabled,
    ffLibrary,
    ffCourseEvaluations,
    ffGradeSubmission,
    ffClassroomSignals,
  } = usePlatformFeatures()
  const courseViewPreview = useCourseViewAs(courseCode)
  const viewerEnrollmentRoles = useViewerEnrollmentRoles(courseCode)

  const base = `/courses/${encodeURIComponent(courseCode)}`
  const canViewGradebook = !permLoading && allows(courseGradebookViewPermission(courseCode))
  const canViewEnrollments =
    viewerEnrollmentRoles !== null &&
    viewerIsCourseStaffEnrollment(viewerEnrollmentRoles) &&
    !viewerShouldHideCourseEnrollmentsNav(viewerEnrollmentRoles, courseViewPreview) &&
    !permLoading &&
    allows(courseEnrollmentsReadPermission(courseCode))
  const canViewMyGrades = viewerShouldShowMyGradesNav(viewerEnrollmentRoles, courseViewPreview)
  const canManageCourse = !permLoading && allows(courseItemCreatePermission(courseCode))
  const canManageQuestionBank = !permLoading && allows(courseItemsCreatePermission(courseCode))

  const boardsNavVisible = visualBoardsEnabled
  const liveQuizzesNavVisible = interactiveQuizzesEnabled
  const screenShareNavVisible = screenShareEnabled
  const showCollaboration =
    feedEnabled ||
    discussionsEnabled ||
    collabDocsEnabled ||
    groupSpacesEnabled ||
    liveSessionsEnabled ||
    officeHoursEnabled ||
    (whiteboardEnabled && canManageCourse) ||
    boardsNavVisible

  const showYourLearning =
    notebookEnabled || calendarEnabled || attendanceEnabled || canViewMyGrades

  const showGradingInsights =
    canViewGradebook || (standardsAlignmentEnabled && canManageCourse)

  const showAssessmentTools =
    (canManageQuestionBank && questionBankEnabled) || liveQuizzesNavVisible || screenShareNavVisible

  return (
    <>
      <SideNavLink to="/courses" icon={<ArrowLeft className="h-5 w-5" />}>
        Back
      </SideNavLink>
      <SideNavLink to={base} end icon={<LayoutDashboard className="h-5 w-5" />}>
        Dashboard
      </SideNavLink>

      <SideNavSectionLabel first>Content</SideNavSectionLabel>
      {filesEnabled && canManageCourse ? (
        <SideNavLink to={`${base}/files`} icon={<FolderOpen className="h-5 w-5" />}>
          Files
        </SideNavLink>
      ) : null}
      <SideNavLink to={`${base}/modules`} icon={<Layers className="h-5 w-5" />}>
        Modules
      </SideNavLink>
      <SideNavLink to={`${base}/syllabus`} icon={<FileText className="h-5 w-5" />}>
        Syllabus
      </SideNavLink>

      {showCollaboration ? (
        <>
          <SideNavSectionLabel>Collaboration</SideNavSectionLabel>
          {boardsNavVisible ? (
            <SideNavLink to={`${base}/boards`} icon={<LayoutGrid className="h-5 w-5" />}>
              Boards
            </SideNavLink>
          ) : null}
          {collabDocsEnabled ? (
            <SideNavLink to={`${base}/collab-docs`} icon={<PenLine className="h-5 w-5" />}>
              Collab docs
            </SideNavLink>
          ) : null}
          {discussionsEnabled ? (
            <SideNavLink to={`${base}/discussions`} icon={<MessagesSquare className="h-5 w-5" />}>
              Discussions
            </SideNavLink>
          ) : null}
          {feedEnabled ? (
            <SideNavLink to={`${base}/feed`} icon={<MessageSquare className="h-5 w-5" />}>
              Feed
            </SideNavLink>
          ) : null}
          {groupSpacesEnabled ? (
            <SideNavLink to={`${base}/groups`} icon={<UsersRound className="h-5 w-5" />}>
              Groups
            </SideNavLink>
          ) : null}
          {liveSessionsEnabled ? (
            <SideNavLink to={`${base}/live`} icon={<Video className="h-5 w-5" />}>
              Live Sessions
            </SideNavLink>
          ) : null}
          {officeHoursEnabled ? (
            <SideNavLink to={`${base}/office-hours`} icon={<Clock className="h-5 w-5" />}>
              Office Hours
            </SideNavLink>
          ) : null}
          {whiteboardEnabled && canManageCourse ? (
            <SideNavLink to={`${base}/whiteboard`} icon={<Pencil className="h-5 w-5" />}>
              Whiteboard
            </SideNavLink>
          ) : null}
        </>
      ) : null}

      {showYourLearning ? (
        <>
          <SideNavSectionLabel>Your learning</SideNavSectionLabel>
          {attendanceEnabled ? (
            <SideNavLink to={`${base}/attendance`} icon={<ClipboardList className="h-5 w-5" />}>
              Attendance
            </SideNavLink>
          ) : null}
          {calendarEnabled ? (
            <SideNavLink to={`${base}/calendar`} icon={<Calendar className="h-5 w-5" />}>
              Calendar
            </SideNavLink>
          ) : null}
          {canViewMyGrades ? (
            <SideNavLink to={`${base}/my-grades`} icon={<Award className="h-5 w-5" />}>
              My grades
            </SideNavLink>
          ) : null}
          {notebookEnabled ? (
            <SideNavLink to={`${base}/notebook`} icon={<NotebookPen className="h-5 w-5" />}>
              Notebook
            </SideNavLink>
          ) : null}
        </>
      ) : null}

      {showAssessmentTools ? (
        <>
          <SideNavSectionLabel>Assessment</SideNavSectionLabel>
          {liveQuizzesNavVisible ? (
            <SideNavLink to={`${base}/live-quizzes`} icon={<Gamepad2 className="h-5 w-5" />}>
              Live Quizzes
            </SideNavLink>
          ) : null}
          {screenShareNavVisible ? (
            <SideNavLink to={`${base}/screen-share`} icon={<Video className="h-5 w-5" />}>
              Screen share
            </SideNavLink>
          ) : null}
          {canManageQuestionBank && questionBankEnabled ? (
            <>
              <SideNavLink to={`${base}/misconception-report`} icon={<Lightbulb className="h-5 w-5" />}>
                Misconceptions
              </SideNavLink>
              <SideNavLink to={`${base}/questions`} icon={<ListChecks className="h-5 w-5" />}>
                Question bank
              </SideNavLink>
            </>
          ) : null}
        </>
      ) : null}

      {showGradingInsights ? (
        <>
          <SideNavSectionLabel>Grades & insights</SideNavSectionLabel>
          {canViewGradebook && atRiskFeatureEnabled() ? (
            <SideNavLink to={`${base}/at-risk`} icon={<AlertTriangle className="h-5 w-5" />}>
              {atRiskI18n.title}
            </SideNavLink>
          ) : null}
          {ffClassroomSignals && canViewGradebook ? (
            <SideNavLink to={`${base}/behavior`} icon={<Activity className="h-5 w-5" />}>
              Behavior
            </SideNavLink>
          ) : null}
          {ffCourseEvaluations && canViewGradebook ? (
            <SideNavLink to={`${base}/evaluation-results`} icon={<Star className="h-5 w-5" />}>
              Evaluation results
            </SideNavLink>
          ) : null}
          {canViewGradebook && xapiEmissionFeatureEnabled() ? (
            <SideNavLink to={`${base}/event-log`} icon={<Activity className="h-5 w-5" />}>
              Event log
            </SideNavLink>
          ) : null}
          {ffGradeSubmission && canViewGradebook ? (
            <SideNavLink to={`${base}/final-grades`} icon={<Send className="h-5 w-5" />}>
              Final grades
            </SideNavLink>
          ) : null}
          {canViewGradebook ? (
            <SideNavLink to={`${base}/gradebook`} icon={<ClipboardList className="h-5 w-5" />}>
              Gradebook
            </SideNavLink>
          ) : null}
          {sbgEnabled && canViewGradebook ? (
            <SideNavLink to={`${base}/mastery-heatmap`} icon={<TrendingUp className="h-5 w-5" />}>
              Mastery heatmap
            </SideNavLink>
          ) : null}
          {canViewGradebook && outcomesReportFeatureEnabled() ? (
            <SideNavLink to={`${base}/outcomes-report`} icon={<Target className="h-5 w-5" />}>
              Outcomes report
            </SideNavLink>
          ) : null}
          {ffLibrary && canViewGradebook ? (
            <SideNavLink to={`${base}/reading-dashboard`} icon={<Library className="h-5 w-5" />}>
              Reading dashboard
            </SideNavLink>
          ) : null}
          {reportCardsEnabled && canViewGradebook ? (
            <SideNavLink to={`${base}/report-cards`} icon={<ClipboardCheck className="h-5 w-5" />}>
              Report cards
            </SideNavLink>
          ) : null}
          {canViewGradebook && studentProgressFeatureEnabled() ? (
            <SideNavLink to={`${base}/reports`} icon={<BarChart3 className="h-5 w-5" />}>
              Reports
            </SideNavLink>
          ) : null}
          {standardsAlignmentEnabled && (canViewGradebook || canManageCourse) ? (
            <SideNavLink to={`${base}/standards-coverage`} icon={<BookMarked className="h-5 w-5" />}>
              Standards coverage
            </SideNavLink>
          ) : null}
          {sbgEnabled && canViewGradebook ? (
            <SideNavLink to={`${base}/standards-gradebook`} icon={<BookMarked className="h-5 w-5" />}>
              Standards gradebook
            </SideNavLink>
          ) : null}
          {instructorInsightsEnabled && canViewGradebook ? (
            <SideNavLink to={`${base}/whats-working`} icon={<TrendingUp className="h-5 w-5" />}>
              What&apos;s working
            </SideNavLink>
          ) : null}
        </>
      ) : null}

      {canViewEnrollments ? (
        <>
          <SideNavSectionLabel>People</SideNavSectionLabel>
          <SideNavLink to={`${base}/enrollments`} icon={<Users className="h-5 w-5" />}>
            Enrollments
          </SideNavLink>
        </>
      ) : null}

      {canManageCourse ? (
        <>
          <SideNavSectionLabel>Manage</SideNavSectionLabel>
          <SideNavLink
            to={`${base}/settings/general`}
            className={() => {
              const settingsPrefix = `${base}/settings`
              const onSettings =
                location.pathname === settingsPrefix ||
                location.pathname.startsWith(`${settingsPrefix}/`)
              return onSettings ? sideNavActiveClass : ''
            }}
            icon={<Settings className="h-5 w-5" />}
          >
            Settings
          </SideNavLink>
        </>
      ) : null}
    </>
  )
}
