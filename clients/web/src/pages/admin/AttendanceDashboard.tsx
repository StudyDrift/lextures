import { useEffect, useState } from 'react'
import { fetchAttendanceDashboard, type DashboardEntry } from '../../lib/attendance-api'

function today(): string {
  return new Date().toISOString().slice(0, 10)
}

export default function AttendanceDashboard() {
  const [unitId, setUnitId] = useState<string>('')
  const [date, setDate] = useState<string>(today())
  const [entries, setEntries] = useState<DashboardEntry[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const loadDashboard = () => {
    if (!unitId.trim()) return
    setLoading(true)
    setError(null)
    void (async () => {
      try {
        const data = await fetchAttendanceDashboard(unitId.trim(), date)
        setEntries(data.entries)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load dashboard.')
      } finally {
        setLoading(false)
      }
    })()
  }

  useEffect(() => {
    if (unitId.trim()) loadDashboard()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [unitId, date])

  const absenceRate = (e: DashboardEntry): string => {
    if (e.totalStudents === 0) return '—'
    const pct = ((e.absentCount / e.totalStudents) * 100).toFixed(0)
    return `${pct}%`
  }

  return (
    <div className="p-4 max-w-5xl mx-auto">
      <h1 className="text-xl font-semibold mb-4">Attendance Dashboard</h1>

      <div className="flex flex-wrap gap-3 mb-4">
        <div>
          <label htmlFor="unit-id-input" className="block text-sm font-medium mb-1">
            School (Org Unit ID)
          </label>
          <input
            id="unit-id-input"
            type="text"
            placeholder="Paste org unit UUID"
            value={unitId}
            onChange={(e) => setUnitId(e.target.value)}
            className="border rounded px-2 py-1 text-sm w-80"
          />
        </div>
        <div>
          <label htmlFor="dash-date" className="block text-sm font-medium mb-1">
            Date
          </label>
          <input
            id="dash-date"
            type="date"
            value={date}
            max={today()}
            onChange={(e) => setDate(e.target.value)}
            className="border rounded px-2 py-1 text-sm"
          />
        </div>
      </div>

      {error && (
        <div role="alert" className="mb-3 text-red-600 text-sm">
          {error}
        </div>
      )}

      {loading && (
        <p className="text-sm text-gray-500" aria-busy="true">
          Loading…
        </p>
      )}

      {!loading && entries !== null && (
        <div className="border rounded overflow-hidden">
          <table className="w-full text-sm" aria-label="School attendance summary">
            <thead className="bg-gray-50">
              <tr>
                <th scope="col" className="px-3 py-2 text-start font-medium">Section</th>
                <th scope="col" className="px-3 py-2 text-start font-medium">Course</th>
                <th scope="col" className="px-3 py-2 text-end font-medium">Total</th>
                <th scope="col" className="px-3 py-2 text-end font-medium">Present</th>
                <th scope="col" className="px-3 py-2 text-end font-medium">Absent</th>
                <th scope="col" className="px-3 py-2 text-end font-medium">Tardy</th>
                <th scope="col" className="px-3 py-2 text-end font-medium">Absence%</th>
                <th scope="col" className="px-3 py-2 text-start font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {entries.length === 0 ? (
                <tr>
                  <td colSpan={8} className="px-3 py-4 text-center text-gray-500">
                    No sections found for this school.
                  </td>
                </tr>
              ) : (
                entries.map((e) => (
                  <tr key={e.sectionId} className="border-t hover:bg-gray-50">
                    <td className="px-3 py-2 font-mono">{e.sectionCode}</td>
                    <td className="px-3 py-2">{e.courseName}</td>
                    <td className="px-3 py-2 text-end">{e.totalStudents}</td>
                    <td className="px-3 py-2 text-end text-green-700">{e.presentCount}</td>
                    <td className="px-3 py-2 text-end text-red-600">{e.absentCount}</td>
                    <td className="px-3 py-2 text-end text-yellow-600">{e.tardyCount}</td>
                    <td className="px-3 py-2 text-end">{absenceRate(e)}</td>
                    <td className="px-3 py-2">
                      {e.notTaken ? (
                        <span className="inline-block px-2 py-0.5 rounded text-xs bg-orange-100 text-orange-700">
                          Not taken
                        </span>
                      ) : (
                        <span className="inline-block px-2 py-0.5 rounded text-xs bg-green-100 text-green-700">
                          Taken
                        </span>
                      )}
                    </td>
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
