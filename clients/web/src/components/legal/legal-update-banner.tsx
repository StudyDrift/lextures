import { useEffect, useState } from 'react'
import { acknowledgeLegalDocument, fetchPendingLegalDocuments, type PendingLegalDocument } from '../../lib/legal-api'
import { PRIVACY_POLICY, TERMS_OF_SERVICE } from '../../lib/legal-documents'

const DOC_LINKS: Record<string, { label: string; href: string }> = {
  privacy_policy: { label: PRIVACY_POLICY.title, href: PRIVACY_POLICY.url },
  terms_of_service: { label: TERMS_OF_SERVICE.title, href: TERMS_OF_SERVICE.url },
}

export function LegalUpdateBanner() {
  const [pending, setPending] = useState<PendingLegalDocument[]>([])
  const [dismissing, setDismissing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let active = true
    fetchPendingLegalDocuments()
      .then(docs => { if (active) { setPending(docs); setError(null) } })
      .catch(() => { if (active) setPending([]) })
    return () => { active = false }
  }, [])

  if (pending.length === 0) {
    return null
  }

  const primary = pending[0]
  const link = DOC_LINKS[primary.document]

  async function handleAcknowledge() {
    setDismissing(true)
    setError(null)
    try {
      for (const doc of pending) {
        await acknowledgeLegalDocument(doc.document, doc.version)
      }
      setPending([])
    } catch {
      setError('Could not save your acknowledgement. Please try again.')
    } finally {
      setDismissing(false)
    }
  }

  return (
    <div
      role="region"
      aria-label="Legal policy update"
      className="border-b border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/40 dark:text-amber-100"
    >
      <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
        <p>
          <strong>Privacy Policy has been updated.</strong>{' '}
          {link ? (
            <>
              Please review the{' '}
              <a href={link.href} className="font-medium underline underline-offset-2">
                {link.label}
              </a>
              {pending.length > 1 ? ' and other updated policies' : ''} (effective{' '}
              {primary.effectiveDate}).
            </>
          ) : (
            <>Please review the updated legal documents (effective {primary.effectiveDate}).</>
          )}
        </p>
        <div className="flex items-center gap-2">
          {error ? <span className="text-red-700 dark:text-red-300">{error}</span> : null}
          <button
            type="button"
            onClick={() => void handleAcknowledge()}
            disabled={dismissing}
            className="rounded-md bg-amber-800 px-3 py-1.5 text-sm font-medium text-white transition hover:bg-amber-900 disabled:opacity-60 dark:bg-amber-700 dark:hover:bg-amber-600"
          >
            I acknowledge
          </button>
        </div>
      </div>
    </div>
  )
}
