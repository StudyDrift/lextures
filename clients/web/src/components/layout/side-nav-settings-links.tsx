import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import {
  Archive,
  ArrowLeft,
  Bell,
  BookOpen,
  Bot,
  Building2,
  CalendarRange,
  ChevronDown,
  FileText,
  FolderTree,
  GraduationCap,
  Link2,
  Palette,
  Plug,
  Settings2,
  Shield,
  ShieldCheck,
  Store,
  User,
  Users,
  Workflow,
  Lock,
  Mail,
  MessageSquare,
  Sparkles,
} from 'lucide-react'
import { SideNavAdminLinks } from './side-nav-admin-links'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformScimEnabled } from '../../hooks/use-platform-scim-enabled'
import { oerLibraryEnabled } from '../../lib/oer-api'
import { xapiEmissionFeatureEnabled } from '../../lib/platform-features'
import {
  PERM_ACCOMMODATIONS_MANAGE,
  PERM_RBAC_MANAGE,
  PERM_TENANT_ORG_ROLES_MANAGE,
  PERM_TENANT_ORG_ROLES_VIEW,
  PERM_TENANT_ORG_UNITS_ADMIN,
} from '../../lib/rbac-api'
import { settingsViewFromPathname } from './side-nav-path-utils'
import { sideNavActiveClass, sideNavLinkClass } from './side-nav-styles'
import { SideNavLink } from './side-nav-link'
import { useShellNav } from './use-shell-nav'

