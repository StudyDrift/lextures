import { useState } from 'react'
import { BookCopy } from 'lucide-react'
import type { InclusiveAccessStatus } from '../../lib/courses-api'

const PROVIDER_LABELS: Record<string, string> = {
  vitalsource: 'VitalSource',
  redshelf: 'RedShelf',
}

type Props = {
  status: InclusiveAccessStatus
  /** Persisted acknowledgement key namespacing (e.g. course code). */
  storageKey?: string
}

/**
 * Inclusive Access opt-out banner (plan 14.11, FR-4 / AC-2).
 *
 * Non-dismissible until the student either confirms they will use the included
 * digital materials or proceeds to the bookstore opt-out form. The opt-out step
 * requires an explicit confirmation to avoid accidental opt-outs (risk mitigation).
 */
export function InclusiveAccessBanner({ status, storageKey }: Props) {
  const ackKey = storageKey ? `ia-ack:${storageKey}` : null
  const [acknowledged, setAcknowledged] = useState<boolean>(() => {
    if (!ackKey) return false
    try {
      return localStorage.getItem(ackKey) === '1'
    } catch {
      return false
    }
  })
  const [confirmingOptOut, setConfirmingOptOut] = useState(false)

  if (!status.enabled || acknowledged) return null

  const providerLabel = PROVIDER_LABELS[status.provider ?? 'vitalsource'] ?? status.provider

  function confirmKeep() {
    if (ackKey) {
      try {
        localStorage.setItem(ackKey, '1')
      } catch {
        /* ignore */
      }
    }
    setAcknowledged(true)
  }

  return (
    <div
      role="alert"
      className="mb-4 rounded-2xl border border-amber-300/80 bg-amber-50 p-4 shadow-sm dark:border-amber-500/40 dark:bg-amber-950/40"
    >
      <div className="flex items-start gap-3">
        <span
          className="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-xl border border-amber-300/90 bg-white text-amber-700 dark:border-amber-500/40 dark:bg-amber-900/40 dark:text-amber-200"
          aria-hidden
        >
          <BookCopy className="h-5 w-5" strokeWidth={2} />
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="text-sm font-semibold text-amber-900 dark:text-amber-100">
            Inclusive Access — {providerLabel}
          </h2>
          <p className="mt-1 text-sm text-amber-800 dark:text-amber-200/90">
            Your required textbook{status.title ? <strong> {status.title}</strong> : null} is included
            in your course charges as a digital edition
            {status.isbn ? (
              <>
                {' '}
                (ISBN <span className="font-mono">{status.isbn}</span>)
              </>
            ) : null}
            . If you already have a copy, you may opt out before the deadline.
          </p>

          {!confirmingOptOut ? (
            <div className="mt-3 flex flex-wrap gap-2">
              <button
                type="button"
                onClick={confirmKeep}
                className="rounded-xl bg-amber-600 px-3.5 py-2 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-amber-500"
              >
                Keep included access
              </button>
              {status.optOutUrl ? (
                <button
                  type="button"
                  onClick={() => setConfirmingOptOut(true)}
                  className="rounded-xl border border-amber-400 bg-white px-3.5 py-2 text-sm font-semibold text-amber-800 transition-[background-color,color,border-color] hover:bg-amber-100 dark:border-amber-500/50 dark:bg-amber-900/30 dark:text-amber-100"
                >
                  Opt out…
                </button>
              ) : null}
            </div>
          ) : (
            <div className="mt-3 rounded-xl border border-amber-300/80 bg-white/70 p-3 dark:border-amber-500/40 dark:bg-amber-900/30">
              <p className="text-sm text-amber-900 dark:text-amber-100">
                Opting out removes your digital access and you will be responsible for obtaining the
                materials yourself. Continue to the bookstore opt-out form?
              </p>
              <div className="mt-3 flex flex-wrap gap-2">
                <a
                  href={status.optOutUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  onClick={confirmKeep}
                  className="rounded-xl bg-rose-600 px-3.5 py-2 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-rose-500"
                >
                  Continue to opt-out form
                </a>
                <button
                  type="button"
                  onClick={() => setConfirmingOptOut(false)}
                  className="rounded-xl border border-amber-400 bg-white px-3.5 py-2 text-sm font-semibold text-amber-800 transition-[background-color,color,border-color] hover:bg-amber-100 dark:border-amber-500/50 dark:bg-amber-900/30 dark:text-amber-100"
                >
                  Cancel
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
