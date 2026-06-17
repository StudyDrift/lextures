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
  Route,
  School,
  Shield,
  ShieldAlert,
  ShieldCheck,
  Users,
  Video,
  Archive,
  Activity,
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
    ffContentFilterIntegration,
    ffIncompleteGradeWorkflow,
    ffGradeSubmission,
    ffAcademicCalendar,
    ffCourseEvaluations,
    ffCoCurricularTranscript,
    ffLibrary,
    ffLearningPaths,
  } = usePlatformFeatures()

  const captionsEnabled = videoCaptionsEnabled || autoCaptioningEnabled
  const broadcastsPath = orgId ? `/admin/broadcasts/${encodeURIComponent(orgId)}` : '/admin/broadcasts'
  const libraryPath = orgId ? `/library/${encodeURIComponent(orgId)}` : '/library'

  const active = (path: string) =>
    location.pathname === path || location.pathname.startsWith(`${path}/`)

  const showCcrAdmin = canManageAccommodations && ffCoCurricularTranscript
  if (!canManageRbac && !showCcrAdmin) return null

  return (
    <>
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
          <SideNavLink
            to="/admin/compliance/backup"
            className={() => (active('/admin/compliance/backup') ? sideNavActiveClass : '')}
            icon={<Archive className="h-5 w-5" />}
          >
            Backup operations
          </SideNavLink>
          {avScanningEnabled ? (
            <SideNavLink
              to="/admin/quarantine"
              className={() => (active('/admin/quarantine') ? sideNavActiveClass : '')}
              icon={<Shield className="h-5 w-5" />}
            >
              File quarantine
            </SideNavLink>
          ) : null}
          {captionsEnabled ? (
            <SideNavLink
              to="/admin/caption-compliance"
              className={() => (active('/admin/caption-compliance') ? sideNavActiveClass : '')}
              icon={<Video className="h-5 w-5" />}
            >
              Caption compliance
            </SideNavLink>
          ) : null}

          <SideNavSectionLabel>Integrations</SideNavSectionLabel>
          {ffSisIntegration ? (
            <SideNavLink
              to={orgPath('/admin/sis', orgId)}
              className={() => (active('/admin/sis') ? sideNavActiveClass : '')}
              icon={<School className="h-5 w-5" />}
            >
              SIS integration
            </SideNavLink>
          ) : null}
          {ffContentFilterIntegration ? (
            <SideNavLink
              to={orgPath('/admin/content-filter', orgId)}
              className={() => (active('/admin/content-filter') ? sideNavActiveClass : '')}
              icon={<Filter className="h-5 w-5" />}
            >
              Content filter
            </SideNavLink>
          ) : null}

          <SideNavSectionLabel>Student records</SideNavSectionLabel>
          {ffDemographics ? (
            <>
              <SideNavLink
                to="/admin/demographics/student"
                className={() => (active('/admin/demographics/student') ? sideNavActiveClass : '')}
                icon={<Users className="h-5 w-5" />}
              >
                Student demographics
              </SideNavLink>
              <SideNavLink
                to="/admin/demographics/title1"
                className={() => (active('/admin/demographics/title1') ? sideNavActiveClass : '')}
                icon={<BarChart2 className="h-5 w-5" />}
              >
                Title I report
              </SideNavLink>
            </>
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
          {ffGradeSubmission ? (
            <SideNavLink
              to="/admin/final-grades/status"
              className={() => (active('/admin/final-grades') ? sideNavActiveClass : '')}
              icon={<ClipboardList className="h-5 w-5" />}
            >
              Final grade status
            </SideNavLink>
          ) : null}
          {ffCoCurricularTranscript && canManageAccommodations ? (
            <SideNavLink
              to="/admin/ccr/achievements"
              className={() => (active('/admin/ccr') ? sideNavActiveClass : '')}
              icon={<Award className="h-5 w-5" />}
            >
              CCR achievements
            </SideNavLink>
          ) : null}
          <SideNavSectionLabel>School operations</SideNavSectionLabel>
          {ffClassroomSignals ? (
            <>
              <SideNavLink
                to="/admin/attendance/dashboard"
                className={() => (active('/admin/attendance/dashboard') ? sideNavActiveClass : '')}
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
                className={() => (active('/admin/behavior/dashboard') ? sideNavActiveClass : '')}
                icon={<Activity className="h-5 w-5" />}
              >
                Behavior dashboard
              </SideNavLink>
            </>
          ) : null}
          {ffAcademicCalendar ? (
            <SideNavLink
              to={orgPath('/admin/academic-calendar', orgId)}
              className={() => (active('/admin/academic-calendar') ? sideNavActiveClass : '')}
              icon={<CalendarRange className="h-5 w-5" />}
            >
              Academic calendar
            </SideNavLink>
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
                to="/admin/conferences/schedule"
                className={() => (active('/admin/conferences/schedule') ? sideNavActiveClass : '')}
                icon={<CalendarRange className="h-5 w-5" />}
              >
                Conference schedule
              </SideNavLink>
              <SideNavLink
                to="/conferences/availability"
                className={() =>
                  location.pathname === '/conferences/availability' ? sideNavActiveClass : ''
                }
                icon={<CalendarRange className="h-5 w-5" />}
              >
                Conference availability
              </SideNavLink>
            </>
          ) : null}
          {ffCourseEvaluations ? (
            <>
              <SideNavLink
                to="/admin/evaluations/templates"
                className={() => (active('/admin/evaluations/templates') ? sideNavActiveClass : '')}
                icon={<ClipboardList className="h-5 w-5" />}
              >
                Evaluation templates
              </SideNavLink>
              <SideNavLink
                to="/admin/evaluations/report"
                className={() => (active('/admin/evaluations/report') ? sideNavActiveClass : '')}
                icon={<BarChart2 className="h-5 w-5" />}
              >
                Evaluation report
              </SideNavLink>
            </>
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
          {ffLearningPaths ? (
            <SideNavLink
              to="/creator/learning-paths"
              className={() => (active('/creator/learning-paths') ? sideNavActiveClass : '')}
              icon={<Route className="h-5 w-5" />}
            >
              Learning path builder
            </SideNavLink>
          ) : null}
        </>
      ) : null}
    </>
  )
}