import privacyPolicyMd from '../content/legal/privacy-policy.md?raw'
import privacyHistoryMd from '../content/legal/privacy-history.md?raw'
import termsOfServiceMd from '../content/legal/terms-of-service.md?raw'
import termsHistoryMd from '../content/legal/terms-history.md?raw'
import { renderMarkdown, slugifyHeading } from './markdown'
import { LEGAL_VERSIONS, type LegalDocumentId } from './legal-versions'
import { SITE_LINKS } from './site-links'

export { slugifyHeading, LEGAL_VERSIONS, type LegalDocumentId }

export type LegalDocumentConfig = {
  id: LegalDocumentId
  title: string
  path: string
  historyPath: string
  bodyMarkdown: string
  historyMarkdown: string
  bodyHtml: string
  historyHtml: string
  version: string
  effectiveDateLabel: string
  jsonLdType: 'PrivacyPolicy' | 'TermsOfService'
}

export const PRIVACY_POLICY: LegalDocumentConfig = {
  id: 'privacy_policy',
  title: 'Privacy Policy',
  path: SITE_LINKS.privacy,
  historyPath: SITE_LINKS.privacyHistory,
  bodyMarkdown: privacyPolicyMd,
  historyMarkdown: privacyHistoryMd,
  bodyHtml: renderMarkdown(privacyPolicyMd),
  historyHtml: renderMarkdown(privacyHistoryMd),
  version: LEGAL_VERSIONS.privacy_policy.version,
  effectiveDateLabel: LEGAL_VERSIONS.privacy_policy.effectiveDateLabel,
  jsonLdType: 'PrivacyPolicy',
}

export const TERMS_OF_SERVICE: LegalDocumentConfig = {
  id: 'terms_of_service',
  title: 'Terms of Service',
  path: SITE_LINKS.terms,
  historyPath: SITE_LINKS.termsHistory,
  bodyMarkdown: termsOfServiceMd,
  historyMarkdown: termsHistoryMd,
  bodyHtml: renderMarkdown(termsOfServiceMd),
  historyHtml: renderMarkdown(termsHistoryMd),
  version: LEGAL_VERSIONS.terms_of_service.version,
  effectiveDateLabel: LEGAL_VERSIONS.terms_of_service.effectiveDateLabel,
  jsonLdType: 'TermsOfService',
}

/** Extract level-2 headings from markdown for the table of contents. */
export function extractTocEntries(markdown: string): { id: string; title: string }[] {
  const entries: { id: string; title: string }[] = []
  for (const line of markdown.split('\n')) {
    const match = /^##\s+(.+)$/.exec(line.trim())
    if (match) {
      const title = match[1].replace(/\s+/g, ' ').trim()
      entries.push({ id: slugifyHeading(title), title })
    }
  }
  return entries
}
