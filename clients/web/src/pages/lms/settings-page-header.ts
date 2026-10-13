/** Settings shell helpers: which /settings/* paths are admin-only and what header they show. */

export function isSystemSettingsPath(pathname: string): boolean {
  if (pathname.startsWith('/settings/ai/')) return true
  if (pathname.startsWith('/settings/feedback')) return true
  return (
    pathname === '/settings/roles' ||
    pathname === '/settings/lti-tools' ||
    pathname === '/settings/platform' ||
    pathname === '/settings/email-templates' ||
    pathname === '/settings/organizations' ||
    pathname === '/settings/org-units' ||
    pathname === '/settings/terms' ||
    pathname === '/settings/org-branding' ||
    pathname === '/settings/scim-provisioning' ||
    pathname === '/settings/cloud-providers' ||
    pathname === '/settings/lrs-integrations' ||
    pathname === '/settings/oer-providers' ||
    pathname === '/settings/transcripts' ||
    pathname === '/settings/advising' ||
    pathname === '/settings/archive' ||
    pathname === '/settings/people' ||
    pathname === '/settings/courses' ||
    pathname === '/settings/intro-course'
  )
}

/** Page header for /settings/* — admin (system) views get admin copy, not the personal-preferences blurb (#791). */
export function settingsPageHeader(pathname: string): { title: string; description: string } {
  if (isSystemSettingsPath(pathname)) {
    return {
      title: 'System settings',
      description: 'Platform, organization, and integration settings for administrators.',
    }
  }
  return { title: 'Settings', description: 'Account and learning preferences.' }
}
