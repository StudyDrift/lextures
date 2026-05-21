import { authorizedFetch } from './api'
import type { LegalDocumentId } from './legal-documents'

export type PendingLegalDocument = {
  document: LegalDocumentId
  version: string
  effectiveDate: string
}

export async function fetchPendingLegalDocuments(): Promise<PendingLegalDocument[]> {
  const res = await authorizedFetch('/api/v1/legal/pending')
  if (!res.ok) {
    throw new Error(`Failed to load pending legal documents (${res.status})`)
  }
  const data = (await res.json()) as { documents?: PendingLegalDocument[] }
  return data.documents ?? []
}

export async function acknowledgeLegalDocument(
  document: LegalDocumentId,
  version: string,
): Promise<void> {
  const res = await authorizedFetch('/api/v1/legal/acknowledge', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ document, version }),
  })
  if (!res.ok) {
    throw new Error(`Failed to acknowledge legal document (${res.status})`)
  }
}