export function SideNavSettingsLinks() {
  const { allows, loading: permLoading } = usePermissions()
  const { sideNavCollapsed } = useShellNav()
  const canManageRbac = !permLoading && allows(PERM_RBAC_MANAGE)
  const canManageAccommodations = !permLoading && allows(PERM_ACCOMMODATIONS_MANAGE)
  const canOrgUnits = !permLoading && (canManageRbac || allows(PERM_TENANT_ORG_UNITS_ADMIN))
  const canOrgRoles =
    !permLoading && (allows(PERM_TENANT_ORG_ROLES_MANAGE) || allows(PERM_TENANT_ORG_ROLES_VIEW))
  const { scimEnabled: platformScimEnabled } = usePlatformScimEnabled(canManageRbac)
  const {
    ffBookstoreIntegration,
    ffTranscripts,
    ffAdvisingIntegration,
    ffResearchConsent,
    ffAccessibilityIntake,
    ffConsortiumSharing,
    ffCoCurricularTranscript,
    gdprModuleEnabled,
    learnerProfileEnabled,
    ffFeedback,
    emailTemplateEditorEnabled,
  } = usePlatformFeatures()
  const location = useLocation()
  const view = settingsViewFromPathname(location.pathname)
  const aiSectionActive = view === 'ai-models' || view === 'ai-prompts' || view === 'ai-reports'
  const [aiOpen, setAiOpen] = useState(() => location.pathname.startsWith('/settings/ai'))

  useEffect(() => {
    if (location.pathname.startsWith('/settings/ai')) {
      queueMicrotask(() => setAiOpen(true))
    }
  }, [location.pathname])

  return (
    <>
      <SideNavLink to="/" icon={<ArrowLeft className="h-5 w-5" />} end>
        Back
      </SideNavLink>
      {!sideNavCollapsed && (
        <p className="px-3 pb-1 pt-3 text-sm font-bold tracking-tight text-slate-900 dark:text-neutral-100">
          User Settings
        </p>
      )}
      <SideNavLink
        to="/settings/account"
        className={() => (view === 'account' ? sideNavActiveClass : '')}
        icon={<User className="h-5 w-5" />}
      >
        Account
      </SideNavLink>
      <SideNavLink
        to="/settings/integrations"
        className={() => (view === 'integrations' ? sideNavActiveClass : '')}
        icon={<Workflow className="h-5 w-5" />}
      >
        Integrations
      </SideNavLink>
      {learnerProfileEnabled && (
        <SideNavLink
          to="/settings/learner-profile"
          className={() => (view === 'learner-profile' ? sideNavActiveClass : '')}
          icon={<Sparkles className="h-5 w-5" />}
        >
          Learner Profile
        </SideNavLink>
      )}
      <SideNavLink
        to="/settings/notifications"
        className={() => (view === 'notifications' ? sideNavActiveClass : '')}
        icon={<Bell className="h-5 w-5" />}
      >
        Notifications
      </SideNavLink>
      {gdprModuleEnabled && (
        <SideNavLink
          to="/privacy-centre"
          className={() => (location.pathname === '/privacy-centre' ? sideNavActiveClass : '')}
          icon={<Lock className="h-5 w-5" />}
        >
          Privacy Center
        </SideNavLink>
      )}
      {(canOrgUnits || canOrgRoles || canManageRbac) && (
        <>
          {!sideNavCollapsed && (
            <p className="px-3 pb-1 pt-4 text-sm font-bold tracking-tight text-slate-900 dark:text-neutral-100">
              System Settings
            </p>
          )}
          {(canOrgUnits || canOrgRoles) && (
            <SideNavLink
              to="/settings/terms"
              className={() => (view === 'terms' ? sideNavActiveClass : '')}
              icon={<CalendarRange className="h-5 w-5" />}
            >
              Academic terms
            </SideNavLink>
          )}
          {canManageRbac && ffAccessibilityIntake && (
            <SideNavLink
              to="/admin/accessibility"
              className={() =>
                location.pathname === '/admin/accessibility' ? sideNavActiveClass : ''
              }
              icon={<ShieldCheck className="h-5 w-5" />}
            >
              Accessibility services
            </SideNavLink>
          )}
          {canManageRbac && ffAdvisingIntegration && (
            <SideNavLink
              to="/settings/advising"
              className={() => (view === 'advising' ? sideNavActiveClass : '')}
              icon={<GraduationCap className="h-5 w-5" />}
            >
              Advising
            </SideNavLink>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/archive"
              className={() => (view === 'archive' ? sideNavActiveClass : '')}
              icon={<Archive className="h-5 w-5" />}
            >
              Archive
            </SideNavLink>
          )}
          {canManageRbac && ffBookstoreIntegration && (
            <SideNavLink
              to="/admin/bookstore"
              className={() =>
                location.pathname === '/admin/bookstore' ? sideNavActiveClass : ''
              }
              icon={<Store className="h-5 w-5" />}
            >
              Bookstore
            </SideNavLink>
          )}
          {(canOrgUnits || canOrgRoles) && (
            <SideNavLink
              to="/settings/org-branding"
              className={() => (view === 'org-branding' ? sideNavActiveClass : '')}
              icon={<Palette className="h-5 w-5" />}
            >
              Branding
            </SideNavLink>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/cloud-providers"
              className={() => (view === 'cloud-providers' ? sideNavActiveClass : '')}
              icon={<Link2 className="h-5 w-5" />}
            >
              Cloud file pickers
            </SideNavLink>
          )}
          {canManageRbac && ffConsortiumSharing && (
            <SideNavLink
              to="/admin/consortium"
              className={() =>
                location.pathname === '/admin/consortium' ? sideNavActiveClass : ''
              }
              icon={<GraduationCap className="h-5 w-5" />}
            >
              Consortium sharing
            </SideNavLink>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/courses"
              className={() => (view === 'courses' ? sideNavActiveClass : '')}
              icon={<BookOpen className="h-5 w-5" />}
            >
              Courses
            </SideNavLink>
          )}
          {canManageRbac && emailTemplateEditorEnabled && (
            <SideNavLink
              to="/settings/email-templates"
              className={() => (view === 'email-templates' ? sideNavActiveClass : '')}
              icon={<Mail className="h-5 w-5" />}
            >
              Email templates
            </SideNavLink>
          )}
          {canManageRbac && ffFeedback && (
            <SideNavLink
              to="/settings/feedback"
              className={() => (view === 'feedback' ? sideNavActiveClass : '')}
              icon={<MessageSquare className="h-5 w-5" />}
            >
              Feedback
            </SideNavLink>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/platform"
              className={() => (view === 'platform' ? sideNavActiveClass : '')}
              icon={<Settings2 className="h-5 w-5" />}
            >
              Global platform
            </SideNavLink>
          )}
          {canManageRbac && (
            <div className="flex flex-col gap-0.5">
              <button
                type="button"
                onClick={() => setAiOpen((o) => !o)}
                className={`${sideNavLinkClass} ${
                  sideNavCollapsed ? 'justify-center' : ''
                } ${
                  aiOpen || aiSectionActive
                    ? 'text-slate-900 dark:text-neutral-50'
                    : 'text-slate-500 dark:text-neutral-400'
                }`}
                aria-expanded={aiOpen}
                title={sideNavCollapsed ? 'Intelligence' : undefined}
              >
                <span className="flex h-5 w-5 shrink-0 items-center justify-center text-current opacity-90">
                  <Bot className="h-5 w-5" aria-hidden />
                </span>
                {!sideNavCollapsed && (
                  <span className="flex min-w-0 flex-1 items-center justify-between gap-2">
                    <span className="truncate">Intelligence</span>
                    <ChevronDown
                      className={`h-4 w-4 shrink-0 text-current opacity-70 transition-transform duration-200 ease-out ${
                        aiOpen ? 'rotate-180' : 'rotate-0'
                      }`}
                      aria-hidden
                    />
                  </span>
                )}
              </button>
              {!sideNavCollapsed && (
                <div
                  className={`grid transition-[grid-template-rows] duration-200 ease-out ${
                    aiOpen ? 'grid-rows-[1fr]' : 'grid-rows-[0fr]'
                  }`}
                >
                  <div className="min-h-0 overflow-hidden">
                    <div className="flex flex-col gap-0.5 pb-0.5">
                      <SideNavLink to="/settings/ai/models" nested>
                        Models
                      </SideNavLink>
                      <SideNavLink to="/settings/ai/reports" nested>
                        Reports
                      </SideNavLink>
                      <SideNavLink to="/settings/ai/system-prompts" nested>
                        System Prompts
                      </SideNavLink>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/intro-course"
              className={() => (view === 'intro-course' ? sideNavActiveClass : '')}
              icon={<GraduationCap className="h-5 w-5" />}
            >
              Intro course
            </SideNavLink>
          )}
          {canManageRbac && xapiEmissionFeatureEnabled() && (
            <SideNavLink
              to="/settings/lrs-integrations"
              className={() => (view === 'lrs-integrations' ? sideNavActiveClass : '')}
              icon={<Link2 className="h-5 w-5" />}
            >
              Learning Record Stores
            </SideNavLink>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/lti-tools"
              className={() => (view === 'lti-tools' ? sideNavActiveClass : '')}
              icon={<Plug className="h-5 w-5" />}
            >
              LTI tools
            </SideNavLink>
          )}
          {canManageRbac && oerLibraryEnabled() && (
            <SideNavLink
              to="/settings/oer-providers"
              className={() => (view === 'oer-providers' ? sideNavActiveClass : '')}
              icon={<BookOpen className="h-5 w-5" />}
            >
              OER library
            </SideNavLink>
          )}
          {canOrgUnits && (
            <SideNavLink
              to="/settings/org-units"
              className={() => (view === 'org-units' ? sideNavActiveClass : '')}
              icon={<FolderTree className="h-5 w-5" />}
            >
              Org structure
            </SideNavLink>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/organizations"
              className={() => (view === 'organizations' ? sideNavActiveClass : '')}
              icon={<Building2 className="h-5 w-5" />}
            >
              Organizations
            </SideNavLink>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/people"
              className={() => (view === 'people' ? sideNavActiveClass : '')}
              icon={<Users className="h-5 w-5" />}
            >
              People
            </SideNavLink>
          )}
          {canManageRbac && ffResearchConsent && (
            <SideNavLink
              to="/admin/consent-studies"
              className={() =>
                location.pathname === '/admin/consent-studies' ? sideNavActiveClass : ''
              }
              icon={<ShieldCheck className="h-5 w-5" />}
            >
              Research consent
            </SideNavLink>
          )}
          {canManageRbac && (
            <SideNavLink
              to="/settings/roles"
              className={() => (view === 'roles' ? sideNavActiveClass : '')}
              icon={<Shield className="h-5 w-5" />}
            >
              Roles and Permissions
            </SideNavLink>
          )}
          {canManageRbac && platformScimEnabled && (
            <SideNavLink
              to="/settings/scim-provisioning"
              className={() => (view === 'scim-provisioning' ? sideNavActiveClass : '')}
              icon={<Link2 className="h-5 w-5" />}
            >
              SCIM provisioning
            </SideNavLink>
          )}
          {canManageRbac && ffTranscripts && (
            <SideNavLink
              to="/settings/transcripts"
              className={() => (view === 'transcripts' ? sideNavActiveClass : '')}
              icon={<FileText className="h-5 w-5" />}
            >
              Transcripts
            </SideNavLink>
          )}
        </>
      )}
      {(canManageRbac || (canManageAccommodations && ffCoCurricularTranscript)) && (
        <SideNavAdminLinks />
      )}
    </>
  )
}
