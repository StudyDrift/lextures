/**
 * UX.8 — Settings search index (FR-5).
 *
 * Indexes individual settings (label, description, synonyms, scope, section),
 * not only page titles. Built at module load; ≤30 KB gzip target.
 */

import { fuzzyMatches } from '../fuzzy-match'
import { CLASSIFIED_PAGES } from './classification'
import type { SettingsIndexEntry, SettingsScope } from './types'

/** Seed index: page-level entries + high-value control-level entries. */
export const SETTINGS_INDEX: SettingsIndexEntry[] = [
  // Page-level entries (one per classified configuration page)
  ...CLASSIFIED_PAGES.filter((p) => p.kind === 'configuration').map((p) => ({
    key: p.id,
    scope: p.scope,
    route: p.route,
    anchor: null as string | null,
    label: p.label,
    labelKey: p.labelKey,
    description: p.description ?? p.label,
    descriptionKey: p.labelKey,
    synonyms: [] as string[],
    requiredPermission: p.requiredPermission,
    featureFlag: p.featureFlag,
    pageId: p.id,
    section: p.section,
  })),

  // ── Control-level entries (FR-5 deep-link targets) ─────────────────
  {
    key: 'platform.sso.saml',
    scope: 'platform',
    route: '/settings/platform',
    anchor: 'platform.sso.saml',
    label: 'SAML single sign-on',
    labelKey: 'settings.index.saml',
    description: 'Configure SAML 2.0 identity provider for institutional SSO',
    descriptionKey: 'settings.index.samlDesc',
    synonyms: ['SAML', 'SSO', 'single sign-on', 'IdP', 'identity provider', 'federation'],
    requiredPermission: 'rbac.manage',
    pageId: 'settings.platform',
    section: 'platform',
  },
  {
    key: 'platform.sso.oidc',
    scope: 'platform',
    route: '/settings/platform',
    anchor: 'platform.sso.oidc',
    label: 'OIDC single sign-on',
    labelKey: 'settings.index.oidc',
    description: 'OpenID Connect identity provider configuration',
    descriptionKey: 'settings.index.oidcDesc',
    synonyms: ['OIDC', 'OpenID', 'SSO', 'OAuth'],
    requiredPermission: 'rbac.manage',
    pageId: 'settings.platform',
    section: 'platform',
  },
  {
    key: 'platform.retention',
    scope: 'platform',
    route: '/settings/platform',
    anchor: 'platform.retention',
    label: 'Data retention',
    labelKey: 'settings.index.retention',
    description: 'How long platform data is retained before deletion',
    descriptionKey: 'settings.index.retentionDesc',
    synonyms: ['retention', 'data retention', 'purge', 'delete data', 'TTL', 'lifecycle'],
    requiredPermission: 'rbac.manage',
    pageId: 'settings.platform',
    section: 'platform',
  },
  {
    key: 'platform.ff.navigation-v2',
    scope: 'platform',
    route: '/settings/platform',
    anchor: 'platform.ff.navigation-v2',
    label: 'Navigation IA v2',
    labelKey: 'settings.index.navV2',
    description: 'Task-based sidebar taxonomy and primary destinations budget',
    descriptionKey: 'settings.index.navV2Desc',
    synonyms: ['navigation', 'sidebar', 'IA', 'nav v2'],
    requiredPermission: 'rbac.manage',
    pageId: 'settings.platform',
    section: 'platform',
  },
  {
    key: 'platform.ff.settings-ia-v2',
    scope: 'platform',
    route: '/settings/platform',
    anchor: 'platform.ff.settings-ia-v2',
    label: 'Settings IA v2',
    labelKey: 'settings.index.settingsIaV2',
    description: 'Me / Organisation / Platform configuration hierarchy',
    descriptionKey: 'settings.index.settingsIaV2Desc',
    synonyms: ['settings IA', 'settings hierarchy', 'admin settings'],
    requiredPermission: 'rbac.manage',
    pageId: 'settings.platform',
    section: 'platform',
  },
  {
    key: 'account.mfa',
    scope: 'me',
    route: '/settings/account',
    anchor: 'account.mfa',
    label: 'Multi-factor authentication',
    labelKey: 'settings.index.mfa',
    description: 'TOTP and passkey factors for your account',
    descriptionKey: 'settings.index.mfaDesc',
    synonyms: ['MFA', '2FA', 'two-factor', 'authenticator', 'passkey', 'WebAuthn', 'TOTP'],
    requiredPermission: null,
    pageId: 'settings.account',
    section: 'user-settings',
  },
  {
    key: 'account.password',
    scope: 'me',
    route: '/settings/account',
    anchor: 'account.password',
    label: 'Password',
    labelKey: 'settings.index.password',
    description: 'Change your account password',
    descriptionKey: 'settings.index.passwordDesc',
    synonyms: ['password', 'credentials', 'login'],
    requiredPermission: null,
    pageId: 'settings.account',
    section: 'user-settings',
  },
  {
    key: 'account.sessions',
    scope: 'me',
    route: '/settings/account',
    anchor: 'account.sessions',
    label: 'Active sessions',
    labelKey: 'settings.index.sessions',
    description: 'View and revoke signed-in sessions',
    descriptionKey: 'settings.index.sessionsDesc',
    synonyms: ['sessions', 'devices', 'signed in', 'logout other'],
    requiredPermission: null,
    pageId: 'settings.account',
    section: 'user-settings',
  },
  {
    key: 'account.locale',
    scope: 'me',
    route: '/settings/account',
    anchor: 'account.locale',
    label: 'Language and locale',
    labelKey: 'settings.index.locale',
    description: 'Display language, date and number formats',
    descriptionKey: 'settings.index.localeDesc',
    synonyms: ['language', 'locale', 'i18n', 'timezone', 'date format'],
    requiredPermission: null,
    pageId: 'settings.account',
    section: 'user-settings',
  },
  {
    key: 'notifications.email',
    scope: 'me',
    route: '/settings/notifications',
    anchor: 'notifications.email',
    label: 'Email notifications',
    labelKey: 'settings.index.emailNotif',
    description: 'Which events send email to you',
    descriptionKey: 'settings.index.emailNotifDesc',
    synonyms: ['email', 'notifications', 'digest', 'alerts'],
    requiredPermission: null,
    pageId: 'settings.notifications',
    section: 'user-settings',
  },
  {
    key: 'org.scim',
    scope: 'org',
    route: '/settings/scim-provisioning',
    anchor: 'org.scim',
    label: 'SCIM provisioning',
    labelKey: 'settings.index.scim',
    description: 'Automated user provisioning via SCIM 2.0',
    descriptionKey: 'settings.index.scimDesc',
    synonyms: ['SCIM', 'provisioning', 'directory sync', 'Okta', 'Azure AD'],
    requiredPermission: 'rbac.manage',
    featureFlag: 'scimEnabled',
    pageId: 'settings.scim',
    section: 'org-settings',
  },
  {
    key: 'org.branding',
    scope: 'org',
    route: '/settings/org-branding',
    anchor: 'org.branding',
    label: 'Organisation branding',
    labelKey: 'settings.index.branding',
    description: 'Logo, colours, and custom domain',
    descriptionKey: 'settings.index.brandingDesc',
    synonyms: ['branding', 'logo', 'theme', 'colours', 'colors', 'domain'],
    requiredPermission: 'tenant.org_units.admin',
    pageId: 'settings.org-branding',
    section: 'org-settings',
  },
  {
    key: 'org.roles',
    scope: 'org',
    route: '/settings/roles',
    anchor: 'org.roles',
    label: 'Roles and permissions',
    labelKey: 'settings.index.roles',
    description: 'RBAC roles and permission grants',
    descriptionKey: 'settings.index.rolesDesc',
    synonyms: ['roles', 'permissions', 'RBAC', 'access control'],
    requiredPermission: 'rbac.manage',
    pageId: 'settings.roles',
    section: 'org-settings',
  },
  {
    key: 'ai.models.default',
    scope: 'platform',
    route: '/settings/ai/models',
    anchor: 'ai.models.default',
    label: 'Default AI models',
    labelKey: 'settings.index.aiModels',
    description: 'Default text and image models for the platform',
    descriptionKey: 'settings.index.aiModelsDesc',
    synonyms: ['AI models', 'LLM', 'GPT', 'model picker', 'OpenAI', 'Anthropic'],
    requiredPermission: 'rbac.manage',
    pageId: 'settings.ai-models',
    section: 'platform',
  },
  {
    key: 'content-filter.policy',
    scope: 'org',
    route: '/settings/org/content-filter',
    anchor: 'content-filter.policy',
    label: 'Content filter policy',
    labelKey: 'settings.index.contentFilter',
    description: 'Web content filtering rules for learners',
    descriptionKey: 'settings.index.contentFilterDesc',
    synonyms: ['content filter', 'web filter', 'safe search', 'blocklist'],
    requiredPermission: 'rbac.manage',
    featureFlag: 'ffContentFilterIntegration',
    pageId: 'settings.content-filter',
    section: 'org-settings',
  },
  {
    key: 'sis.connection',
    scope: 'org',
    route: '/settings/org/sis',
    anchor: 'sis.connection',
    label: 'SIS connection',
    labelKey: 'settings.index.sis',
    description: 'Student information system integration settings',
    descriptionKey: 'settings.index.sisDesc',
    synonyms: ['SIS', 'student information', 'OneRoster', 'PowerSchool', 'Banner'],
    requiredPermission: 'rbac.manage',
    featureFlag: 'ffSisIntegration',
    pageId: 'settings.sis',
    section: 'org-settings',
  },
  {
    key: 'bookstore.provider',
    scope: 'org',
    route: '/settings/org/bookstore',
    anchor: 'bookstore.provider',
    label: 'Bookstore provider',
    labelKey: 'settings.index.bookstore',
    description: 'VitalSource / RedShelf textbook integration',
    descriptionKey: 'settings.index.bookstoreDesc',
    synonyms: ['bookstore', 'VitalSource', 'RedShelf', 'textbooks', 'inclusive access'],
    requiredPermission: 'rbac.manage',
    featureFlag: 'ffBookstoreIntegration',
    pageId: 'settings.bookstore',
    section: 'org-settings',
  },
  {
    key: 'accessibility.intake',
    scope: 'org',
    route: '/settings/org/accessibility',
    anchor: 'accessibility.intake',
    label: 'Accessibility services intake',
    labelKey: 'settings.index.a11yIntake',
    description: 'Disability services accommodation request workflow',
    descriptionKey: 'settings.index.a11yIntakeDesc',
    synonyms: ['accessibility', 'accommodations', 'disability services', 'IEP', '504', 'intake'],
    requiredPermission: 'rbac.manage',
    featureFlag: 'ffAccessibilityIntake',
    pageId: 'settings.accessibility',
    section: 'org-settings',
  },
]

