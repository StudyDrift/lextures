import { useCallback, useState } from 'react'
import { LayoutGrid } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { formatConferenceSlotTime, listSchoolConferenceSchedule, type ScheduleEntry } from '../../lib/conferences-api'

function statusClass(status: ScheduleEntry['status']): string {
  switch (status) {
    case 'open':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300'
    case 'booked':
      return 'bg-sky-100 text-sky-800 dark:bg-sky-900/40 dark:text-sky-300'
    case 'cancelled':
      return 'bg-rose-100 text-rose-800 dark:bg-rose-900/40 dark:text-rose-300'
  }
}

export default function ConferenceScheduleGrid() {
  const { ffConferenceScheduling } = usePlatformFeatures()
  const [orgUnitId, setOrgUnitId] = useState('')
  const [date, setDate] = useState('')
  const [schedule, setSchedule] = useState<ScheduleEntry[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const load = useCallback(async () => {
    if (!orgUnitId.trim() || !date) return
    setLoading(true)
    setError(null)
    try {
      const rows = await listSchoolConferenceSchedule(orgUnitId.trim(), date)
      setSchedule(rows)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load schedule.')
      setSchedule(null)
    } finally {
      setLoading(false)
    }
  }, [orgUnitId, date])

  if (!ffConferenceScheduling) {
    return (
      <div className="mx-auto max-w-4xl px-4 py-10">
        <p className="text-sm text-neutral-600 dark:text-neutral-400">
          Conference scheduling is not enabled on this platform.
        </p>
      </div>
    )
  }

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-4 py-8 md:px-8">
      <header className="flex flex-col gap-2 border-b border-slate-200 pb-6 dark:border-neutral-800">
        <div className="flex items-center gap-2 text-sm font-medium text-indigo-700 dark:text-indigo-300">
          <LayoutGrid className="h-4 w-4" aria-hidden />
          Admin
        </div>
        <h1 className="text-2xl font-semibold tracking-tight">Conference schedule grid</h1>
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          View all teacher bookings and room assignments for a conference day.
        </p>
      </header>

      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">School org unit ID</span>
          <input
            type="text"
            value={orgUnitId}
            onChange={(e) => setOrgUnitId(e.target.value)}
            className="min-w-[280px] rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          <span className="font-medium">Date</span>
          <input
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="rounded-lg border border-neutral-300 px-3 py-2 dark:border-neutral-600 dark:bg-neutral-900"
          />
        </label>
        <button
          type="button"
          onClick={() => void load()}
          disabled={loading}
          className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-60"
        >
          {loading ? 'Loading…' : 'Load grid'}
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-900">{error}</div>
      )}

      {schedule && (
        <div className="overflow-x-auto rounded-xl border border-neutral-200 dark:border-neutral-700">
          <table className="min-w-full text-left text-sm">
            <thead className="bg-neutral-50 text-neutral-600 dark:bg-neutral-900 dark:text-neutral-300">
              <tr>
                <th className="px-4 py-3 font-medium">Teacher</th>
                <th className="px-4 py-3 font-medium">Time</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Student</th>
                <th className="px-4 py-3 font-medium">Location</th>
              </tr>
            </thead>
            <tbody>
              {schedule.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-6 text-neutral-500">
                    No slots for this date.
                  </td>
                </tr>
              ) : (
                schedule.map((row) => (
                  <tr key={row.id} className="border-t border-neutral-200 dark:border-neutral-800">
                    <td className="px-4 py-3">{row.teacherDisplayName ?? 'Teacher'}</td>
                    <td className="px-4 py-3">
                      <time dateTime={row.startAt}>{formatConferenceSlotTime(row)}</time>
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex rounded-full px-2 py-0.5 text-xs font-semibold ${statusClass(row.status)}`}>
                        {row.status}
                      </span>
                    </td>
                    <td className="px-4 py-3">{row.childDisplayName ?? '—'}</td>
                    <td className="px-4 py-3">{row.videoLink || row.location || '—'}</td>
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
