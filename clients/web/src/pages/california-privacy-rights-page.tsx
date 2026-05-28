import React, { useEffect, useState } from 'react'
import { formatDate } from '../lib/format'

const API = '/api/v1/compliance/ccpa'

interface OptOutState {
  doNotSell: boolean
  limitSensitivePI: boolean
}

interface CCPARequest {
  id: string
  requestType: string
  status: string
  requestedAt: string
  dueAt: string
  completedAt?: string
  responsePayload?: string
}

const requestTypeLabels: Record<string, string> = {
  know_categories: 'Right to Know — Categories of personal information collected',
  know_specific: 'Right to Know — Specific pieces of personal information',
  delete: 'Right to Delete — Erasure of personal information',
  correct: 'Right to Correct — Correction of inaccurate personal information',
  limit_sensitive: 'Right to Limit — Limit use of sensitive personal information',
}

export default function CaliforniaPrivacyRightsPage() {
  const [optOut, setOptOut] = useState<OptOutState | null>(null)
  const [requests, setRequests] = useState<CCPARequest[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [requestType, setRequestType] = useState('know_categories')
  const [message, setMessage] = useState<string | null>(null)

  useEffect(() => {
    async function load() {
      try {
        const [optOutRes, reqRes] = await Promise.all([
          fetch(API + '/opt-out'),
          fetch(API + '/requests'),
        ])
        if (optOutRes.ok) {
          const data = await optOutRes.json()
          setOptOut(data)
        }
        if (reqRes.ok) {
          const data = await reqRes.json()
          setRequests(data.requests ?? [])
        }
      } catch {
        setError('Failed to load California privacy rights data.')
      } finally {
        setLoading(false)
      }
    }
    load()
  }, [])

  async function toggleDoNotSell() {
    if (!optOut) return
    setSubmitting(true)
    setMessage(null)
    try {
      const res = await fetch(API + '/opt-out', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ doNotSell: !optOut.doNotSell }),
      })
      if (res.ok) {
        const data = await res.json()
        setOptOut((prev) => (prev ? { ...prev, doNotSell: data.doNotSell } : prev))
        setMessage(
          data.doNotSell
            ? 'Do Not Sell or Share preference saved. Lextures will not sell or share your personal information.'
            : 'Do Not Sell or Share preference removed.',
        )
      } else {
        setMessage('Failed to update preference.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  async function toggleLimitSensitivePI() {
    if (!optOut) return
    setSubmitting(true)
    setMessage(null)
    try {
      const res = await fetch(API + '/opt-out', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ limitSensitivePI: !optOut.limitSensitivePI }),
      })
      if (res.ok) {
        const data = await res.json()
        setOptOut((prev) => (prev ? { ...prev, limitSensitivePI: data.limitSensitivePI } : prev))
        setMessage(
          data.limitSensitivePI
            ? 'Sensitive personal information will only be used for service delivery.'
            : 'Limit Sensitive PI preference removed.',
        )
      } else {
        setMessage('Failed to update preference.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  async function submitRequest(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setMessage(null)
    try {
      const res = await fetch(API + '/requests', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ requestType }),
      })
      if (res.status === 201) {
        const data = await res.json()
        setMessage(
          `Request submitted. We will respond within 45 calendar days as required by CPRA § 1798.130(a)(2). Reference: ${data.id}`,
        )
        const reqRes = await fetch(API + '/requests')
        if (reqRes.ok) {
          const d = await reqRes.json()
          setRequests(d.requests ?? [])
        }
      } else if (res.status === 409) {
        setMessage(
          'A request of this type is already in progress. Please wait for the current one to complete.',
        )
      } else {
        setMessage('Failed to submit request.')
      }
    } finally {
      setSubmitting(false)
    }
  }

  if (loading) {
    return (
      <div
        className="flex items-center justify-center min-h-[40vh]"
        role="status"
        aria-label="Loading"
      >
        <div className="h-8 w-8 rounded-full border-4 border-indigo-200 border-t-indigo-600 animate-spin" />
      </div>
    )
  }

  return (
    <div className="max-w-3xl mx-auto px-4 py-8 space-y-8">
      <header>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-neutral-50">
          Your California Privacy Rights
        </h1>
        <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
          California residents have rights under the California Consumer Privacy Act (CCPA) and
          California Privacy Rights Act (CPRA), including the right to know, delete, correct, and
          opt out of the sale or sharing of your personal information.
        </p>
      </header>

      {error && (
        <div
          role="alert"
          className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-950 dark:text-red-300"
        >
          {error}
        </div>
      )}

      {message && (
        <div
          role="status"
          className="rounded-md bg-blue-50 px-4 py-3 text-sm text-blue-700 dark:bg-blue-950 dark:text-blue-300"
        >
          {message}
        </div>
      )}

      {/* Do Not Sell or Share */}
      {optOut && (
        <section aria-labelledby="dns-heading">
          <h2
            id="dns-heading"
            className="text-lg font-semibold text-slate-800 dark:text-neutral-100 mb-3"
          >
            Do Not Sell or Share My Personal Information
          </h2>
          <div className="rounded-lg border border-slate-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4 space-y-4">
            <p className="text-sm text-slate-600 dark:text-neutral-400">
              Under CPRA § 1798.120, you have the right to opt out of the sale or sharing of your
              personal information. Lextures respects the Global Privacy Control (GPC) browser
              signal as an automatic opt-out.
            </p>
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                  Do Not Sell or Share My Personal Information
                </p>
                <p className="text-xs text-slate-500 dark:text-neutral-400 mt-0.5">
                  {optOut.doNotSell
                    ? 'You have opted out. Lextures will not sell or share your personal information.'
                    : 'Your information may be shared for analytics and platform improvement.'}
                </p>
              </div>
              <button
                type="button"
                disabled={submitting}
                onClick={toggleDoNotSell}
                aria-pressed={optOut.doNotSell}
                className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-indigo-600 disabled:opacity-50 ${
                  optOut.doNotSell ? 'bg-indigo-600' : 'bg-slate-200 dark:bg-neutral-700'
                }`}
              >
                <span className="sr-only">
                  {optOut.doNotSell ? 'Opt back in' : 'Opt out of sale/sharing'}
                </span>
                <span
                  className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                    optOut.doNotSell ? 'translate-x-5' : 'translate-x-0'
                  }`}
                />
              </button>
            </div>

            <div className="flex items-center justify-between border-t border-slate-100 dark:border-neutral-800 pt-4">
              <div>
                <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                  Limit Use of Sensitive Personal Information
                </p>
                <p className="text-xs text-slate-500 dark:text-neutral-400 mt-0.5">
                  {optOut.limitSensitivePI
                    ? 'Sensitive information (browsing history, inferences) is limited to service delivery only.'
                    : 'Sensitive information may be used for AI tutoring and analytics.'}
                </p>
              </div>
              <button
                type="button"
                disabled={submitting}
                onClick={toggleLimitSensitivePI}
                aria-pressed={optOut.limitSensitivePI}
                className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-indigo-600 disabled:opacity-50 ${
                  optOut.limitSensitivePI ? 'bg-indigo-600' : 'bg-slate-200 dark:bg-neutral-700'
                }`}
              >
                <span className="sr-only">
                  {optOut.limitSensitivePI
                    ? 'Remove limit on sensitive PI'
                    : 'Limit use of sensitive PI'}
                </span>
                <span
                  className={`pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out ${
                    optOut.limitSensitivePI ? 'translate-x-5' : 'translate-x-0'
                  }`}
                />
              </button>
            </div>
          </div>
        </section>
      )}

      {/* Rights Request Form */}
      <section aria-labelledby="rights-heading">
        <h2
          id="rights-heading"
          className="text-lg font-semibold text-slate-800 dark:text-neutral-100 mb-3"
        >
          Submit a California Privacy Rights Request
        </h2>
        <form
          onSubmit={submitRequest}
          className="rounded-lg border border-slate-200 dark:border-neutral-800 bg-white dark:bg-neutral-900 p-4 space-y-4"
        >
          <div>
            <label
              htmlFor="request-type"
              className="block text-sm font-medium text-slate-700 dark:text-neutral-300 mb-1"
            >
              Request type
            </label>
            <select
              id="request-type"
              value={requestType}
              onChange={(e) => setRequestType(e.target.value)}
              className="block w-full rounded-md border border-slate-300 dark:border-neutral-700 bg-white dark:bg-neutral-900 px-3 py-2 text-sm text-slate-900 dark:text-neutral-50 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              {Object.entries(requestTypeLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </div>
          <p className="text-xs text-slate-500 dark:text-neutral-400">
            We will respond within 45 calendar days in accordance with CPRA § 1798.130(a)(2). You
            will not be discriminated against for exercising your rights (§ 1798.125).
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

      {/* Existing Requests */}
      {requests.length > 0 && (
        <section aria-labelledby="requests-history-heading">
          <h2
            id="requests-history-heading"
            className="text-lg font-semibold text-slate-800 dark:text-neutral-100 mb-3"
          >
            Your Requests
          </h2>
          <ul className="divide-y divide-slate-200 dark:divide-neutral-800 border border-slate-200 dark:border-neutral-800 rounded-lg overflow-hidden">
            {requests.map((r) => (
              <li
                key={r.id}
                className="flex items-center justify-between px-4 py-3 bg-white dark:bg-neutral-900"
              >
                <div>
                  <p className="text-sm font-medium text-slate-900 dark:text-neutral-50">
                    {requestTypeLabels[r.requestType] ?? r.requestType}
                  </p>
                  <p className="text-xs text-slate-500 dark:text-neutral-400">
                    Submitted {formatDate(r.requestedAt, { dateStyle: 'medium' })} · Due by{' '}
                    {formatDate(r.dueAt, { dateStyle: 'medium' })}
                  </p>
                </div>
                <span
                  className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
                    r.status === 'completed'
                      ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950 dark:text-emerald-300'
                      : r.status === 'denied'
                        ? 'bg-red-50 text-red-700 dark:bg-red-950 dark:text-red-300'
                        : 'bg-amber-50 text-amber-700 dark:bg-amber-950 dark:text-amber-300'
                  }`}
                >
                  {r.status}
                </span>
              </li>
            ))}
          </ul>
        </section>
      )}

      {/* Contact */}
      <section aria-labelledby="contact-heading">
        <h2
          id="contact-heading"
          className="text-lg font-semibold text-slate-800 dark:text-neutral-100 mb-2"
        >
          Contact Us
        </h2>
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          For questions about your California privacy rights, contact us at{' '}
          <a
            href="mailto:privacy@lextures.com"
            className="text-indigo-600 underline hover:text-indigo-500 dark:text-indigo-400"
          >
            privacy@lextures.com
          </a>
          . You may also designate an authorized agent to submit a request on your behalf.
        </p>
      </section>
    </div>
  )
}
