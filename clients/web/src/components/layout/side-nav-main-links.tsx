import {
  Accessibility,
  Award,
  BarChart3,
  BookMarked,
  BookOpen,
  Bot,
  Calendar,
  CreditCard,
  DollarSign,
  FileText,
  FolderOpen,
  GraduationCap,
  Inbox,
  LayoutDashboard,
  Library,
  ListTodo,
  Route,
  RotateCcw,
  Settings,
  ShieldCheck,
  Sparkles,
  Store,
  UsersRound,
} from 'lucide-react'
import { useInboxUnreadCount } from '../../context/use-inbox-unread'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { usePermissions } from '../../context/use-permissions'
import {
  PERM_ACCOMMODATIONS_MANAGE,
  PERM_PARENT_DASHBOARD,
  PERM_REPORTS_VIEW,
} from '../../lib/rbac-api'
import { SideNavLink } from './side-nav-link'
import { SideNavSectionLabel } from './side-nav-section-label'

export function SideNavMainLinks() {
  const unreadInboxCount = useInboxUnreadCount()
  const { allows, loading: permLoading } = usePermissions()
  const {
    accommodationsEngineEnabled,
    selfReflectionEnabled,
    ffEportfolio,
    ffTranscripts,
    ffAdvisingIntegration,
    ffResearchConsent,
    ffAccessibilityIntake,
    ffCoCurricularTranscript,
    ffCeuTracking,
    ffStripeBilling,
    ffRevenueShare,
    ffLearningPaths,
    ffCompletionCredentials,
    ffCatalogIntegration,
    ffCourseMarketplace,
    ffLibrary,
    ffConferenceScheduling,
    ragNotebookEnabled,
  } = usePlatformFeatures()

  const canViewReports = !permLoading && allows(PERM_REPORTS_VIEW)
  const canManageAccommodations = !permLoading && allows(PERM_ACCOMMODATIONS_MANAGE)
  const isParent = !permLoading && allows(PERM_PARENT_DASHBOARD)

  const unreadBadge = unreadInboxCount > 0 && (
    <span
      className="inline-flex min-h-5 min-w-5 shrink-0 items-center justify-center rounded-full bg-red-600 px-1.5 text-[11px] font-semibold tabular-nums leading-none text-white"
      aria-label={`${unreadInboxCount} unread`}
    >
      {unreadInboxCount > 99 ? '99+' : unreadInboxCount}
    </span>
  )

  const showRecords =
    ffTranscripts ||
    ffAdvisingIntegration ||
    ffCoCurricularTranscript ||
    ffCompletionCredentials ||
    ffCeuTracking ||
    ffResearchConsent ||
    ffAccessibilityIntake ||
    ffStripeBilling ||
    ffRevenueShare

  return (
    <>
      <SideNavLink to="/" end icon={<LayoutDashboard className="h-5 w-5" />}>
        Dashboard
      </SideNavLink>
      <SideNavLink to="/courses" icon={<BookOpen className="h-5 w-5" />}>
        Courses
      </SideNavLink>
      <SideNavLink to="/calendar" icon={<Calendar className="h-5 w-5" />}>
        Calendar
      </SideNavLink>
      <SideNavLink to="/todos" icon={<ListTodo className="h-5 w-5" />}>
        Todos
      </SideNavLink>

      <SideNavSectionLabel first>Learning</SideNavSectionLabel>
      {ragNotebookEnabled ? (
        <SideNavLink to="/ai" icon={<Bot className="h-5 w-5" />}>
          Ask AI
        </SideNavLink>
      ) : null}
      <SideNavLink to="/review" icon={<RotateCcw className="h-5 w-5" />}>
        Review practice
      </SideNavLink>
      {selfReflectionEnabled ? (
        <SideNavLink to="/me/study-insights" icon={<Sparkles className="h-5 w-5" />}>
          Study insights
        </SideNavLink>
      ) : null}
      {ffLearningPaths ? (
        <SideNavLink to="/my-paths" icon={<Route className="h-5 w-5" />}>
          My learning paths
        </SideNavLink>
      ) : null}
      {ffCatalogIntegration ? (
        <SideNavLink to="/catalog" icon={<GraduationCap className="h-5 w-5" />}>
          Course catalog
        </SideNavLink>
      ) : null}
      {ffCourseMarketplace ? (
        <SideNavLink to="/marketplace" icon={<Store className="h-5 w-5" />}>
          Marketplace
        </SideNavLink>
      ) : null}
      {ffLibrary ? (
        <SideNavLink to="/reading-log" icon={<Library className="h-5 w-5" />}>
          Reading log
        </SideNavLink>
      ) : null}

      <SideNavSectionLabel>Notes & portfolio</SideNavSectionLabel>
      <SideNavLink to="/notebooks" end icon={<BookMarked className="h-5 w-5" />}>
        My Notebooks
      </SideNavLink>
      <SideNavLink to="/notebooks/global" icon={<BookMarked className="h-5 w-5" />}>
        Global notebook
      </SideNavLink>
      {ffEportfolio ? (
        <SideNavLink to="/portfolios" icon={<FolderOpen className="h-5 w-5" />}>
          My Portfolio
        </SideNavLink>
      ) : null}

      {showRecords ? (
        <>
          <SideNavSectionLabel>Records</SideNavSectionLabel>
          {ffTranscripts ? (
            <SideNavLink to="/transcripts" icon={<FileText className="h-5 w-5" />}>
              Transcripts
            </SideNavLink>
          ) : null}
          {ffAdvisingIntegration ? (
            <SideNavLink to="/advising-notes" icon={<GraduationCap className="h-5 w-5" />}>
              Advising notes
            </SideNavLink>
          ) : null}
          {ffCoCurricularTranscript ? (
            <SideNavLink to="/me/ccr" icon={<Award className="h-5 w-5" />}>
              My achievements
            </SideNavLink>
          ) : null}
          {ffCompletionCredentials ? (
            <SideNavLink to="/me/credentials" icon={<Award className="h-5 w-5" />}>
              My credentials
            </SideNavLink>
          ) : null}
          {ffCeuTracking ? (
            <SideNavLink to="/me/ce-transcript" icon={<FileText className="h-5 w-5" />}>
              CE transcript
            </SideNavLink>
          ) : null}
          {ffResearchConsent ? (
            <SideNavLink to="/me/research-studies" icon={<ShieldCheck className="h-5 w-5" />}>
              Research studies
            </SideNavLink>
          ) : null}
          {ffAccessibilityIntake ? (
            <SideNavLink to="/me/accommodations" icon={<ShieldCheck className="h-5 w-5" />}>
              My accommodations
            </SideNavLink>
          ) : null}
          {ffStripeBilling ? (
            <SideNavLink to="/me/billing" icon={<CreditCard className="h-5 w-5" />}>
              Billing
            </SideNavLink>
          ) : null}
          {ffRevenueShare ? (
            <SideNavLink to="/me/creator/earnings" icon={<DollarSign className="h-5 w-5" />}>
              Creator earnings
            </SideNavLink>
          ) : null}
        </>
      ) : null}

      {isParent ? (
        <>
          <SideNavSectionLabel>Family</SideNavSectionLabel>
          <SideNavLink to="/parent" icon={<UsersRound className="h-5 w-5" />}>
            Family dashboard
          </SideNavLink>
          {ffConferenceScheduling ? (
            <SideNavLink to="/parent/conferences" icon={<Calendar className="h-5 w-5" />}>
              Conference booking
            </SideNavLink>
          ) : null}
        </>
      ) : null}

      {(canViewReports || canManageAccommodations) ? (
        <>
          <SideNavSectionLabel>Administration</SideNavSectionLabel>
          {canViewReports ? (
            <SideNavLink to="/reports" icon={<BarChart3 className="h-5 w-5" />}>
              Reports
            </SideNavLink>
          ) : null}
          {canManageAccommodations ? (
            <SideNavLink to="/admin/accommodations" icon={<Accessibility className="h-5 w-5" />}>
              Accommodations
            </SideNavLink>
          ) : null}
          {canManageAccommodations && accommodationsEngineEnabled ? (
            <SideNavLink to="/admin/accommodations/audit" icon={<Accessibility className="h-5 w-5" />}>
              Accommodation audit
            </SideNavLink>
          ) : null}
        </>
      ) : null}

      <SideNavSectionLabel>Account</SideNavSectionLabel>
      <SideNavLink
        to="/inbox"
        data-onboarding="nav-inbox"
        icon={<Inbox className="h-5 w-5" />}
        badge={unreadBadge}
      >
        Inbox
      </SideNavLink>
      <SideNavLink to="/settings" data-onboarding="nav-settings" icon={<Settings className="h-5 w-5" />}>
        Settings
      </SideNavLink>
    </>
  )
}