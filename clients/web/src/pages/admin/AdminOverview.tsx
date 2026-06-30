import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { BookOpen, HardDrive, UserPlus, Users } from 'lucide-react'
import { LicenseDetailsCard } from '../../components/admin/seat-utilization'
import {
  fetchAdminOverview,
  formatStorageBytes,
  type AdminOverview,
  type AuditEvent,
} from '../../lib/admin-console-api'

function KpiCard({
  label,
  value,
  icon: Icon,
}: {
  label: string
  value: string | number
  icon: typeof Users
}) {
  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
      <div className="flex items-start justify-between gap-2">
        <div>
          <p className="text-sm text-slate-500 dark:text-slate-400">{label}</p>
          <p className="mt-1 text-2xl font-semibold tabular-nums text-slate-900 dark:text-slate-100">
            {value}
          </p>
        </div>
        <Icon className="h-5 w-5 text-indigo-500" aria-hidden />
      </div>
    </div>
  )
}

export default function AdminOverview() {
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [data, setData] = useState<AdminOverview | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setData(await fetchAdminOverview(orgId))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load overview.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void load()
  }, [load])

  const contractBanner =
    data?.license?.contractExpiringSoon && data.license.contractEnd ? (
      <div
        role="status"
        className="mt-4 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100"
      >
        Your license contract ends on {data.license.contractEnd}. Contact your Lextures representative to
        renew before access is affected.
      </div>
    ) : null

  return (
    <div>
      <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        Overview
      </h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        Organization utilization and recent administrative activity.
      </p>

      {contractBanner}

      {error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      {loading ? (
        <p className="mt-6 text-sm text-slate-500">Loading…</p>
      ) : data ? (
        <>
          <div className="mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <KpiCard label="Total users" value={data.totalUsers} icon={Users} />
            <KpiCard label="Active courses" value={data.activeCourses} icon={BookOpen} />
            <KpiCard label="Pending enrollments" value={data.pendingEnrollments} icon={UserPlus} />
            <KpiCard
              label="Storage used"
              value={formatStorageBytes(data.storageBytes)}
              icon={HardDrive}
            />
          </div>

          {data.license ? (
            <div className="mt-6">
              <LicenseDetailsCard license={data.license} />
            </div>
          ) : null}

          <section className="mt-8" aria-labelledby={`${titleId}-audit`}>
            <h2 id={`${titleId}-audit`} className="text-base font-semibold text-slate-900 dark:text-slate-100">
              Recent audit events
            </h2>
            <div className="mt-3 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
              <table className="min-w-full text-left text-sm">
                <caption className="sr-only">Recent administrative audit log entries</caption>
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
                  {data.recentAuditEvents.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="px-4 py-6 text-center text-slate-500">
                        No recent audit events.
                      </td>
                    </tr>
                  ) : (
                    data.recentAuditEvents.map((e: AuditEvent) => (
                      <tr
                        key={e.eventId}
                        className="border-t border-slate-100 dark:border-neutral-800"
                      >
                        <td className="px-4 py-2 whitespace-nowrap">
                          {new Date(e.timestamp).toLocaleString()}
                        </td>
                        <td className="px-4 py-2">{e.eventType}</td>
                        <td className="px-4 py-2 font-mono text-xs">{e.actorId.slice(0, 8)}…</td>
                        <td className="px-4 py-2">
                          {e.targetType ?? '—'}
                          {e.targetId ? ` (${e.targetId.slice(0, 8)}…)` : ''}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </section>
        </>
      ) : null}
    </div>
  )
}
