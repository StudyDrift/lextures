import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { authorizedFetch } from '../lib/api'

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
          <h1 className="text-2xl font-semibold text-slate-900 dark:text-neutral-50">Security reports</h1>
          <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
            Responsible disclosure triage (plan 10.16). Public policy:{' '}
            <Link to="/security" className="text-indigo-700 underline dark:text-indigo-300">/security</Link>
          </p>
        </div>
        <button
          type="button"
          className="text-sm text-indigo-700 underline dark:text-indigo-300"
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

      <form onSubmit={handleCreate} className="mb-8 rounded-lg border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
        <label htmlFor="summary" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
          Log incoming report (summary)
        </label>
        <textarea
          id="summary"
          required
          value={summary}
          onChange={(e) => setSummary(e.target.value)}
          rows={3}
          className="mt-1 w-full rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
        />
        <button
          type="submit"
          className="mt-3 rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
        >
          Add report
        </button>
        {message ? <p className="mt-2 text-sm text-emerald-700 dark:text-emerald-300">{message}</p> : null}
      </form>

      {loading ? <p className="text-sm text-slate-500">Loading…</p> : null}
      {error ? <p className="text-sm text-red-600 dark:text-red-400">{error}</p> : null}

      {!loading && !error ? (
        <div className="overflow-x-auto">
          <table className="min-w-full text-sm border-collapse" aria-label="Security vulnerability reports">
            <thead>
              <tr className="border-b border-slate-200 dark:border-neutral-700">
                <th scope="col" className="py-2 pr-3 text-left font-semibold">Date</th>
                <th scope="col" className="py-2 pr-3 text-left font-semibold">Severity</th>
                <th scope="col" className="py-2 pr-3 text-left font-semibold">Summary</th>
                <th scope="col" className="py-2 pr-3 text-left font-semibold">Status</th>
                <th scope="col" className="py-2 text-left font-semibold">SLA met</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-100 dark:divide-neutral-800">
              {reports.length === 0 ? (
                <tr>
                  <td colSpan={5} className="py-4 text-slate-500">No reports on file.</td>
                </tr>
              ) : (
                reports.map((r) => (
                  <tr key={r.id}>
                    <td className="py-2 pr-3 whitespace-nowrap">{r.reportDate}</td>
                    <td className="py-2 pr-3">{r.severity ?? '—'}</td>
                    <td className="py-2 pr-3 max-w-md truncate" title={r.summary}>{r.summary}</td>
                    <td className="py-2 pr-3">{r.status}</td>
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
