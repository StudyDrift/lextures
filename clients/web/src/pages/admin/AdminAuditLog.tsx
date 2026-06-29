import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { fetchAdminAuditLog, type AuditEvent } from '../../lib/admin-console-api'

export default function AdminAuditLog() {
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [action, setAction] = useState('')
  const [events, setEvents] = useState<AuditEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetchAdminAuditLog({ orgId, action: action || undefined })
      setEvents(res.events)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load audit log.')
    } finally {
      setLoading(false)
    }
  }, [orgId, action])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <div>
      <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        Audit log
      </h1>

      <label className="mt-4 flex flex-col text-sm">
        <span className="mb-1 text-slate-600 dark:text-slate-400">Filter by action type</span>
        <input
          type="search"
          value={action}
          onChange={(e) => setAction(e.target.value)}
          placeholder="e.g. user_deactivate"
          className="max-w-xs rounded-lg border border-slate-300 px-3 py-2 dark:border-neutral-700 dark:bg-neutral-900"
        />
      </label>

      {error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      <div className="mt-4 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
        <table className="min-w-full text-left text-sm">
          <caption className="sr-only">Administrative audit log</caption>
          <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-slate-400">
            <tr>
              <th scope="col" className="px-4 py-2 font-medium">
                Time
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Event
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Actor
              </th>
              <th scope="col" className="px-4 py-2 font-medium">
                Target
              </th>
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-slate-500">
                  Loading…
                </td>
              </tr>
            ) : events.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-8 text-center text-slate-500">
                  No audit events found.
                </td>
              </tr>
            ) : (
              events.map((e) => (
                <tr key={e.eventId} className="border-t border-slate-100 dark:border-neutral-800">
                  <td className="px-4 py-2 whitespace-nowrap">
                    {new Date(e.timestamp).toLocaleString()}
                  </td>
                  <td className="px-4 py-2">{e.eventType}</td>
                  <td className="px-4 py-2 font-mono text-xs">{e.actorId}</td>
                  <td className="px-4 py-2">
                    {e.targetType ?? '—'}
                    {e.targetId ? ` / ${e.targetId}` : ''}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}
