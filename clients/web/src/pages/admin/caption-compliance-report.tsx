import { useCallback, useEffect, useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { fetchCaptionCompliance, type CaptionCoverageRow } from '../../lib/captions-api'
import { videoCaptionsFeatureEnabled } from '../../lib/platform-features'

export default function CaptionComplianceReportPage() {
  const titleId = useId()
  const [rows, setRows] = useState<CaptionCoverageRow[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await fetchCaptionCompliance()
      setRows(data.rows ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load report.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  if (!videoCaptionsFeatureEnabled()) {
    return (
      <main className="mx-auto max-w-3xl p-6">
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Video captions are not enabled on this platform. Enable{' '}
          <strong>Video captions</strong> in Settings → Global platform.
        </p>
      </main>
    )
  }

  return (
    <main className="mx-auto max-w-4xl p-6">
      <h1 id={titleId} className="text-xl font-bold text-slate-900 dark:text-neutral-100">
        Caption compliance report
      </h1>
      <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
        Videos missing reviewed captions across all courses ({rows.length} shown, max 500).
      </p>

      {loading ? (
        <p className="mt-6 text-sm" role="status">
          Loading…
        </p>
      ) : null}
      {error ? (
        <p className="mt-6 text-sm text-rose-700 dark:text-rose-200" role="alert">
          {error}
        </p>
      ) : null}

      {!loading && !error ? (
        <table className="mt-6 w-full border-collapse text-sm" aria-labelledby={titleId}>
          <caption className="sr-only">Uncaptioned platform videos</caption>
          <thead>
            <tr className="border-b border-slate-200 text-left dark:border-neutral-600">
              <th scope="col" className="py-2 pe-4">
                Object
              </th>
              <th scope="col" className="py-2 pe-4">
                Type
              </th>
              <th scope="col" className="py-2">
                Caption status
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.length === 0 ? (
              <tr>
                <td colSpan={3} className="py-4 text-slate-500 dark:text-neutral-400">
                  All videos have captions — great job.
                </td>
              </tr>
            ) : (
              rows.map((r) => (
                <tr key={r.object_id} className="border-b border-slate-100 dark:border-neutral-800">
                  <td className="py-2 pe-4 font-mono text-xs">{r.object_key}</td>
                  <td className="py-2 pe-4">{r.mime_type}</td>
                  <td className="py-2">
                    {r.caption_status ?? 'missing'}
                    {r.caption_id ? (
                      <>
                        {' '}
                        <Link
                          to={`/admin/caption-compliance?object=${r.object_id}`}
                          className="text-indigo-600 underline dark:text-indigo-300"
                        >
                          Edit
                        </Link>
                      </>
                    ) : null}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      ) : null}
    </main>
  )
}
