/** Legal document versions — no markdown imports (keeps interactive chunks lean). */
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
