import { useState } from 'react'
import { exportAttendance } from '../../lib/attendance-api'

function firstDayOfMonth(): string {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-01`
}

function today(): string {
  return new Date().toISOString().slice(0, 10)
}

export default function AttendanceExport() {
  const [orgId, setOrgId] = useState('')
  const [startDate, setStartDate] = useState(firstDayOfMonth())
  const [endDate, setEndDate] = useState(today())
  const [format, setFormat] = useState<'csv' | 'calpads'>('csv')
  const [exporting, setExporting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  const handleExport = async () => {
    if (!orgId.trim()) {
      setError('Organization ID is required.')
      return
    }
    setExporting(true)
    setError(null)
    setSuccess(false)
    try {
      const blob = await exportAttendance(orgId.trim(), startDate, endDate, format)
      const filename = format === 'calpads'
        ? `calpads_attendance_${startDate}_${endDate}.csv`
        : `attendance_${startDate}_${endDate}.csv`
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      setSuccess(true)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Export failed.')
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="p-4 max-w-2xl mx-auto">
      <h1 className="text-xl font-semibold mb-4">Attendance Export</h1>

      <div className="space-y-4">
        <div>
          <label htmlFor="export-org-id" className="block text-sm font-medium mb-1">
            Organization ID
          </label>
          <input
            id="export-org-id"
            type="text"
            placeholder="Paste organization UUID"
            value={orgId}
            onChange={(e) => setOrgId(e.target.value)}
            className="border rounded px-2 py-1 text-sm w-full max-w-md"
          />
        </div>

        <div className="flex gap-4">
          <div>
            <label htmlFor="export-start" className="block text-sm font-medium mb-1">
              Start date
            </label>
            <input
              id="export-start"
              type="date"
              value={startDate}
              onChange={(e) => setStartDate(e.target.value)}
              className="border rounded px-2 py-1 text-sm"
            />
          </div>
          <div>
            <label htmlFor="export-end" className="block text-sm font-medium mb-1">
              End date
            </label>
            <input
              id="export-end"
              type="date"
              value={endDate}
              max={today()}
              onChange={(e) => setEndDate(e.target.value)}
              className="border rounded px-2 py-1 text-sm"
            />
          </div>
        </div>

        <fieldset>
          <legend className="text-sm font-medium mb-1">Export format</legend>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="radio"
                name="format"
                value="csv"
                checked={format === 'csv'}
                onChange={() => setFormat('csv')}
              />
              Generic CSV
            </label>
            <label className="flex items-center gap-2 text-sm cursor-pointer">
              <input
                type="radio"
                name="format"
                value="calpads"
                checked={format === 'calpads'}
                onChange={() => setFormat('calpads')}
              />
              CALPADS SB format (California)
            </label>
          </div>
        </fieldset>

        {format === 'calpads' && (
          <p className="text-xs text-gray-500 bg-yellow-50 border border-yellow-200 rounded p-2">
            CALPADS export includes SSID, attendance codes, and state-specific code mapping.
            Verify that your attendance codes have CALPADS state codes configured before exporting.
          </p>
        )}

        {error && (
          <div role="alert" className="text-red-600 text-sm">
            {error}
          </div>
        )}
        {success && (
          <div role="status" className="text-green-700 text-sm font-medium">
            Export downloaded successfully.
          </div>
        )}

        <button
          onClick={handleExport}
          disabled={exporting || !orgId.trim()}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 font-medium"
          aria-label="Download attendance export"
        >
          {exporting ? 'Exporting…' : 'Download export'}
        </button>
      </div>
    </div>
  )
}
