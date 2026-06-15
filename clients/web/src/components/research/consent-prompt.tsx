import { useEffect, useMemo, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ShieldCheck } from 'lucide-react'
import {
  fetchPendingConsentStudies,
  respondToConsentStudy,
  type ConsentStudy,
} from '../../lib/research-consent-api'

const SESSION_KEY = 'lextures.consentPrompt.dismissed'

function dismissedThisSession(): Set<string> {
  try {
    const raw = sessionStorage.getItem(SESSION_KEY)
    return new Set<string>(raw ? (JSON.parse(raw) as string[]) : [])
  } catch {
    return new Set<string>()
  }
}

function markDismissed(studyId: string) {
  try {
    const set = dismissedThisSession()
    set.add(studyId)
    sessionStorage.setItem(SESSION_KEY, JSON.stringify([...set]))
  } catch {
    // ignore storage errors (private mode, etc.)
  }
}

/**
 * ConsentPrompt presents pending IRB consent studies to a targeted student as a
 * full-screen interstitial (plan 14.15, FR-2). Shown at most once per study per
 * session; deciding (grant/decline) removes it permanently.
 */
export function ConsentPrompt() {
  const [studies, setStudies] = useState<ConsentStudy[]>([])
  const [index, setIndex] = useState(0)
  const [agreed, setAgreed] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const bottomRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    let cancelled = false
    void fetchPendingConsentStudies()
      .then((all) => {
        if (cancelled) return
        const dismissed = dismissedThisSession()
        setStudies(all.filter((s) => !dismissed.has(s.id)))
      })
      .catch(() => {
        /* prompts must not block the dashboard (NFR Scalability) */
      })
    return () => {
      cancelled = true
    }
  }, [])

  const current = studies[index]

  // Reset per-study local state when advancing.
  useEffect(() => {
    setAgreed(false)
    setError(null)
  }, [index])

  const remaining = useMemo(() => studies.length - index, [studies.length, index])

  if (!current) {
    return null
  }

  function advance() {
    setSubmitting(false)
    if (index + 1 >= studies.length) {
      setStudies([])
      setIndex(0)
    } else {
      setIndex((i) => i + 1)
    }
  }

  async function respond(decision: 'granted' | 'declined') {
    if (!current) return
    setSubmitting(true)
    setError(null)
    try {
      await respondToConsentStudy(current.id, decision)
      advance()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not record your decision.')
      setSubmitting(false)
    }
  }

  function remindLater() {
    if (!current) return
    markDismissed(current.id)
    advance()
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby="consent-prompt-title"
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/60 p-4 backdrop-blur-sm"
    >
      <div className="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-2xl bg-white shadow-xl dark:bg-neutral-900">
        <header className="flex items-start gap-3 border-b border-slate-200 px-6 py-4 dark:border-neutral-800">
          <ShieldCheck className="mt-0.5 h-6 w-6 shrink-0 text-violet-600 dark:text-violet-400" aria-hidden="true" />
          <div className="min-w-0">
            <h2 id="consent-prompt-title" className="text-lg font-semibold text-slate-900 dark:text-neutral-100">
              Research participation request
            </h2>
            <p className="mt-0.5 text-sm text-slate-600 dark:text-neutral-400">
              {current.title}
              <span className="ms-2 text-xs text-slate-400">IRB protocol {current.irbProtocol}</span>
            </p>
          </div>
        </header>

        <div className="flex items-center justify-between border-b border-slate-100 px-6 py-2 dark:border-neutral-800">
          <a
            href="#consent-prompt-actions"
            className="text-xs font-medium text-violet-600 hover:underline dark:text-violet-400"
            onClick={(e) => {
              e.preventDefault()
              bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
            }}
          >
            Skip to bottom
          </a>
          {remaining > 1 ? (
            <span className="text-xs text-slate-400">{remaining} studies awaiting your response</span>
          ) : null}
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-4">
          <div className="prose prose-sm max-w-none dark:prose-invert">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{current.consentText}</ReactMarkdown>
          </div>
          {current.dataUseDescription ? (
            <div className="mt-4 rounded-lg bg-slate-50 px-4 py-3 dark:bg-neutral-800/60">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
                How your data will be used
              </p>
              <p className="mt-1 text-sm text-slate-700 dark:text-neutral-300">{current.dataUseDescription}</p>
            </div>
          ) : null}
          <div ref={bottomRef} />
        </div>

        <footer
          id="consent-prompt-actions"
          className="space-y-3 border-t border-slate-200 px-6 py-4 dark:border-neutral-800"
        >
          {error ? (
            <p role="alert" className="text-sm text-red-600 dark:text-red-400">
              {error}
            </p>
          ) : null}
          <label className="flex items-start gap-2 text-sm text-slate-700 dark:text-neutral-300">
            <input
              type="checkbox"
              checked={agreed}
              onChange={(e) => setAgreed(e.target.checked)}
              className="mt-0.5 h-4 w-4 rounded border-slate-300"
            />
            <span>I have read and agree to participate in this research study.</span>
          </label>
          <div className="flex flex-wrap items-center justify-end gap-3">
            <button
              type="button"
              onClick={remindLater}
              disabled={submitting}
              className="text-sm font-medium text-slate-500 hover:text-slate-700 disabled:opacity-50 dark:text-neutral-400"
            >
              Remind me later
            </button>
            <button
              type="button"
              onClick={() => void respond('declined')}
              disabled={submitting}
              className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200 dark:hover:bg-neutral-800"
            >
              No, I decline
            </button>
            <button
              type="button"
              onClick={() => void respond('granted')}
              disabled={!agreed || submitting}
              className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-50"
            >
              I agree
            </button>
          </div>
        </footer>
      </div>
    </div>
  )
}
