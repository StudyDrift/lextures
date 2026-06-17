import React, { useEffect, useState } from 'react'
import { usePlatformFeatures } from '../context/platform-features-context'
import { formatDate } from '../lib/format'

const API = '/api/v1/compliance/gdpr'

interface Consent {
  id: string
  purpose: string
  lawfulBasis: string
  consentVersion: string
  grantedAt: string
  withdrawnAt?: string
}

interface DSARRequest {
  id: string
  requestType: string
  status: string
  requestedAt: string
  dueAt: string
  completedAt?: string
}

function purposeLabel(purpose: string): string {
  const labels: Record<string, string> = {
    ai_processing: 'AI-assisted tutoring and feedback',
    analytics: 'Platform analytics',
    marketing: 'Marketing communications',
  }
  return labels[purpose] ?? purpose
}

export default function PrivacyCentrePage() {
  const { gdprModuleEnabled, loading: featuresLoading } = usePlatformFeatures()
  const [consents, setConsents] = useState<Consent[]>([])
  const [dsars, setDsars] = useState<DSARRequest[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [dsarType, setDsarType] = useState('access')
  const [message, setMessage] = useState<string | null>(null)

  useEffect(() => {
    if (featuresLoading || !gdprModuleEnabled) return

    async function load() {
      try {
        const [cRes, dRes] = await Promise.all([
          fetch(API + '/consents'),
          fetch(API + '/dsar'),
        ])
        if (cRes.ok) {
          const data = await cRes.json()
          setConsents(data.consents ?? [])
        }
        if (dRes.ok) {
          const data = await dRes.json()
          setDsars(data.requests ?? [])
        }
      } catch {
        setError('Failed to load privacy data.')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [featuresLoading, gdprModuleEnabled])

  async function withdrawConsent(id: string) {
    setSubmitting(true)
    setMessage(null)
    try {
      const res = await fetch(`${API}/consents/${id}`, { method: 'DELETE' })
      if (res.ok) {
        setConsents((prev) =>
          prev.map((c) =>
            c.id === id ? { ...c, withdrawnAt: new Date().toISOString() } : c,
          ),
        )
        setMessage('Consent withdrawn successfully.')
      } else {
        setMessage('Failed to withdraw consent.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  async function submitDSAR(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setMessage(null)
    try {
      const res = await fetch(`${API}/dsar`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ requestType: dsarType }),
      })
      if (res.status === 201) {
        const data = await res.json()
        setMessage(`Request submitted. We will respond within 30 days. Reference: ${data.id}`)
        const dRes = await fetch(API + '/dsar')
        if (dRes.ok) {
          const d = await dRes.json()
          setDsars(d.requests ?? [])
        }
      } else if (res.status === 409) {
        setMessage('Your request is already in progress. Please wait for the current one to complete.')
      } else {
        setMessage('Failed to submit request.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  if (featuresLoading || loading) {
    return (
      <div className="flex items-center justify-center min-h-[40vh]" role="status" aria-label="Loading">
        <div className="h-8 w-8 rounded-full border-4 border-indigo-200 border-t-indigo-600 animate-spin" />
      </div>
    )
  }

  if (!gdprModuleEnabled) {
    return (
      <div className="max-w-3xl mx-auto px-4 py-8">
        <h1 className="text-2xl font-bold text-slate-900 dark:text-neutral-50">Privacy Center</h1>
        <p className="mt-2 text-sm text-slate-500 dark:text-neutral-400">
          Privacy features are not enabled for this platform. Contact your system administrator.
        </p>
      </div>
    )
  }

  return (
    <div className="max-w-3xl mx-auto px-4 py-8 space-y-8">
      <header>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-neutral-50">Privacy Center</h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
          Manage your data rights under GDPR / UK GDPR. You can view and withdraw consents, and
          request a copy or deletion of your personal data.
        </p>
      </header>

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

      {/* Consent management */}
      <section aria-labelledby="consents-heading">
        <h2 id="consents-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100 mb-3">
          Active Consents
        </h2>
        {consents.length === 0 ? (
          <p className="text-sm text-slate-500 dark:text-neutral-400">No consent records found.</p>
        ) : (
          <ul className="divide-y divide-slate-200 dark:divide-neutral-800 border border-slate-200 dark:border-neutral-800 rounded-lg overflow-hidden">
            {consents.map((c) => (
              <li key={c.id} className="flex items-center justify-between px-4 py-3 bg-white dark:bg-neutral-900">
                <div>
                  <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">{purposeLabel(c.purpose)}</p>
                  <p className="text-xs text-slate-500 dark:text-neutral-400">
                    Basis: {c.lawfulBasis} · Version {c.consentVersion} ·{' '}
                    {c.withdrawnAt ? (
                      <span className="text-amber-600 dark:text-amber-400">Withdrawn {formatDate(c.withdrawnAt, { dateStyle: 'medium' })}</span>
                    ) : (
                      <span className="text-emerald-600 dark:text-emerald-400">Active</span>
                    )}
                  </p>
                </div>
                {!c.withdrawnAt && (
                  <button
                    type="button"
                    disabled={submitting}
                    onClick={() => withdrawConsent(c.id)}
                    className="ms-4 shrink-0 rounded-md border border-slate-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-1.5 text-xs font-medium text-slate-700 dark:text-neutral-200 hover:bg-slate-50 dark:hover:bg-neutral-800 disabled:opacity-50"
                  >
                    Withdraw
                  </button>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* DSAR form */}
      <section aria-labelledby="dsar-heading">
        <h2 id="dsar-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100 mb-3">
          Submit a Data Request
        </h2>
        <form onSubmit={submitDSAR} className="rounded-lg border border-slate-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4 space-y-4">
          <div>
            <label htmlFor="dsar-type" className="block text-sm font-medium text-slate-700 dark:text-neutral-300 mb-1">
              Request type
            </label>
            <select
              id="dsar-type"
              value={dsarType}
              onChange={(e) => setDsarType(e.target.value)}
              className="block w-full rounded-md border border-slate-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-2 py-1.5 text-sm text-slate-900 dark:text-neutral-50 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              <option value="access">Access (download my data)</option>
              <option value="portability">Portability (machine-readable export)</option>
              <option value="erasure">Erasure (delete my account and data)</option>
              <option value="rectification">Rectification (correct my data)</option>
              <option value="restriction">Restriction (limit processing)</option>
              <option value="objection">Objection (object to processing)</option>
            </select>
          </div>
          <p className="text-xs text-slate-500 dark:text-neutral-400">
            We will respond within 30 calendar days in accordance with GDPR Article 12(3).
          </p>
          <button
            type="submit"
            disabled={submitting}
            className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:opacity-50"
          >
            {submitting ? 'Submitting…' : 'Submit Request'}
          </button>
        </form>
      </section>

      {/* Existing DSARs */}
      {dsars.length > 0 && (
        <section aria-labelledby="dsar-history-heading">
          <h2 id="dsar-history-heading" className="text-lg font-semibold text-slate-800 dark:text-neutral-100 mb-3">
            Your Requests
          </h2>
          <ul className="divide-y divide-slate-200 dark:divide-neutral-800 border border-slate-200 dark:border-neutral-800 rounded-lg overflow-hidden">
            {dsars.map((r) => (
              <li key={r.id} className="flex items-center justify-between px-4 py-3 bg-white dark:bg-neutral-900">
                <div>
                  <p className="text-sm font-medium capitalize text-slate-900 dark:text-neutral-50">{r.requestType}</p>
                  <p className="text-xs text-slate-500 dark:text-neutral-400">
                    Submitted {formatDate(r.requestedAt, { dateStyle: 'medium' })} · Due {formatDate(r.dueAt, { dateStyle: 'medium' })}
                  </p>
                </div>
                <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
                  r.status === 'completed'
                    ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
                    : r.status === 'rejected'
                    ? 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300'
                    : 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
                }`}>
                  {r.status}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}
    </div>
  )
}
