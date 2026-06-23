import { matchPath } from 'react-router-dom'

export type SettingsNavView =
  | 'ai-models'
  | 'ai-prompts'
  | 'ai-reports'
  | 'account'
  | 'notifications'
  | 'integrations'
  | 'roles'
  | 'lti-tools'
  | 'platform'
  | 'organizations'
  | 'org-units'
  | 'terms'
  | 'org-branding'
  | 'scim-provisioning'
  | 'cloud-providers'
  | 'lrs-integrations'
  | 'oer-providers'
  | 'transcripts'
  | 'advising'
  | 'archive'

export function settingsViewFromPathname(pathname: string): SettingsNavView {
  if (pathname.startsWith('/settings/ai/system-prompts')) return 'ai-prompts'
  if (pathname.startsWith('/settings/ai/reports')) return 'ai-reports'
  if (pathname.startsWith('/settings/ai/models')) return 'ai-models'
  const m = matchPath({ path: '/settings/:tab', end: true }, pathname)
  const raw = m?.params.tab
  if (
    raw === 'account' ||
    raw === 'notifications' ||
    raw === 'integrations' ||
    raw === 'roles' ||
    raw === 'lti-tools' ||
    raw === 'platform' ||
    raw === 'organizations' ||
    raw === 'org-units' ||
    raw === 'terms' ||
    raw === 'org-branding' ||
    raw === 'scim-provisioning' ||
    raw === 'cloud-providers' ||
    raw === 'lrs-integrations' ||
    raw === 'oer-providers' ||
    raw === 'transcripts' ||
    raw === 'advising' ||
    raw === 'archive'
  )
    return raw
  return 'account'
}

export type CourseSettingsSection =
  | 'general'
  | 'grading'
  | 'grading-agents'
  | 'plagiarism'
  | 'outcomes'
  | 'features'
  | 'accessibility'
  | 'translations'
  | 'sections'
  | 'import-export'
  | 'blueprint'
  | 'archive'

/** Routes that use the settings sidebar (user + system admin tools). */
export function isSettingsShellRoute(pathname: string): boolean {
  if (pathname.startsWith('/settings')) return true
  if (pathname === '/privacy-centre') return true
  if (pathname === '/conferences/availability') return true
  if (pathname.startsWith('/creator/learning-paths')) return true
  if (pathname.startsWith('/library/')) return true

  const adminSettingsRoutes = [
    '/admin/bookstore',
    '/admin/consent-studies',
    '/admin/consortium',
    '/admin/accessibility',
    '/admin/ccr/',
    '/admin/quarantine',
    '/admin/compliance/',
    '/admin/caption-compliance',
    '/admin/attendance/',
    '/admin/behavior/',
    '/admin/broadcasts/',
    '/admin/conferences/',
    '/admin/demographics/',
    '/admin/content-filter',
    '/admin/sis',
    '/admin/final-grades/',
    '/admin/incompletes',
    '/admin/academic-calendar',
    '/admin/evaluations/',
  ]
  return adminSettingsRoutes.some(
    (prefix) => pathname === prefix || pathname.startsWith(prefix),
  )
}

export function courseSettingsSectionFromPathname(pathname: string): CourseSettingsSection {
  const m = matchPath({ path: '/courses/:courseCode/settings/*', end: true }, pathname)
  const raw = m?.params['*']?.replace(/^\/+/, '') ?? ''
  const parts = raw.split('/').filter(Boolean)
  if (parts.length > 1) return 'general'
  const seg = parts[0] ?? ''
  if (
    seg === '' ||
    seg === 'general' ||
    seg === 'dates' ||
    seg === 'branding' ||
    seg === 'basic'
  ) {
    return 'general'
  }
  if (seg === 'grading') return 'grading'
  if (seg === 'grading-agents') return 'grading-agents'
  if (seg === 'plagiarism') return 'plagiarism'
  if (seg === 'outcomes') return 'outcomes'
  if (seg === 'features' || seg === 'features-tools') return 'features'
  if (seg === 'accessibility') return 'accessibility'
  if (seg === 'translations') return 'translations'
  if (seg === 'sections') return 'sections'
  if (seg === 'import-export' || seg === 'export-import') return 'import-export'
  if (seg === 'blueprint') return 'blueprint'
  if (seg === 'archive' || seg === 'archived') return 'archive'
  return 'general'
}
