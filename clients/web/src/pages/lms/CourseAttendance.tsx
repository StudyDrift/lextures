import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  fetchSectionAttendance,
  saveSectionAttendance,
  type AttendanceCode,
  type RosterEntry,
} from '../../lib/attendance-api'
import { authorizedFetch } from '../../lib/api'

type Section = { id: string; sectionCode: string; name?: string | null }

function today(): string {
  return new Date().toISOString().slice(0, 10)
}

function studentLabel(s: RosterEntry): string {
  return s.displayName?.trim() || s.email
}

export default function CourseAttendance() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const [sections, setSections] = useState<Section[] | null>(null)
  const [selectedSection, setSelectedSection] = useState<string>('')
  const [date, setDate] = useState<string>(today())
  const [roster, setRoster] = useState<RosterEntry[]>([])
  const [codes, setCodes] = useState<AttendanceCode[]>([])
  const [draftMap, setDraftMap] = useState<Record<string, string>>({})
  const [noteMap, setNoteMap] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveMsg, setSaveMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const announceRef = useRef<HTMLDivElement>(null)

  // Load sections for this course
  useEffect(() => {
    if (!courseCode) return
    void (async () => {
      try {
        const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/sections`)
        if (!res.ok) return
        const body = (await res.json()) as { sections?: Section[] }
        const list = body.sections ?? []
        setSections(list)
        if (list.length > 0) setSelectedSection(list[0].id)
      } catch {
        // sections may not be enabled; ignore
      }
    })()
  }, [courseCode])

  // Load attendance when section/date changes
  useEffect(() => {
    if (!selectedSection) return
    setLoading(true)
    setError(null)
    void (async () => {
      try {
        const data = await fetchSectionAttendance(selectedSection, date)
        setRoster(data.roster)
        setCodes(data.codes)
        // Build draft from existing records
        const map: Record<string, string> = {}
        const notes: Record<string, string> = {}
        for (const r of data.records) {
          map[r.studentId] = r.codeId
          if (r.note) notes[r.studentId] = r.note
        }
        setDraftMap(map)
        setNoteMap(notes)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load attendance.')
      } finally {
        setLoading(false)
      }
    })()
  }, [selectedSection, date])

  const presentCode = codes.find((c) => c.category === 'present')

  const handleMarkAllPresent = useCallback(() => {
    if (!presentCode) return
    const next: Record<string, string> = {}
    for (const s of roster) {
      next[s.userId] = presentCode.id
    }
    setDraftMap(next)
    announce('All students marked present.')
  }, [roster, presentCode])

  function announce(msg: string) {
    if (announceRef.current) {
      announceRef.current.textContent = msg
    }
  }

  const handleCodeChange = useCallback((studentId: string, codeId: string) => {
    setDraftMap((prev) => ({ ...prev, [studentId]: codeId }))
    const codeName = codes.find((c) => c.id === codeId)?.label ?? codeId
    announce(`Status changed to ${codeName}.`)
  }, [codes])

  const handleSave = useCallback(async () => {
    if (!selectedSection) return
    setSaving(true)
    setSaveMsg(null)
    setError(null)
    try {
      const records = roster.map((s) => ({
        studentId: s.userId,
        codeId: draftMap[s.userId] ?? (presentCode?.id ?? ''),
        note: noteMap[s.userId] ?? null,
      }))
      const result = await saveSectionAttendance(selectedSection, date, records)
      setSaveMsg(result.message)
      announce(result.message)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Save failed.')
    } finally {
      setSaving(false)
    }
  }, [selectedSection, date, roster, draftMap, noteMap, presentCode])

  // Handle keyboard navigation in the grid
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLSelectElement>, studentId: string) => {
      if (e.key === ' ' || e.key === 'Enter') {
        e.preventDefault()
        // Cycle to next code
        const currentIdx = codes.findIndex((c) => c.id === draftMap[studentId])
        const nextCode = codes[(currentIdx + 1) % codes.length]
        if (nextCode) handleCodeChange(studentId, nextCode.id)
      }
    },
    [codes, draftMap, handleCodeChange],
  )

  if (!courseCode) {
    return <p className="p-4 text-red-600">No course selected.</p>
  }

  return (
    <div className="p-4 max-w-4xl mx-auto">
      {/* Live region for screen reader announcements */}
      <div
        ref={announceRef}
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      />

      <h1 className="text-xl font-semibold mb-4">Attendance — {courseCode}</h1>

      {/* Controls */}
      <div className="flex flex-wrap gap-3 mb-4">
        {sections && sections.length > 0 && (
          <div>
            <label htmlFor="section-select" className="block text-sm font-medium mb-1">
              Section
            </label>
            <select
              id="section-select"
              value={selectedSection}
              onChange={(e) => setSelectedSection(e.target.value)}
              className="border rounded px-2 py-1 text-sm"
            >
              {sections.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.name ?? s.sectionCode}
                </option>
              ))}
            </select>
          </div>
        )}

        <div>
          <label htmlFor="attendance-date" className="block text-sm font-medium mb-1">
            Date
          </label>
          <input
            id="attendance-date"
            type="date"
            value={date}
            max={today()}
            onChange={(e) => setDate(e.target.value)}
            className="border rounded px-2 py-1 text-sm"
          />
        </div>

        <div className="flex items-end">
          <button
            onClick={handleMarkAllPresent}
            disabled={!presentCode || roster.length === 0}
            className="px-3 py-1 text-sm bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
            aria-label="Mark all students present"
          >
            Mark all present
          </button>
        </div>
      </div>

      {error && (
        <div role="alert" className="mb-3 text-red-600 text-sm">
          {error}
        </div>
      )}
      {saveMsg && (
        <div role="status" className="mb-3 text-green-700 text-sm font-medium">
          {saveMsg}
        </div>
      )}

      {loading ? (
        <p className="text-sm text-gray-500" aria-busy="true">
          Loading…
        </p>
      ) : roster.length === 0 ? (
        <p className="text-sm text-gray-500">
          {selectedSection ? 'No students enrolled in this section.' : 'Select a section to take attendance.'}
        </p>
      ) : (
        <>
          {/* Roll-taking grid */}
          <div
            role="grid"
            aria-label="Attendance roll"
            aria-rowcount={roster.length + 1}
            className="border rounded overflow-hidden mb-4"
          >
            {/* Header row */}
            <div
              role="row"
              className="grid grid-cols-[1fr_180px] bg-gray-50 border-b font-medium text-sm"
            >
              <div role="columnheader" className="px-3 py-2">
                Student
              </div>
              <div role="columnheader" className="px-3 py-2">
                Status
              </div>
            </div>

            {/* Student rows */}
            {roster.map((student, idx) => {
              const selectedCode = draftMap[student.userId]
              const codeObj = codes.find((c) => c.id === selectedCode)
              const rowClass = [
                'grid grid-cols-[1fr_180px] border-b last:border-0 text-sm',
                codeObj?.category === 'absent' ? 'bg-red-50' :
                codeObj?.category === 'tardy' ? 'bg-yellow-50' :
                codeObj?.category === 'present' ? 'bg-green-50' : '',
              ].join(' ')

              return (
                <div
                  key={student.userId}
                  role="row"
                  aria-rowindex={idx + 2}
                  className={rowClass}
                >
                  <div role="gridcell" className="px-3 py-2">
                    {studentLabel(student)}
                  </div>
                  <div role="gridcell" className="px-3 py-2">
                    {codes.length === 0 ? (
                      <span className="text-gray-400 text-xs">No codes configured</span>
                    ) : (
                      <select
                        value={selectedCode ?? ''}
                        onChange={(e) => handleCodeChange(student.userId, e.target.value)}
                        onKeyDown={(e) => handleKeyDown(e, student.userId)}
                        aria-label={`Attendance status for ${studentLabel(student)}`}
                        className="w-full border rounded px-2 py-0.5 text-sm"
                      >
                        <option value="">— not taken —</option>
                        {codes.map((c) => (
                          <option key={c.id} value={c.id}>
                            {c.code} – {c.label}
                          </option>
                        ))}
                      </select>
                    )}
                  </div>
                </div>
              )
            })}
          </div>

          <button
            onClick={handleSave}
            disabled={saving || codes.length === 0}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 font-medium"
            aria-label="Save attendance"
          >
            {saving ? 'Saving…' : 'Save attendance'}
          </button>
        </>
      )}
    </div>
  )
}
