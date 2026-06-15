import { matchPath } from 'react-router-dom'

export type SettingsNavView =
  | 'ai-models'
  | 'ai-prompts'
  | 'account'
  | 'notifications'
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

export function settingsViewFromPathname(pathname: string): SettingsNavView {
  if (pathname.startsWith('/settings/ai/system-prompts')) return 'ai-prompts'
  if (pathname.startsWith('/settings/ai/models')) return 'ai-models'
  const m = matchPath({ path: '/settings/:tab', end: true }, pathname)
  const raw = m?.params.tab
  if (
    raw === 'account' ||
    raw === 'notifications' ||
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
    raw === 'advising'
  )
    return raw
  return 'account'
}

export type CourseSettingsSection =
  | 'general'
  | 'grading'
  | 'plagiarism'
  | 'outcomes'
  | 'features'
  | 'accessibility'
  | 'translations'
  | 'sections'
  | 'import-export'
  | 'blueprint'
  | 'archive'

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