const byKey = new Map(SETTINGS_INDEX.map((e) => [e.key, e]))

export function settingsIndexEntry(key: string): SettingsIndexEntry | undefined {
  return byKey.get(key)
}

export function allSettingsIndexKeys(): string[] {
  return SETTINGS_INDEX.map((e) => e.key)
}

export type SettingsSearchContext = {
  allows: (permission: string) => boolean
  /** Platform feature flags / derived booleans. */
  platform: Record<string, unknown>
  /** When true, include course-scoped index entries (default false for global search). */
  includeCourse?: boolean
}

function flagVisible(entry: SettingsIndexEntry, platform: Record<string, unknown>): boolean {
  if (!entry.featureFlag) return true
  const v = platform[entry.featureFlag]
  // scimEnabled and similar may be loaded separately; treat missing as hidden when gated
  if (entry.featureFlag === 'xapiEmissionEnabled') {
    return platform.xapiEmissionEnabled === true || platform.ffXapiEmission === true
  }
  if (entry.featureFlag === 'oerLibraryEnabled') {
    return platform.oerLibraryEnabled === true
  }
  return v === true
}

function permissionVisible(entry: SettingsIndexEntry, allows: (p: string) => boolean): boolean {
  if (!entry.requiredPermission) return true
  return allows(entry.requiredPermission)
}

