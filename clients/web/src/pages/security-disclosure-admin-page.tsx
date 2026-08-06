import { useCallback, useEffect, useState } from 'react'
import { authorizedFetch } from '../lib/api'
import { MARKETING_SITE_URLS } from '../lib/marketing-site'

const API = '/api/v1/compliance/security-reports'

interface SecurityReport {
  id: string
  reporterHandle?: string
  reportDate: string
  severity?: string
  cvssScore?: number
  summary: string
  status: string
  patchDate?: string
  slaMet?: boolean
  bountyPaid: boolean
}

export default function SecurityDisclosureAdminPage() {
  const [reports, setReports] = useState<SecurityReport[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [summary, setSummary] = useState('')
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await authorizedFetch(API)
      if (res.status === 404) {
        setError('Security disclosure module is not enabled on this environment.')
        return
      }
      if (res.status === 403) {
        setError('You do not have permission to view security reports.')
        return
      }
      if (!res.ok) {
        setError('Could not load security reports.')
        return
      }
      const body = (await res.json()) as { reports: SecurityReport[] }
      setReports(body.reports ?? [])
    } catch {
      setError('Could not load security reports.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setMessage(null)
    const res = await authorizedFetch(API, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ summary, severity: 'medium' }),
    })
    if (res.status === 201) {
      setSummary('')
      setMessage('Report logged.')
      void load()
    } else {
      setMessage('Failed to log report.')
    }
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-fg-default">Security reports</h1>
          <p className="mt-1 text-sm text-fg-muted">
            Public policy:{' '}
            <a href={MARKETING_SITE_URLS.security} className="text-accent-fg underline dark:text-indigo-300">
              {MARKETING_SITE_URLS.security}
            </a>
          </p>
        </div>
        <button
          type="button"
          className="text-sm text-accent-fg underline dark:text-indigo-300"
          onClick={() => {
            void authorizedFetch(`${API}/export`).then(async (res) => {
              if (!res.ok) return
              const blob = await res.blob()
              const url = URL.createObjectURL(blob)
              const a = document.createElement('a')
              a.href = url
              a.download = 'security_reports.csv'
              a.click()
              URL.revokeObjectURL(url)
            })
          }}
        >
          Export CSV
        </button>
      </div>

      <form onSubmit={handleCreate} className="mb-8 rounded-lg border border-border-default bg-surface-raised p-4 dark:border-border-subtle dark:bg-surface-raised">
        <label htmlFor="summary" className="block text-sm font-medium text-fg-muted">
          Log incoming report (summary)
        </label>
        <textarea
          id="summary"
          required
          value={summary}
          onChange={(e) => setSummary(e.target.value)}
          rows={3}
          className="mt-1 w-full rounded-md border border-border-strong px-3 py-2 text-sm dark:border-border-default dark:bg-surface-base"
        />
        <button
          type="submit"
          className="mt-3 rounded-md bg-accent-solid px-4 py-2 text-sm font-semibold text-white hover:bg-accent"
        >
          Add report
        </button>
        {message ? <p className="mt-2 text-sm text-emerald-700 dark:text-emerald-300">{message}</p> : null}
      </form>

      {loading ? <p className="text-sm text-fg-muted">Loading…</p> : null}
      {error ? <p className="text-sm text-danger-fg">{error}</p> : null}

      {!loading && !error ? (
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm border-collapse" aria-label="Security vulnerability reports">
            <thead>
              <tr className="border-b border-border-default">
                <th scope="col" className="py-2 pe-3 text-start font-semibold">Date</th>
                <th scope="col" className="py-2 pe-3 text-start font-semibold">Severity</th>
                <th scope="col" className="py-2 pe-3 text-start font-semibold">Summary</th>
                <th scope="col" className="py-2 pe-3 text-start font-semibold">Status</th>
                <th scope="col" className="py-2 text-start font-semibold">SLA met</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
              {reports.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-4 text-fg-muted">No reports on file.</td>
                </tr>
              ) : (
                reports.map((r) => (
                  <tr key={r.id}>
                    <td className="py-2 pe-3 whitespace-nowrap">{r.reportDate}</td>
                    <td className="py-2 pe-3">{r.severity ?? '—'}</td>
                    <td className="py-2 pe-3 max-w-md truncate" title={r.summary}>{r.summary}</td>
                    <td className="py-2 pe-3">{r.status}</td>
                    <td className="py-2">
                      {r.slaMet === undefined || r.slaMet === null ? '—' : r.slaMet ? 'Yes' : 'No'}
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  )
}
