import { lazy, Suspense, useEffect, useId, useMemo, useRef, useState } from 'react'

import { ChevronDown, LogOut, Menu, User } from 'lucide-react'
import { Link, matchPath, useLocation, useNavigate } from 'react-router-dom'
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

function UserMenu() {
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const [profile, setProfile] = useState<TopBarAccountProfile | null>(null)
  const rootRef = useRef<HTMLDivElement>(null)
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

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  async function signOut() {
    setOpen(false)
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
    <div ref={rootRef} className="relative">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label="User menu"
        onClick={() => setOpen((o) => !o)}
        className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-white py-1.5 ps-1.5 pe-2.5 text-sm font-medium text-slate-700 shadow-sm transition-[background-color,color,border-color] hover:border-slate-300 hover:bg-slate-50 focus:border-indigo-300 focus:outline-none focus:ring-2 focus:ring-indigo-500/20 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:border-neutral-500 dark:hover:bg-neutral-700"
      >
        <EnrollmentAvatar
          userId={viewerId}
          name={name}
          avatarUrl={profile?.avatarUrl}
          showPreview={false}
        />
        <span className="hidden max-w-[10rem] truncate sm:inline">{name}</span>
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-slate-500 transition-transform dark:text-neutral-400 ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="Account"
          className="absolute end-0 z-50 mt-1 min-w-[11rem] overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
        >
          <div className="border-b border-slate-100 px-3 py-2 dark:border-neutral-700">
            <p className="truncate text-sm font-medium text-slate-800 dark:text-neutral-100">{name}</p>
            {profile?.email && (
              <p className="truncate text-xs text-slate-500 dark:text-neutral-400">{profile.email}</p>
            )}
          </div>
          <Link
            to="/settings/account"
            role="menuitem"
            onClick={() => setOpen(false)}
            className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:text-neutral-200 dark:hover:bg-neutral-700"
          >
            <User className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
            Profile
          </Link>
          <button
            type="button"
            role="menuitem"
            onClick={signOut}
            className="flex w-full items-center gap-2 border-t border-slate-100 px-2.5 py-2 text-start text-sm text-slate-700 transition-[background-color,color,border-color] hover:bg-slate-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-700"
          >
            <LogOut className="h-4 w-4 shrink-0 text-slate-500 dark:text-neutral-400" aria-hidden />
            Sign out
          </button>
        </div>
      )}
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
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  const hasTeacher = viewerRoles?.includes('teacher') ?? false
  const hasStudent = viewerRoles?.includes('student') ?? false
  const show = Boolean(courseCode && hasTeacher && hasStudent)

  if (!show || !courseCode) return null

  const label = courseViewMode === 'student' ? 'Student' : 'Teacher'

  return (
    <div ref={rootRef} className="relative shrink-0 text-start">
      <button
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        aria-label={`View course as ${label}. Open menu to switch between teacher and student preview.`}
        onClick={() => setOpen((o) => !o)}
        className="inline-flex max-w-full items-center gap-1.5 rounded-xl bg-indigo-600 px-2 py-1.5 text-xs font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white dark:focus-visible:ring-neutral-400/40 md:gap-2 md:px-3 md:py-2 md:text-sm"
      >
        <span className="max-md:sr-only">View as: </span>
        {label}
        <ChevronDown
          className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>

      {open && (
        <div
          id={menuId}
          role="menu"
          aria-label="View course as"
          className="absolute end-0 z-50 mt-1 min-w-[14rem] overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-800 dark:shadow-black/40"
        >
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              setCourseViewAs(courseCode, 'teacher')
              setOpen(false)
            }}
            className={`flex w-full flex-col gap-0.5 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-slate-50 dark:hover:bg-neutral-700 ${
              courseViewMode === 'teacher' ? 'bg-indigo-50 dark:bg-neutral-800' : ''
            }`}
          >
            <span className="font-semibold text-slate-950 dark:text-neutral-100">Teacher</span>
            <span className="text-xs text-slate-500 dark:text-neutral-400">
              Manage course content, gradebook, and settings
            </span>
          </button>
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              setCourseViewAs(courseCode, 'student')
              setOpen(false)
            }}
            className={`flex w-full flex-col gap-0.5 px-2.5 py-2 text-start text-sm transition-[background-color,color,border-color] hover:bg-slate-50 dark:hover:bg-neutral-700 ${
              courseViewMode === 'student' ? 'bg-indigo-50 dark:bg-neutral-800' : ''
            }`}
          >
            <span className="font-semibold text-slate-950 dark:text-neutral-100">Student</span>
            <span className="text-xs text-slate-500 dark:text-neutral-400">
              Preview the course as a learner would see it
            </span>
          </button>
        </div>
      )}
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
    <header className="lms-chrome flex h-14 shrink-0 items-center gap-1.5 border-b border-slate-200 bg-white px-2 shadow-sm shadow-slate-900/5 print:hidden sm:gap-3 sm:px-4 md:gap-4 md:px-6 dark:border-neutral-700 dark:bg-neutral-900 dark:shadow-black/20">
      <button
        type="button"
        className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl text-slate-600 transition-[background-color,color,border-color] hover:bg-slate-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 md:hidden dark:text-neutral-300 dark:hover:bg-neutral-800"
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
            className={`inline-flex h-9 w-9 items-center justify-center rounded-xl text-sm font-semibold transition-[background-color,color,border-color] focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 ${
              readingPanelOpen
                ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
                : 'text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800'
            }`}
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
