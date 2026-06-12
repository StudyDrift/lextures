import { useCallback, useEffect, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { formatDate } from '../../lib/format'
import {
  fetchTranscriptRequests,
  submitTranscriptRequest,
  type TranscriptRequest,
} from '../../lib/transcripts-api'
import { LmsPage } from './lms-page'

function statusLabel(status: TranscriptRequest['status']): string {
  switch (status) {
    case 'queued':
      return 'Queued'
    case 'submitted':
      return 'Submitted to institution'
    case 'failed':
      return 'Failed'
    default: {
      const _exhaustive: never = status
      return _exhaustive
    }
  }
}

function statusClass(status: TranscriptRequest['status']): string {
  switch (status) {
    case 'submitted':
      return 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
    case 'failed':
      return 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300'
    case 'queued':
      return 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
    default: {
      const _exhaustive: never = status
      return _exhaustive
    }
  }
}

export default function TranscriptsPage() {
  const { ffTranscripts, loading: featuresLoading } = usePlatformFeatures()
  const [requests, setRequests] = useState<TranscriptRequest[]>([])
  const [loading, setLoading] = useState(true)
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setRequests(await fetchTranscriptRequests())
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load transcript requests.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffTranscripts) return
    void load()
  }, [featuresLoading, ffTranscripts, load])

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setMessage(null)
    setError(null)
    try {
      const req = await submitTranscriptRequest()
      setMessage('Your transcript request has been queued. We will notify your institution shortly.')
      setRequests((prev) => [req, ...prev])
      window.setTimeout(() => void load(), 3000)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not submit request.')
    } finally {
      setSubmitting(false)
    }
  }

  if (featuresLoading) {
    return (
      <LmsPage title="Transcripts" description="Request an official transcript from your institution.">
        <p className="mt-8 text-sm text-slate-500">Loading…</p>
      </LmsPage>
    )
  }

  if (!ffTranscripts) {
    return (
      <LmsPage title="Transcripts" description="Request an official transcript from your institution.">
        <p className="mt-8 text-sm text-slate-600 dark:text-neutral-400">
          Transcripts is not enabled on this platform.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Transcripts" description="Request an official transcript from your institution.">
      <div className="mt-8 max-w-2xl space-y-8">
        {error && (
          <div role="alert" className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-300">
            {error}
          </div>
        )}
        {message && (
          <div role="status" className="rounded-md bg-blue-50 px-4 py-3 text-sm text-blue-700 dark:bg-blue-950 dark:text-blue-300">
            {message}
          </div>
        )}

        <section aria-labelledby="request-heading">
          <h2 id="request-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100">
            Request a transcript
          </h2>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Submit a request to receive an official transcript. Your institution will process the request
            and contact you with next steps.
          </p>
          <form onSubmit={handleSubmit} className="mt-4">
            <button
              type="submit"
              disabled={submitting || loading}
              className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
            >
              {submitting ? 'Submitting…' : 'Request transcript'}
            </button>
          </form>
        </section>

        <section aria-labelledby="history-heading">
          <h2 id="history-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100">
            Your requests
          </h2>
          {loading ? (
            <p className="mt-4 text-sm text-slate-500">Loading…</p>
          ) : requests.length === 0 ? (
            <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">No transcript requests yet.</p>
          ) : (
            <ul className="mt-4 divide-y divide-slate-200 overflow-hidden rounded-lg border border-slate-200 dark:divide-neutral-800 dark:border-neutral-800">
              {requests.map((r) => (
                <li key={r.id} className="flex items-start justify-between gap-4 bg-white px-4 py-3 dark:bg-neutral-900">
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                      Transcript request
                    </p>
                    <p className="text-xs text-slate-500 dark:text-neutral-400">
                      Submitted {formatDate(r.requestedAt, { dateStyle: 'medium', timeStyle: 'short' })}
                      {r.submittedAt
                        ? ` · Delivered ${formatDate(r.submittedAt, { dateStyle: 'medium', timeStyle: 'short' })}`
                        : null}
                    </p>
                    {r.errorMessage && (
                      <p className="mt-1 text-xs text-red-600 dark:text-red-400">{r.errorMessage}</p>
                    )}
                  </div>
                  <span className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-medium ${statusClass(r.status)}`}>
                    {statusLabel(r.status)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </LmsPage>
  )
}
