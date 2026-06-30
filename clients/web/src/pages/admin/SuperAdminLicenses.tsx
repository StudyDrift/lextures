import { useCallback, useEffect, useState } from 'react'
import { SeatUtilizationBar } from '../../components/admin/seat-utilization'
import {
  fetchAdminLicenses,
  patchOrgLicense,
  type OrgLicense,
} from '../../lib/admin-console-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { Navigate } from 'react-router-dom'

export default function SuperAdminLicenses() {
  const { seatManagementEnabled } = usePlatformFeatures()
  const [items, setItems] = useState<OrgLicense[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [savingOrgId, setSavingOrgId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetchAdminLicenses()
      setItems(res.items ?? [])
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load licenses.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (seatManagementEnabled) void load()
  }, [load, seatManagementEnabled])

  if (!seatManagementEnabled) {
    return <Navigate to="/" replace />
  }

  async function saveMaxSeats(orgId: string, maxSeats: number) {
    setSavingOrgId(orgId)
    setError(null)
    try {
      const updated = await patchOrgLicense(orgId, { maxSeats })
      setItems((prev) => prev.map((row) => (row.orgId === orgId ? { ...row, ...updated } : row)))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update license.')
    } finally {
      setSavingOrgId(null)
    }
  }

  return (
    <div className="mx-auto max-w-6xl p-6">
      <h1 className="text-xl font-semibold text-slate-900 dark:text-slate-100">Seat licenses</h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        Manage seat limits and contract metadata for all organizations.
      </p>

      {error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      {loading ? (
        <p className="mt-6 text-sm text-slate-500">Loading…</p>
      ) : (
        <div className="mt-6 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
          <table className="min-w-full text-left text-sm">
            <caption className="sr-only">Organization seat licenses</caption>
            <thead className="bg-slate-50 text-slate-600 dark:bg-neutral-950 dark:text-slate-400">
              <tr>
                <th scope="col" className="px-4 py-2 font-medium">
                  Organization
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Tier
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Utilization
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Max seats
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  Contract end
                </th>
              </tr>
            </thead>
            <tbody>
              {items.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-slate-500">
                    No license records yet. Patch an organization to create one.
                  </td>
                </tr>
              ) : (
                items.map((row) => (
                  <tr key={row.orgId} className="border-t border-slate-100 dark:border-neutral-800">
                    <td className="px-4 py-3">
                      <div className="font-medium text-slate-900 dark:text-slate-100">
                        {row.orgName ?? row.orgId}
                      </div>
                      {row.orgSlug ? (
                        <div className="text-xs text-slate-500">{row.orgSlug}</div>
                      ) : null}
                    </td>
                    <td className="px-4 py-3 capitalize">{row.tier}</td>
                    <td className="min-w-[12rem] px-4 py-3">
                      <SeatUtilizationBar license={row} />
                    </td>
                    <td className="px-4 py-3">
                      <form
                        className="flex items-center gap-2"
                        onSubmit={(e) => {
                          e.preventDefault()
                          const fd = new FormData(e.currentTarget)
                          const raw = String(fd.get('maxSeats') ?? '').trim()
                          const n = raw === '' || raw === 'unlimited' ? -1 : Number(raw)
                          if (!Number.isFinite(n)) return
                          void saveMaxSeats(row.orgId, n)
                        }}
                      >
                        <label className="sr-only" htmlFor={`max-${row.orgId}`}>
                          Max seats for {row.orgName ?? row.orgId}
                        </label>
                        <input
                          id={`max-${row.orgId}`}
                          name="maxSeats"
                          type="number"
                          min={-1}
                          defaultValue={row.maxSeats}
                          className="w-24 rounded border border-slate-300 px-2 py-1 dark:border-neutral-700 dark:bg-neutral-900"
                        />
                        <button
                          type="submit"
                          disabled={savingOrgId === row.orgId}
                          className="rounded bg-indigo-600 px-2 py-1 text-xs font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
                        >
                          Save
                        </button>
                      </form>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">{row.contractEnd ?? '—'}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
