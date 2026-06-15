import { useCallback, useEffect, useState } from 'react'
import { ShieldCheck } from 'lucide-react'
import { LmsPage } from './lms-page'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchConsentHistory,
  fetchPendingConsentStudies,
  respondToConsentStudy,
  type ConsentDecision,
  type ConsentHistoryEntry,
  type ConsentStudy,
} from '../../lib/research-consent-api'
import { formatDateTime } from '../../lib/format'

function decisionLabel(decision: ConsentDecision): string {
  switch (decision) {
    case 'granted':
      return 'Consented'
    case 'declined':
      return 'Declined'
    case 'withdrawn':
      return 'Withdrawn'
    default:
      return decision
  }
}

// latestByStudy keeps only the most recent decision per study (history is newest-first).
function latestByStudy(history: ConsentHistoryEntry[]): ConsentHistoryEntry[] {
  const seen = new Set<string>()
  const out: ConsentHistoryEntry[] = []
  for (const entry of history) {
    if (seen.has(entry.studyId)) continue
    seen.add(entry.studyId)
    out.push(entry)
  }
  return out
}

export default function ResearchStudiesPage() {
  const { ffResearchConsent, loading: featuresLoading } = usePlatformFeatures()
  const [history, setHistory] = useState<ConsentHistoryEntry[]>([])
  const [pending, setPending] = useState<ConsentStudy[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const load = useCallback(async () => {
    const [h, p] = await Promise.all([fetchConsentHistory(), fetchPendingConsentStudies()])
    setHistory(h)
    setPending(p)
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffResearchConsent) {
      setLoading(false)
      return
    }
    let cancelled = false
    void load()
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load research studies.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [ffResearchConsent, featuresLoading, load])

  async function withdraw(studyId: string) {
    setBusyId(studyId)
    setError(null)
    try {
      await respondToConsentStudy(studyId, 'withdrawn')
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not withdraw consent.')
    } finally {
      setBusyId(null)
    }
  }

  if (featuresLoading || loading) {
    return (
      <LmsPage title="Research studies">
        <p className="text-sm text-slate-500">Loading…</p>
      </LmsPage>
    )
  }

  if (!ffResearchConsent) {
    return (
      <LmsPage title="Research studies">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Research consent features are not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  const decisions = latestByStudy(history)

  return (
    <LmsPage title="Research studies">
      <p className="text-sm text-slate-600 dark:text-neutral-400">
        Manage your participation in IRB-approved research studies. You may withdraw your consent at
        any time; withdrawing removes your data from future research exports.
      </p>

      {error && (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}

      {pending.length > 0 && (
        <section aria-label="Studies awaiting your response" className="mt-6">
          <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
            Awaiting your response
          </h2>
          <ul className="mt-2 space-y-2">
            {pending.map((s) => (
              <li
                key={s.id}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-violet-100 bg-violet-50/70 px-4 py-3 dark:border-violet-900/40 dark:bg-violet-950/30"
              >
                <div className="min-w-0">
                  <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">{s.title}</p>
                  <p className="text-xs text-slate-500">IRB protocol {s.irbProtocol}</p>
                </div>
                <div className="flex gap-2">
                  <button
                    type="button"
                    disabled={busyId === s.id}
                    onClick={() => void respondToConsentStudy(s.id, 'declined').then(load)}
                    className="rounded-lg border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-700 disabled:opacity-50 dark:border-neutral-700 dark:text-neutral-200"
                  >
                    Decline
                  </button>
                  <button
                    type="button"
                    disabled={busyId === s.id}
                    onClick={() => void respondToConsentStudy(s.id, 'granted').then(load)}
                    className="rounded-lg bg-violet-600 px-3 py-1.5 text-xs font-semibold text-white disabled:opacity-50"
                  >
                    Consent
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section aria-label="Your consent decisions" className="mt-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          Your consent decisions
        </h2>
        {decisions.length === 0 ? (
          <p className="mt-2 text-sm text-slate-500">You have not responded to any research studies yet.</p>
        ) : (
          <ul className="mt-2 space-y-2">
            {decisions.map((entry) => (
              <li
                key={entry.studyId}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-slate-200 px-4 py-3 dark:border-neutral-800"
              >
                <div className="min-w-0">
                  <p className="flex items-center gap-2 text-sm font-medium text-slate-900 dark:text-neutral-100">
                    <ShieldCheck className="h-4 w-4 text-violet-500" aria-hidden="true" />
                    {entry.studyTitle || 'Research study'}
                  </p>
                  <p className="text-xs text-slate-500">
                    {decisionLabel(entry.decision)} · {formatDateTime(entry.createdAt)}
                  </p>
                </div>
                {entry.decision === 'granted' && (
                  <button
                    type="button"
                    disabled={busyId === entry.studyId}
                    onClick={() => void withdraw(entry.studyId)}
                    className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-700 hover:bg-red-50 disabled:opacity-50 dark:border-red-900/40 dark:text-red-400"
                  >
                    Withdraw consent
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
    </LmsPage>
  )
}
