import { lazy, Suspense, useEffect, useId, useMemo, useRef, useState } from 'react'

import { ChevronDown, LogOut, Menu, User } from 'lucide-react'
import { matchPath, useLocation, useNavigate } from 'react-router-dom'
const AiTutorMenu = lazy(() => import('../tutor-panel').then((m) => ({ default: m.AiTutorMenu })))
const FeedbackWidgetMenu = lazy(() =>
  import('../feedback/feedback-widget').then((m) => ({ default: m.FeedbackWidgetMenu })),
)
import { useCourseNavFeatures } from '../../context/course-nav-features-context'
import { setCourseViewAs, useCourseViewAs } from '../../lib/course-view-as'
import { apiUrl, authorizedFetch } from '../../lib/api'
import { getJwtSubject } from '../../lib/auth'
import { useViewerEnrollmentRoles } from '../../lib/use-viewer-enrollment-roles'
import { EnrollmentAvatar } from '../enrollment/enrollment-avatar'

import { clearSessionTokens, getRefreshToken } from '../../lib/session-tokens'
import { applyUiTheme } from '../../lib/ui-theme'
import {
  parseAccountProfile,
  profileName,
  type TopBarAccountProfile,
} from './top-bar-utils'
import { useShellNav } from './use-shell-nav'
import { TopBarBreadcrumbs } from './top-bar-breadcrumbs'
import { CanvasImportHeaderWidget } from '../../context/canvas-import-context'
import { HelpWidgetMenu } from './help-widget'
import { NotificationsDrawer, NotificationsDrawerTrigger } from './notifications-drawer'
import { TopBarMobileCommandPaletteButton } from './side-nav-command-palette'
import { ReadingPreferencesPanel } from '../a11y/ReadingPreferencesPanel'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { Menu as UiMenu } from '../ui/menu'

function UserMenu() {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [profile, setProfile] = useState<TopBarAccountProfile | null>(null)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()

  useEffect(() => {
    let cancelled = false
    async function loadProfile() {
      try {
        const res = await authorizedFetch('/api/v1/settings/account')
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok || cancelled) return
        setProfile(parseAccountProfile(raw))
      } catch {
        if (!cancelled) setProfile(null)
      }
    }
    void loadProfile()
    function onProfileUpdated() {
      void loadProfile()
    }
    window.addEventListener('studydrift-profile-updated', onProfileUpdated)
    return () => {
      cancelled = true
      window.removeEventListener('studydrift-profile-updated', onProfileUpdated)
    }
  }, [])

  async function signOut() {
    const rt = getRefreshToken()
    if (rt) {
      try {
        await fetch(apiUrl('/api/v1/auth/logout'), {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: rt }),
        })
      } catch {
        /* ignore network errors — still clear local session */
      }
    }
    clearSessionTokens()
    applyUiTheme('light')
    navigate('/login', { replace: true })
  }

  const name = profileName(profile)
  const viewerId = getJwtSubject() ?? profile?.email ?? 'viewer'

  return (
    <div className="relative">
      <button
        ref={triggerRef}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label="User menu"
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-2 rounded-full border border-border-default bg-surface-raised py-1.5 ps-1.5 pe-2.5 text-sm font-medium text-fg-muted shadow-sm transition-[background-color,color,border-color] hover:border-border-strong hover:bg-surface-base focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default dark:hover:border-neutral-500 dark:hover:bg-neutral-700"
      >
        <EnrollmentAvatar
          userId={viewerId}
          name={name}
          avatarUrl={profile?.avatarUrl}
          showPreview={false}
        />
        <span className="hidden max-w-[10rem] truncate sm:inline">{name}</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-fg-muted transition-transform dark:text-fg-muted ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      <UiMenu
        open={open}
        onOpenChange={setOpen}
        id={menuId}
        anchorRef={triggerRef}
        placement="bottom-end"
        aria-label="Account"
        items={[
          {
            id: 'profile',
            textValue: 'Profile',
            label: (
              <span className="flex items-center gap-2">
                <User className="h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
                Profile
              </span>
            ),
            onSelect: () => navigate('/settings/account'),
          },
          {
            id: 'sign-out',
            textValue: 'Sign out',
            label: (
              <span className="flex items-center gap-2">
                <LogOut className="h-4 w-4 shrink-0 text-fg-muted" aria-hidden />
                Sign out
              </span>
            ),
            onSelect: () => {
              void signOut()
            },
          },
        ]}
      />
    </div>
  )
}

