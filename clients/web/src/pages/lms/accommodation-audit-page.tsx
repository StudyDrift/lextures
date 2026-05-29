import { useCallback, useEffect, useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { usePermissions } from '../../context/use-permissions'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAccommodationAuditLog,
  type AccommodationAuditEntry,
} from '../../lib/courses-api'
import { PERM_ACCOMMODATIONS_MANAGE } from '../../lib/rbac-api'
import { LmsPage } from './lms-page'

function formatValueApplied(value: unknown): string {
  if (value == null) return '—'
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

export default function AccommodationAuditPage() {
  const filterId = useId()
  const { allows, loading: permLoading } = usePermissions()
  const { accommodationsEngineEnabled, loading: featuresLoading } = usePlatformFeatures()
  const canManage = !permLoading && allows(PERM_ACCOMMODATIONS_MANAGE)

  const [studentIdFilter, setStudentIdFilter] = useState('')
  const [rows, setRows] = useState<AccommodationAuditEntry[]>([])
  const [listBusy, setListBusy] = useState(false)
  const [listError, setListError] = useState<string | null>(null)
  const [disabledMessage, setDisabledMessage] = useState<string | null>(null)

  const loadLog = useCallback(async () => {
    setListBusy(true)
    setListError(null)
    setDisabledMessage(null)
    try {
      const data = await fetchAccommodationAuditLog({
        studentId: studentIdFilter.trim() || undefined,
        limit: 200,
      })
      setRows(data)
    } catch (e) {
      setRows([])
      const msg = e instanceof Error ? e.message : 'Could not load audit log.'
      if (/not enabled/i.test(msg)) {
        setDisabledMessage('Accommodations engine is not enabled on this platform.')
      } else {
        setListError(msg)
      }
    } finally {
      setListBusy(false)
    }
  }, [studentIdFilter])

  useEffect(() => {
    if (!canManage || featuresLoading) return
    if (!accommodationsEngineEnabled) {
      setDisabledMessage('Accommodations engine is not enabled on this platform.')
      setRows([])
      return
    }
    void loadLog()
  }, [accommodationsEngineEnabled, canManage, featuresLoading, loadLog])

  if (!canManage) {
    return (
      <LmsPage title="Accommodation audit">
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          You need the accessibility coordinator or Global Admin role to view accommodation audit records.
        </p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Accommodation audit report">
      <div className="max-w-5xl space-y-6">
        <p className="text-sm text-slate-600 dark:text-neutral-300">
          Append-only log of accommodations applied during quiz attempts and content views. Student identifiers
          are shown as user IDs only (no names) for FERPA-aligned compliance reporting.
        </p>
        <p className="text-sm">
          <Link
            to="/admin/accommodations"
            className="font-medium text-indigo-700 hover:underline dark:text-indigo-300"
          >
            ← Student accommodations
          </Link>
        </p>

        {disabledMessage && (
          <p role="alert" className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100">
            {disabledMessage}
          </p>
        )}

        {!disabledMessage && (
          <div className="rounded-xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
            <label
              htmlFor={`${filterId}-student`}
              className="mb-1 block text-xs font-medium text-slate-600 dark:text-neutral-400"
            >
              Filter by student user id (optional)
            </label>
            <div className="flex flex-wrap gap-2">
              <input
                id={`${filterId}-student`}
                value={studentIdFilter}
                onChange={(e) => setStudentIdFilter(e.target.value)}
                className="min-w-[12rem] flex-1 rounded-lg border border-slate-200 px-3 py-2 font-mono text-sm dark:border-neutral-600 dark:bg-neutral-950"
                placeholder="UUID"
                spellCheck={false}
                autoComplete="off"
              />
              <button
                type="button"
                onClick={() => void loadLog()}
                disabled={listBusy}
                className="rounded-lg bg-slate-800 px-3 py-2 text-sm font-semibold text-white hover:bg-slate-700 disabled:opacity-50 dark:bg-neutral-200 dark:text-neutral-900 dark:hover:bg-white"
              >
                {listBusy ? 'Loading…' : 'Refresh'}
              </button>
            </div>
            {listError && (
              <p role="alert" className="mt-2 text-sm text-rose-700 dark:text-rose-300">
                {listError}
              </p>
            )}
          </div>
        )}

        {!disabledMessage && rows.length === 0 && !listBusy && !listError && (
          <p className="text-sm text-slate-500 dark:text-neutral-400">No audit entries yet.</p>
        )}

        {rows.length > 0 && (
          <div className="overflow-x-auto rounded-xl border border-slate-200 bg-white dark:border-neutral-700 dark:bg-neutral-900">
            <table className="min-w-full text-start text-sm">
              <thead className="border-b border-slate-200 bg-slate-50 text-xs font-semibold uppercase text-slate-600 dark:border-neutral-700 dark:bg-neutral-950 dark:text-neutral-400">
                <tr>
                  <th className="px-3 py-2">Applied at</th>
                  <th className="px-3 py-2">Student id</th>
                  <th className="px-3 py-2">Type</th>
                  <th className="px-3 py-2">Value</th>
                  <th className="px-3 py-2">Context</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.id} className="border-b border-slate-100 dark:border-neutral-800">
                    <td className="whitespace-nowrap px-3 py-2 text-xs text-slate-600 dark:text-neutral-400">
                      {r.appliedAt}
                    </td>
                    <td className="px-3 py-2 font-mono text-xs text-slate-800 dark:text-neutral-200">
                      {r.studentId}
                    </td>
                    <td className="px-3 py-2 text-slate-800 dark:text-neutral-200">{r.accommodationType}</td>
                    <td className="max-w-[14rem] truncate px-3 py-2 font-mono text-xs text-slate-600 dark:text-neutral-400">
                      {formatValueApplied(r.valueApplied)}
                    </td>
                    <td className="px-3 py-2 text-xs text-slate-600 dark:text-neutral-400">
                      {r.context}
                      {r.contextId ? (
                        <span className="mt-0.5 block font-mono text-[11px]">{r.contextId}</span>
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </LmsPage>
  )
}
