import { lazy, Suspense, useEffect } from 'react'
import { Outlet, useLocation } from 'react-router-dom'
import { CommandPaletteProvider } from '../command-palette/command-palette-provider'
import { KeyboardShortcutsProvider } from '../keyboard-shortcuts/keyboard-shortcuts-provider'
import { CourseFeedUnreadProvider } from '../../context/course-feed-unread-provider'
import { CoursePinnedProvider } from '../../context/course-pinned-context'
import { CourseHiddenProvider } from '../../context/course-hidden-context'
import { InboxNotificationsProvider } from '../../context/inbox-notifications-provider'
import { CanvasImportProvider } from '../../context/canvas-import-context'
import { InboxUnreadProvider } from '../../context/inbox-unread-provider'
import { CourseNavFeaturesProvider } from '../../context/course-nav-features-context'
import { ContentFilterProvider } from '../../context/content-filter-context'
import { PlatformFeaturesProvider } from '../../context/platform-features-context'
import { ReadingPreferencesProvider } from '../../context/reading-preferences-context'
import { QuizFocusTopBar } from './quiz-focus-top-bar'
import { ReadingFocusTopBar } from './reading-focus-top-bar'
import { useQuizShellFocus } from './quiz-shell-focus-context'
import { QuizShellFocusProvider } from './quiz-shell-focus-provider'
import { ReadingShellFocusProvider, useReadingShellFocus } from './reading-shell-focus-context'
import { ShellNavProvider } from './shell-nav-context'
import { SideNav } from './side-nav'
import { TopBar } from './top-bar'
import { UiThemeSync } from './ui-theme-sync'
import { LocaleBootstrapSync } from './locale-sync'
import { LmsExperienceRoot } from './lms-experience-root'
import { LegalUpdateBanner } from '../legal/legal-update-banner'

const MaintenanceStatusBanner = lazy(() =>
  import('../StatusBanner').then((m) => ({ default: m.StatusBanner })),
)
const IncidentStatusBanner = lazy(() =>
  import('../incident-status-banner').then((m) => ({ default: m.IncidentStatusBanner })),
)
const ImpersonationChrome = lazy(() =>
  import('../impersonation-chrome').then((m) => ({ default: m.ImpersonationChrome })),
)
import { OfflineBanner } from '../offline-banner'
import { SkipLink } from '../skip-link'
import { useFocusOnRoute } from '../../lib/a11y'
import { ReadingRuler } from '../a11y/ReadingRuler'

function AppShellLayout() {
  const location = useLocation()
  const { focus } = useQuizShellFocus()
  const { readingFocus, setReadingFocus } = useReadingShellFocus()
  const hideChrome = Boolean(focus || readingFocus)
  const shellClassName = `flex h-dvh min-h-0 overflow-hidden bg-slate-50 dark:bg-neutral-950 ${
    focus ? 'ring-2 ring-inset ring-indigo-900/35 dark:ring-amber-400/25' : ''
  }`

  useFocusOnRoute()

  useEffect(() => {
    setReadingFocus(false)
  }, [location.pathname, setReadingFocus])

  return (
    <CourseNavFeaturesProvider>
      <LmsExperienceRoot>
      <UiThemeSync />
      <LocaleBootstrapSync />
      <ReadingRuler />
      <SkipLink />
      <Suspense fallback={null}>
        <ImpersonationChrome shellClassName={shellClassName}>
        {!hideChrome ? <SideNav /> : null}
        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-white dark:bg-neutral-900">
          {focus ? (
            <QuizFocusTopBar model={focus} />
          ) : readingFocus ? (
            <ReadingFocusTopBar />
          ) : (
            <TopBar />
          )}
          <OfflineBanner />
          <Suspense fallback={null}>
            <MaintenanceStatusBanner />
          </Suspense>
          <Suspense fallback={null}>
            <IncidentStatusBanner />
          </Suspense>
          <LegalUpdateBanner />
          <main
            id="main-content"
            tabIndex={-1}
            className="lms-scope lms-print-root flex min-h-0 min-w-0 flex-1 flex-col overflow-x-hidden overflow-y-auto outline-none dark:bg-neutral-900"
          >
            <Outlet />
          </main>
        </div>
        </ImpersonationChrome>
      </Suspense>
      </LmsExperienceRoot>
    </CourseNavFeaturesProvider>
  )
}

export function AppShell() {
  return (
    <PlatformFeaturesProvider>
    <ContentFilterProvider>
    <ReadingPreferencesProvider>
    <InboxUnreadProvider>
      <CanvasImportProvider>
      <CoursePinnedProvider>
      <CourseHiddenProvider>
      <InboxNotificationsProvider>
      <CourseFeedUnreadProvider>
        <CommandPaletteProvider>
          <KeyboardShortcutsProvider>
            <ShellNavProvider>
              <QuizShellFocusProvider>
                <ReadingShellFocusProvider>
                  <AppShellLayout />
                </ReadingShellFocusProvider>
              </QuizShellFocusProvider>
            </ShellNavProvider>
          </KeyboardShortcutsProvider>
        </CommandPaletteProvider>
      </CourseFeedUnreadProvider>
      </InboxNotificationsProvider>
      </CourseHiddenProvider>
      </CoursePinnedProvider>
      </CanvasImportProvider>
    </InboxUnreadProvider>
    </ReadingPreferencesProvider>
    </ContentFilterProvider>
    </PlatformFeaturesProvider>
  )
}