/** Filter index to entries the caller may access (AC-6). */
export function filterSettingsIndex(
  ctx: SettingsSearchContext,
  entries: SettingsIndexEntry[] = SETTINGS_INDEX,
): SettingsIndexEntry[] {
  return entries.filter((e) => {
    if (e.scope === 'course' && !ctx.includeCourse) return false
    if (!permissionVisible(e, ctx.allows)) return false
    if (!flagVisible(e, ctx.platform)) return false
    return true
  })
}

export type SettingsSearchHit = SettingsIndexEntry & {
  score: number
}

function haystack(entry: SettingsIndexEntry): string {
  return [entry.label, entry.description, ...entry.synonyms, entry.key, entry.scope]
    .join(' ')
    .toLowerCase()
}

/**
 * Search settings index. Results ≤50 ms on client for typical index size.
 * Matches label, description, synonyms, key (FR-5).
 */
export function searchSettingsIndex(
  query: string,
  ctx: SettingsSearchContext,
  limit = 20,
): SettingsSearchHit[] {
  const q = query.trim().toLowerCase()
  if (!q) return []
  const visible = filterSettingsIndex(ctx)
  const hits: SettingsSearchHit[] = []
  for (const entry of visible) {
    const hay = haystack(entry)
    if (!fuzzyMatches(q, hay) && !hay.includes(q)) {
      // exact synonym word match
      const synonymHit = entry.synonyms.some((s) => s.toLowerCase().includes(q) || q.includes(s.toLowerCase()))
      if (!synonymHit && !entry.label.toLowerCase().includes(q)) continue
    }
    let score = 0
    if (entry.label.toLowerCase() === q) score += 100
    else if (entry.label.toLowerCase().startsWith(q)) score += 80
    else if (entry.label.toLowerCase().includes(q)) score += 60
    if (entry.synonyms.some((s) => s.toLowerCase() === q)) score += 90
    else if (entry.synonyms.some((s) => s.toLowerCase().includes(q))) score += 50
    if (entry.key.includes(q)) score += 40
    if (entry.description.toLowerCase().includes(q)) score += 20
    if (score === 0) score = 10
    hits.push({ ...entry, score })
  }
  hits.sort((a, b) => b.score - a.score || a.label.localeCompare(b.label))
  return hits.slice(0, limit)
}

/** Deep-link path for a hit (route + optional focus query). */
export function settingsHitPath(hit: SettingsIndexEntry): string {
  if (!hit.anchor) return hit.route
  const sep = hit.route.includes('?') ? '&' : '?'
  return `${hit.route}${sep}focus=${encodeURIComponent(hit.anchor)}`
}

export function settingsByScope(scope: SettingsScope): SettingsIndexEntry[] {
  return SETTINGS_INDEX.filter((e) => e.scope === scope)
}

export function assertSettingsIndexIntegrity(): string[] {
  const errors: string[] = []
  const keys = new Set<string>()
  const pageIds = new Set(CLASSIFIED_PAGES.map((p) => p.id))
  for (const e of SETTINGS_INDEX) {
    if (keys.has(e.key)) errors.push(`duplicate index key: ${e.key}`)
    keys.add(e.key)
    if (!pageIds.has(e.pageId)) errors.push(`index ${e.key} references missing page ${e.pageId}`)
    if (!e.label.trim()) errors.push(`index ${e.key} missing label`)
  }
  return errors
}
