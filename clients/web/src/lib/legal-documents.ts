import { MARKETING_LEGAL_URLS } from './marketing-site'

/** Keep in sync with server/internal/httpserver/legal_http.go currentLegalVersions. */
export const LEGAL_VERSIONS = {
  privacy_policy: {
    version: '2026-05-21',
    effectiveDate: '2026-05-21',
    effectiveDateLabel: 'May 21, 2026',
  },
  terms_of_service: {
    version: '2026-05-21',
    effectiveDate: '2026-05-21',
    effectiveDateLabel: 'May 21, 2026',
  },
} as const

export type LegalDocumentId = keyof typeof LEGAL_VERSIONS

export type LegalDocumentMeta = {
  id: LegalDocumentId
  title: string
  url: string
  version: string
  effectiveDateLabel: string
}

export const PRIVACY_POLICY: LegalDocumentMeta = {
  id: 'privacy_policy',
  title: 'Privacy Policy',
  url: MARKETING_LEGAL_URLS.privacy,
  version: LEGAL_VERSIONS.privacy_policy.version,
  effectiveDateLabel: LEGAL_VERSIONS.privacy_policy.effectiveDateLabel,
}

export const TERMS_OF_SERVICE: LegalDocumentMeta = {
  id: 'terms_of_service',
  title: 'Terms of Service',
  url: MARKETING_LEGAL_URLS.terms,
  version: LEGAL_VERSIONS.terms_of_service.version,
  effectiveDateLabel: LEGAL_VERSIONS.terms_of_service.effectiveDateLabel,
}
