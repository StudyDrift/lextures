import { lazy, Suspense } from 'react'
import { useLocation } from 'react-router-dom'
import {
  Award,
  BarChart2,
  CalendarRange,
  ClipboardList,
  FileSpreadsheet,
  Filter,
  Library,
  Megaphone,
  Mail,
  Route,
  School,
  Shield,
  ShieldAlert,
  ShieldCheck,
  Users,
  Video,
  Webhook,
  Archive,
  Activity,
  Clock,
  LayoutGrid,
} from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { usePermissions } from '../../context/use-permissions'
import {
  PERM_ACCOMMODATIONS_MANAGE,
  PERM_RBAC_MANAGE,
} from '../../lib/rbac-api'
import { sideNavActiveClass } from './side-nav-styles'
import { SideNavLink } from './side-nav-link'
import { SideNavSectionLabel } from './side-nav-section-label'
import { useViewerOrgId } from './use-viewer-org-id'

const SideNavAdminConsoleLink = lazy(() => import('./side-nav-admin-console-link'))

function orgPath(base: string, orgId: string | null): string {
  if (!orgId) return base
  const sep = base.includes('?') ? '&' : '?'
  return `${base}${sep}orgId=${encodeURIComponent(orgId)}`
}

export function SideNavAdminLinks() {
  const location = useLocation()
  const orgId = useViewerOrgId()
  const { allows, loading: permLoading } = usePermissions()
  const canManageRbac = !permLoading && allows(PERM_RBAC_MANAGE)
  const canManageAccommodations = !permLoading && allows(PERM_ACCOMMODATIONS_MANAGE)

  const {
    avScanningEnabled,
    videoCaptionsEnabled,
    autoCaptioningEnabled,
    ffClassroomSignals,
    ffDemographics,
    ffBroadcasts,
    ffConferenceScheduling,
    ffSisIntegration,
    ffWebhooks,
    ffContentFilterIntegration,
    ffVisualBoards,
    ffIncompleteGradeWorkflow,
    ffGradeSubmission,
    ffAcademicCalendar,
    ffCourseEvaluations,
    ffCoCurricularTranscript,
    ffLibrary,
    ffLearningPaths,
    adminConsoleEnabled,
    emailTemplateEditorEnabled,
    maintenanceBannerEnabled,
    ffTranscripts,
  } = usePlatformFeatures()

  const captionsEnabled = videoCaptionsEnabled || autoCaptioningEnabled
  const broadcastsPath = orgId ? `/admin/broadcasts/${encodeURIComponent(orgId)}` : '/admin/broadcasts'
  const libraryPath = orgId ? `/library/${encodeURIComponent(orgId)}` : '/library'

  const active = (path: string) =>
    location.pathname === path || location.pathname.startsWith(`${path}/`)

  const showCcrAdmin = canManageAccommodations && ffCoCurricularTranscript
  const showIntegrations = ffSisIntegration || ffWebhooks || ffContentFilterIntegration || emailTemplateEditorEnabled
  const showStudentRecords =
    ffDemographics ||
    ffIncompleteGradeWorkflow ||
    ffGradeSubmission ||
    (ffCoCurricularTranscript && canManageAccommodations)
  const showSchoolOperations =
    ffClassroomSignals ||
    ffAcademicCalendar ||
    (ffBroadcasts && !!orgId) ||
    ffConferenceScheduling ||
    ffCourseEvaluations ||
    (ffLibrary && !!orgId) ||
    ffLearningPaths

  if (!canManageRbac && !showCcrAdmin && !adminConsoleEnabled) {
    return null
  }

  return (
    <>
      {adminConsoleEnabled ? (
        <Suspense fallback={null}>
          <SideNavAdminConsoleLink orgId={orgId} />
        </Suspense>
      ) : null}
      {showCcrAdmin && !canManageRbac ? (
        <>
          <SideNavSectionLabel first>Student records</SideNavSectionLabel>
          <SideNavLink
            to="/admin/ccr/achievements"
            className={() => (active('/admin/ccr') ? sideNavActiveClass : '')}
            icon={<Award className="h-5 w-5" />}
          >
            CCR achievements
          </SideNavLink>
        </>
      ) : null}
      {canManageRbac ? (
        <>
          <SideNavSectionLabel first>Compliance & security</SideNavSectionLabel>
          <SideNavLink
            to="/admin/compliance/backup"
            className={() => (active('/admin/compliance/backup') ? sideNavActiveClass : '')}
            icon={<Archive className="h-5 w-5" />}
          >
            Backup operations
          </SideNavLink>
          {captionsEnabled ? (
            <SideNavLink
              to="/admin/caption-compliance"
              className={() => (active('/admin/caption-compliance') ? sideNavActiveClass : '')}
              icon={<Video className="h-5 w-5" />}
            >
              Caption compliance
            </SideNavLink>
          ) : null}
          {avScanningEnabled ? (
            <SideNavLink
              to="/admin/quarantine"
              className={() => (active('/admin/quarantine') ? sideNavActiveClass : '')}
              icon={<Shield className="h-5 w-5" />}
            >
              File quarantine
            </SideNavLink>
          ) : null}
          <SideNavLink
            to="/admin/compliance/iso"
            className={() => (active('/admin/compliance/iso') ? sideNavActiveClass : '')}
            icon={<ShieldCheck className="h-5 w-5" />}
          >
            ISO compliance
          </SideNavLink>
          <SideNavLink
            to="/admin/compliance/security-reports"
            className={() =>
              active('/admin/compliance/security-reports') ? sideNavActiveClass : ''
            }
            icon={<ShieldAlert className="h-5 w-5" />}
          >
            Security reports
          </SideNavLink>

          <SideNavSectionLabel>Platform</SideNavSectionLabel>
          {maintenanceBannerEnabled ? (
            <SideNavLink
              to={orgPath('/admin/banners', orgId)}
              className={() => (active('/admin/banners') ? sideNavActiveClass : '')}
              icon={<Megaphone className="h-5 w-5" />}
            >
              Notices
            </SideNavLink>
          ) : null}
          {ffTranscripts ? (
            <SideNavLink
              to={orgPath('/admin/transcripts', orgId)}
              className={() => (active('/admin/transcripts') ? sideNavActiveClass : '')}
              icon={<FileSpreadsheet className="h-5 w-5" />}
            >
              Transcripts
            </SideNavLink>
          ) : null}
          <SideNavLink
            to="/admin/scheduled-jobs"
            className={() => (active('/admin/scheduled-jobs') ? sideNavActiveClass : '')}
            icon={<Clock className="h-5 w-5" />}
          >
            Scheduled jobs
          </SideNavLink>
          {ffVisualBoards ? (
            <SideNavLink
              to={orgPath('/admin/boards', orgId)}
              className={() => (active('/admin/boards') ? sideNavActiveClass : '')}
              icon={<LayoutGrid className="h-5 w-5" />}
            >
              Collaboration boards
            </SideNavLink>
          ) : null}

          {showIntegrations ? (
            <>
              <SideNavSectionLabel>Integrations</SideNavSectionLabel>
              {ffContentFilterIntegration ? (
                <SideNavLink
                  to={orgPath('/admin/content-filter', orgId)}
                  className={() => (active('/admin/content-filter') ? sideNavActiveClass : '')}
                  icon={<Filter className="h-5 w-5" />}
                >
                  Content filter
                </SideNavLink>
              ) : null}
              {emailTemplateEditorEnabled ? (
                <SideNavLink
                  to={orgPath('/admin/email-templates', orgId)}
                  className={() => (active('/admin/email-templates') ? sideNavActiveClass : '')}
                  icon={<Mail className="h-5 w-5" />}
                >
                  Email templates
                </SideNavLink>
              ) : null}
              {ffSisIntegration ? (
                <SideNavLink
                  to={orgPath('/admin/sis', orgId)}
                  className={() => (active('/admin/sis') ? sideNavActiveClass : '')}
                  icon={<School className="h-5 w-5" />}
                >
                  SIS integration
                </SideNavLink>
              ) : null}
              {ffWebhooks ? (
                <SideNavLink
                  to={orgPath('/admin/webhooks', orgId)}
                  className={() => (active('/admin/webhooks') ? sideNavActiveClass : '')}
                  icon={<Webhook className="h-5 w-5" />}
                >
                  Webhooks
                </SideNavLink>
              ) : null}
            </>
          ) : null}

          {showStudentRecords ? (
            <>
              <SideNavSectionLabel>Student records</SideNavSectionLabel>
              {ffCoCurricularTranscript && canManageAccommodations ? (
                <SideNavLink
                  to="/admin/ccr/achievements"
                  className={() => (active('/admin/ccr') ? sideNavActiveClass : '')}
                  icon={<Award className="h-5 w-5" />}
                >
                  CCR achievements
                </SideNavLink>
              ) : null}
              {ffGradeSubmission ? (
                <SideNavLink
                  to="/admin/final-grades/status"
                  className={() => (active('/admin/final-grades') ? sideNavActiveClass : '')}
                  icon={<ClipboardList className="h-5 w-5" />}
                >
                  Final grade status
                </SideNavLink>
              ) : null}
              {ffIncompleteGradeWorkflow ? (
                <SideNavLink
                  to="/admin/incompletes"
                  className={() => (active('/admin/incompletes') ? sideNavActiveClass : '')}
                  icon={<FileSpreadsheet className="h-5 w-5" />}
                >
                  Incomplete grades
                </SideNavLink>
              ) : null}
              {ffDemographics ? (
                <>
                  <SideNavLink
                    to="/admin/demographics/student"
                    className={() =>
                      active('/admin/demographics/student') ? sideNavActiveClass : ''
                    }
                    icon={<Users className="h-5 w-5" />}
                  >
                    Student demographics
                  </SideNavLink>
                  <SideNavLink
                    to="/admin/demographics/title1"
                    className={() =>
                      active('/admin/demographics/title1') ? sideNavActiveClass : ''
                    }
                    icon={<BarChart2 className="h-5 w-5" />}
                  >
                    Title I report
                  </SideNavLink>
                </>
              ) : null}
            </>
          ) : null}

          {showSchoolOperations ? (
            <>
              <SideNavSectionLabel>School operations</SideNavSectionLabel>
              {ffAcademicCalendar ? (
                <SideNavLink
                  to={orgPath('/admin/academic-calendar', orgId)}
                  className={() => (active('/admin/academic-calendar') ? sideNavActiveClass : '')}
                  icon={<CalendarRange className="h-5 w-5" />}
                >
                  Academic calendar
                </SideNavLink>
              ) : null}
              {ffClassroomSignals ? (
                <>
                  <SideNavLink
                    to="/admin/attendance/dashboard"
                    className={() =>
                      active('/admin/attendance/dashboard') ? sideNavActiveClass : ''
                    }
                    icon={<ClipboardList className="h-5 w-5" />}
                  >
                    Attendance dashboard
                  </SideNavLink>
                  <SideNavLink
                    to="/admin/attendance/export"
                    className={() => (active('/admin/attendance/export') ? sideNavActiveClass : '')}
                    icon={<FileSpreadsheet className="h-5 w-5" />}
                  >
                    Attendance export
                  </SideNavLink>
                  <SideNavLink
                    to={orgPath('/admin/behavior/dashboard', orgId)}
                    className={() =>
                      active('/admin/behavior/dashboard') ? sideNavActiveClass : ''
                    }
                    icon={<Activity className="h-5 w-5" />}
                  >
                    Behavior dashboard
                  </SideNavLink>
                </>
              ) : null}
              {ffBroadcasts && orgId ? (
                <SideNavLink
                  to={broadcastsPath}
                  className={() => (active('/admin/broadcasts') ? sideNavActiveClass : '')}
                  icon={<Megaphone className="h-5 w-5" />}
                >
                  Broadcasts
                </SideNavLink>
              ) : null}
              {ffConferenceScheduling ? (
                <>
                  <SideNavLink
                    to="/conferences/availability"
                    className={() =>
                      location.pathname === '/conferences/availability' ? sideNavActiveClass : ''
                    }
                    icon={<CalendarRange className="h-5 w-5" />}
                  >
                    Conference availability
                  </SideNavLink>
                  <SideNavLink
                    to="/admin/conferences/schedule"
                    className={() =>
                      active('/admin/conferences/schedule') ? sideNavActiveClass : ''
                    }
                    icon={<CalendarRange className="h-5 w-5" />}
                  >
                    Conference schedule
                  </SideNavLink>
                </>
              ) : null}
              {ffCourseEvaluations ? (
                <>
                  <SideNavLink
                    to="/admin/evaluations/report"
                    className={() => (active('/admin/evaluations/report') ? sideNavActiveClass : '')}
                    icon={<BarChart2 className="h-5 w-5" />}
                  >
                    Evaluation report
                  </SideNavLink>
                  <SideNavLink
                    to="/admin/evaluations/templates"
                    className={() =>
                      active('/admin/evaluations/templates') ? sideNavActiveClass : ''
                    }
                    icon={<ClipboardList className="h-5 w-5" />}
                  >
                    Evaluation templates
                  </SideNavLink>
                </>
              ) : null}
              {ffLearningPaths ? (
                <SideNavLink
                  to="/creator/learning-paths"
                  className={() => (active('/creator/learning-paths') ? sideNavActiveClass : '')}
                  icon={<Route className="h-5 w-5" />}
                >
                  Learning path builder
                </SideNavLink>
              ) : null}
              {ffLibrary && orgId ? (
                <SideNavLink
                  to={libraryPath}
                  className={() => (active('/library') ? sideNavActiveClass : '')}
                  icon={<Library className="h-5 w-5" />}
                >
                  Library catalog
                </SideNavLink>
              ) : null}
            </>
          ) : null}
        </>
      ) : null}
    </>
  )
}
