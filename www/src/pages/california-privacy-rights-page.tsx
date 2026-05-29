import { useEffect, useState, type FormEvent } from 'react'
import { Header } from '../components/header'
import { LegalNav } from '../components/legal-nav'
import { SiteFooter } from '../components/site-footer'
import { API_BASE } from '../lib/api-base'
import { formatDate } from '../lib/format-date'

const API = `${API_BASE}/api/v1/compliance/ccpa`

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

export function CaliforniaPrivacyRightsPage() {
  const [optOut, setOptOut] = useState<OptOutState | null>(null)
  const [requests, setRequests] = useState<CCPARequest[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [requestType, setRequestType] = useState('know_categories')
  const [message, setMessage] = useState<string | null>(null)

  useEffect(() => {
    document.title = 'Your California Privacy Rights — Lextures'
  }, [])

  useEffect(() => {
    async function load() {
      try {
        const [optOutRes, reqRes] = await Promise.all([
          fetch(`${API}/opt-out`),
          fetch(`${API}/requests`),
        ])
        if (optOutRes.ok) {
          const data = (await optOutRes.json()) as OptOutState
          setOptOut(data)
        }
        if (reqRes.ok) {
          const data = (await reqRes.json()) as { requests?: CCPARequest[] }
          setRequests(data.requests ?? [])
        }
      } catch {
        setError('Failed to load California privacy rights data.')
      } finally {
        setLoading(false)
      }
    }
    void load()
  }, [])

  async function toggleDoNotSell() {
    if (!optOut) return
    setSubmitting(true)
    setMessage(null)
    try {
      const res = await fetch(`${API}/opt-out`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ doNotSell: !optOut.doNotSell }),
      })
      if (res.ok) {
        const data = (await res.json()) as { doNotSell: boolean }
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
      const res = await fetch(`${API}/opt-out`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ limitSensitivePI: !optOut.limitSensitivePI }),
      })
      if (res.ok) {
        const data = (await res.json()) as { limitSensitivePI: boolean }
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

  async function submitRequest(e: FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setMessage(null)
    try {
      const res = await fetch(`${API}/requests`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ requestType }),
      })
      if (res.status === 201) {
        const data = (await res.json()) as { id: string }
        setMessage(
          `Request submitted. We will respond within 45 calendar days as required by CPRA § 1798.130(a)(2). Reference: ${data.id}`,
        )
        const reqRes = await fetch(`${API}/requests`)
        if (reqRes.ok) {
          const d = (await reqRes.json()) as { requests?: CCPARequest[] }
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

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-stone-50 text-slate-700">
      <Header />

      <main id="main-content" className="mx-auto max-w-3xl px-4 py-8 sm:px-6 lg:py-12">
        <LegalNav />

        {loading ? (
          <div className="flex min-h-[40vh] items-center justify-center" role="status" aria-label="Loading">
            <div className="h-8 w-8 animate-spin rounded-full border-4 border-stone-200 border-t-accent" />
          </div>
        ) : (
          <div className="space-y-8">
            <header>
              <h1 className="font-display text-3xl font-normal tracking-tight text-stone-900 sm:text-4xl">
                Your California Privacy Rights
              </h1>
              <p className="mt-2 text-base leading-relaxed text-stone-600">
                California residents have rights under the California Consumer Privacy Act (CCPA) and
                California Privacy Rights Act (CPRA), including the right to know, delete, correct, and
                opt out of the sale or sharing of your personal information.
              </p>
            </header>

            {error && (
              <div
                role="alert"
                className="rounded-md bg-red-50 px-4 py-3 text-sm text-red-700"
              >
                {error}
              </div>
            )}

            {message && (
              <div
                role="status"
                className="rounded-md bg-blue-50 px-4 py-3 text-sm text-blue-700"
              >
                {message}
              </div>
            )}

            {optOut && (
              <section aria-labelledby="dns-heading">
                <h2 id="dns-heading" className="mb-3 text-lg font-semibold text-stone-900">
                  Do Not Sell or Share My Personal Information
                </h2>
                <div className="space-y-4 rounded-lg border border-stone-200/90 bg-white p-4">
                  <p className="text-sm text-stone-600">
                    Under CPRA § 1798.120, you have the right to opt out of the sale or sharing of your
                    personal information. Lextures respects the Global Privacy Control (GPC) browser
                    signal as an automatic opt-out.
                  </p>
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <p className="text-sm font-medium text-stone-900">
                        Do Not Sell or Share My Personal Information
                      </p>
                      <p className="mt-0.5 text-xs text-stone-500">
                        {optOut.doNotSell
                          ? 'You have opted out. Lextures will not sell or share your personal information.'
                          : 'Your information may be shared for analytics and platform improvement.'}
                      </p>
                    </div>
                    <button
                      type="button"
                      disabled={submitting}
                      onClick={() => void toggleDoNotSell()}
                      aria-pressed={optOut.doNotSell}
                      className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50 ${
                        optOut.doNotSell ? 'bg-accent' : 'bg-stone-200'
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

                  <div className="flex items-center justify-between gap-4 border-t border-stone-100 pt-4">
                    <div>
                      <p className="text-sm font-medium text-stone-900">
                        Limit Use of Sensitive Personal Information
                      </p>
                      <p className="mt-0.5 text-xs text-stone-500">
                        {optOut.limitSensitivePI
                          ? 'Sensitive information (browsing history, inferences) is limited to service delivery only.'
                          : 'Sensitive information may be used for AI tutoring and analytics.'}
                      </p>
                    </div>
                    <button
                      type="button"
                      disabled={submitting}
                      onClick={() => void toggleLimitSensitivePI()}
                      aria-pressed={optOut.limitSensitivePI}
                      className={`relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent disabled:opacity-50 ${
                        optOut.limitSensitivePI ? 'bg-accent' : 'bg-stone-200'
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

            <section aria-labelledby="rights-heading">
              <h2 id="rights-heading" className="mb-3 text-lg font-semibold text-stone-900">
                Submit a California Privacy Rights Request
              </h2>
              <form
                onSubmit={(e) => void submitRequest(e)}
                className="space-y-4 rounded-lg border border-stone-200/90 bg-white p-4"
              >
                <div>
                  <label
                    htmlFor="request-type"
                    className="mb-1 block text-sm font-medium text-stone-700"
                  >
                    Request type
                  </label>
                  <select
                    id="request-type"
                    value={requestType}
                    onChange={(e) => setRequestType(e.target.value)}
                    className="block w-full rounded-md border border-stone-300 bg-white px-3 py-2 text-sm text-stone-900 focus:outline-none focus:ring-2 focus:ring-accent/30"
                  >
                    {Object.entries(requestTypeLabels).map(([value, label]) => (
                      <option key={value} value={value}>
                        {label}
                      </option>
                    ))}
                  </select>
                </div>
                <p className="text-xs text-stone-500">
                  We will respond within 45 calendar days in accordance with CPRA § 1798.130(a)(2). You
                  will not be discriminated against for exercising your rights (§ 1798.125).
                </p>
                <button
                  type="submit"
                  disabled={submitting}
                  className="rounded-md bg-accent px-4 py-2 text-sm font-semibold text-white hover:bg-accent/90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-50"
                >
                  {submitting ? 'Submitting…' : 'Submit Request'}
                </button>
              </form>
            </section>

            {requests.length > 0 && (
              <section aria-labelledby="requests-history-heading">
                <h2 id="requests-history-heading" className="mb-3 text-lg font-semibold text-stone-900">
                  Your Requests
                </h2>
                <ul className="divide-y divide-stone-200 overflow-hidden rounded-lg border border-stone-200/90">
                  {requests.map((r) => (
                    <li key={r.id} className="flex items-center justify-between bg-white px-4 py-3">
                      <div>
                        <p className="text-sm font-medium text-stone-900">
                          {requestTypeLabels[r.requestType] ?? r.requestType}
                        </p>
                        <p className="text-xs text-stone-500">
                          Submitted {formatDate(r.requestedAt, { dateStyle: 'medium' })} · Due by{' '}
                          {formatDate(r.dueAt, { dateStyle: 'medium' })}
                        </p>
                      </div>
                      <span
                        className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
                          r.status === 'completed'
                            ? 'bg-emerald-50 text-emerald-700'
                            : r.status === 'denied'
                              ? 'bg-red-50 text-red-700'
                              : 'bg-amber-50 text-amber-700'
                        }`}
                      >
                        {r.status}
                      </span>
                    </li>
                  ))}
                </ul>
              </section>
            )}

            <section aria-labelledby="contact-heading">
              <h2 id="contact-heading" className="mb-2 text-lg font-semibold text-stone-900">
                Contact Us
              </h2>
              <p className="text-sm text-stone-600">
                For questions about your California privacy rights, contact us at{' '}
                <a
                  href="mailto:privacy@lextures.com"
                  className="text-accent underline underline-offset-2 hover:text-accent/90"
                >
                  privacy@lextures.com
                </a>
                . You may also designate an authorized agent to submit a request on your behalf.
              </p>
            </section>
          </div>
        )}
      </main>

      <SiteFooter />
    </div>
  )
}