function CourseEnrollmentViewDropdown() {
  const location = useLocation()
  const courseCode = useMemo(() => {
    const m = matchPath({ path: '/courses/:courseCode', end: false }, location.pathname)
    const code = m?.params.courseCode
    return code && code !== 'create' ? code : null
  }, [location.pathname])

  const courseViewMode = useCourseViewAs(courseCode ?? undefined)

  const viewerRoles = useViewerEnrollmentRoles(courseCode)
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuId = useId()

  const hasTeacher = viewerRoles?.includes('teacher') ?? false
  const hasStudent = viewerRoles?.includes('student') ?? false
  const show = Boolean(courseCode && hasTeacher && hasStudent)

  if (!show || !courseCode) return null

  const label = courseViewMode === 'student' ? 'Student' : 'Teacher'

  return (
    <div className="relative shrink-0 text-start">
      <button
        ref={triggerRef}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={`View course as ${label}. Open menu to switch between teacher and student preview.`}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex max-w-full items-center gap-1.5 rounded-xl bg-accent-solid px-2 py-1.5 text-xs font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-surface-raised dark:focus-visible:ring-neutral-400/40 md:gap-2 md:px-3 md:py-2 md:text-sm"
      >
        <span className="max-md:sr-only">View as: </span>
        {label}
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      <UiMenu
        open={open}
        onOpenChange={setOpen}
        id={menuId}
        anchorRef={triggerRef}
        placement="bottom-end"
        aria-label="View course as"
        items={[
          {
            id: 'teacher',
            textValue: 'Teacher',
            label: (
              <span className="flex flex-col gap-0.5">
                <span className="font-semibold text-fg-default">Teacher</span>
                <span className="text-xs font-normal text-fg-muted">
                  Manage course content, gradebook, and settings
                </span>
              </span>
            ),
            onSelect: () => setCourseViewAs(courseCode, 'teacher'),
          },
          {
            id: 'student',
            textValue: 'Student',
            label: (
              <span className="flex flex-col gap-0.5">
                <span className="font-semibold text-fg-default">Student</span>
                <span className="text-xs font-normal text-fg-muted">
                  Preview the course as a learner would see it
                </span>
              </span>
            ),
            onSelect: () => setCourseViewAs(courseCode, 'student'),
          },
        ]}
      />
    </div>
  )
}

export function TopBar() {
  const location = useLocation()
  const { mobileNavOpen, setMobileNavOpen } = useShellNav()
  const [notificationsOpen, setNotificationsOpen] = useState(false)
  const [readingPanelOpen, setReadingPanelOpen] = useState(false)
  const { ffReadingPreferences } = usePlatformFeatures()
  const { aiTutorEnabled } = useCourseNavFeatures()
  const courseCode = useMemo(() => {
    const m = matchPath({ path: '/courses/:courseCode', end: false }, location.pathname)
    const code = m?.params.courseCode
    return code && code !== 'create' ? code : null
  }, [location.pathname])

  return (
    <header className="lms-chrome flex h-14 shrink-0 items-center gap-1.5 border-b border-border-default bg-surface-raised px-2 shadow-sm shadow-slate-900/5 print:hidden sm:gap-3 sm:px-4 md:gap-4 md:px-6 dark:border-border-default dark:bg-surface-raised dark:shadow-black/20">
      <button
        type="button"
        className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-sunken focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 md:hidden dark:text-fg-muted dark:hover:bg-surface-overlay"
        aria-label={mobileNavOpen ? 'Close navigation menu' : 'Open navigation menu'}
        aria-expanded={mobileNavOpen}
        aria-controls="shell-nav"
        onClick={() => setMobileNavOpen((o) => !o)}
      >
        <Menu className="h-5 w-5" aria-hidden />
      </button>
      <div className="flex min-w-0 flex-1 items-center gap-2 md:gap-3">
        <TopBarBreadcrumbs />
      </div>
      <TopBarMobileCommandPaletteButton />
      <div className="ms-auto flex shrink-0 items-center gap-1.5 sm:gap-3">
        {ffReadingPreferences && (
          <button
            type="button"
            aria-label="Open Reading Preferences"
            aria-expanded={readingPanelOpen}
            aria-haspopup="dialog"
            onClick={() => setReadingPanelOpen((o) => !o)}
            data-testid="reading-preferences-trigger"
            className={`inline-flex h-9 w-9 items-center justify-center rounded-xl text-sm font-semibold transition-[background-color,color,border-color] focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 ${ readingPanelOpen ? 'bg-indigo-100 text-accent-fg dark:bg-indigo-900/40 dark:text-indigo-300' : 'text-fg-muted hover:bg-surface-sunken dark:text-fg-muted dark:hover:bg-surface-overlay' }`}
          >
            Aa
          </button>
        )}
        {courseCode && aiTutorEnabled ? (
          <Suspense fallback={null}>
            <AiTutorMenu courseCode={courseCode} />
          </Suspense>
        ) : null}
        <CanvasImportHeaderWidget />
        <Suspense fallback={null}>
          <FeedbackWidgetMenu />
        </Suspense>
        <HelpWidgetMenu />
        <NotificationsDrawerTrigger open={notificationsOpen} onOpen={() => setNotificationsOpen(true)} />
        <CourseEnrollmentViewDropdown />
        <UserMenu />
      </div>
      <NotificationsDrawer open={notificationsOpen} onClose={() => setNotificationsOpen(false)} />
      {ffReadingPreferences && (
        <ReadingPreferencesPanel open={readingPanelOpen} onClose={() => setReadingPanelOpen(false)} />
      )}
    </header>
  )
}
