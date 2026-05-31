import { useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import {
  awardPBISPoints,
  fileBehaviorReferral,
  listBehaviorCategories,
  type BehaviorCategory,
} from '../../lib/behavior-api'
import { authorizedFetch } from '../../lib/api'

type RosterEntry = { userId: string; email: string; displayName?: string | null }

function studentLabel(s: RosterEntry): string {
  return s.displayName?.trim() || s.email
}

export default function CourseBehavior() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const [orgId, setOrgId] = useState<string | null>(null)
  const [categories, setCategories] = useState<BehaviorCategory[]>([])
  const [roster, setRoster] = useState<RosterEntry[]>([])
  const [selectedStudents, setSelectedStudents] = useState<Set<string>>(new Set())
  const [selectedCategory, setSelectedCategory] = useState<string>('')
  const [awardNote, setAwardNote] = useState<string>('')
  const [saving, setSaving] = useState(false)
  const [saveMsg, setSaveMsg] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'award' | 'referral'>('award')

  // Referral form state
  const [refStudent, setRefStudent] = useState<string>('')
  const [refCategory, setRefCategory] = useState<string>('')
  const [refDescription, setRefDescription] = useState<string>('')
  const [refLocation, setRefLocation] = useState<string>('')
  const [refResponse, setRefResponse] = useState<string>('')
  const [refSaving, setRefSaving] = useState(false)
  const [refMsg, setRefMsg] = useState<string | null>(null)
  const [refError, setRefError] = useState<string | null>(null)

  const announceRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/me')
        if (!res.ok) return
        const me = (await res.json()) as { orgId?: string }
        if (me.orgId) setOrgId(me.orgId)
      } catch {
        // ignore
      }
    })()
  }, [])

  useEffect(() => {
    if (!orgId) return
    void (async () => {
      try {
        const data = await listBehaviorCategories(orgId)
        setCategories(data.categories.filter((c) => c.active))
      } catch {
        // categories may not be seeded yet
      }
    })()
  }, [orgId])

  useEffect(() => {
    if (!courseCode) return
    void (async () => {
      try {
        const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments`)
        if (!res.ok) return
        const body = (await res.json()) as {
          enrollments?: Array<{ userId: string; email: string; displayName?: string | null; role: string }>
        }
        const students = (body.enrollments ?? [])
          .filter((e) => e.role === 'student' || e.role === 'learner')
          .map((e) => ({ userId: e.userId, email: e.email, displayName: e.displayName }))
        setRoster(students)
      } catch {
        // ignore
      }
    })()
  }, [courseCode])

  function toggleStudent(userId: string) {
    setSelectedStudents((prev) => {
      const next = new Set(prev)
      if (next.has(userId)) next.delete(userId)
      else next.add(userId)
      return next
    })
  }

  function selectAll() {
    setSelectedStudents(new Set(roster.map((s) => s.userId)))
  }

  function clearAll() {
    setSelectedStudents(new Set())
  }

  async function handleAward() {
    if (selectedStudents.size === 0) {
      setError('Select at least one student.')
      return
    }
    if (!selectedCategory) {
      setError('Select a behavior category.')
      return
    }
    setError(null)
    setSaving(true)
    setSaveMsg(null)
    try {
      const awards = Array.from(selectedStudents).map((sid) => ({
        studentId: sid,
        categoryId: selectedCategory,
        points: 1,
        note: awardNote.trim() || null,
      }))
      const result = await awardPBISPoints(awards)
      setSaveMsg(`Awarded points to ${result.saved} student${result.saved !== 1 ? 's' : ''}.`)
      setSelectedStudents(new Set())
      setAwardNote('')
      if (announceRef.current) {
        announceRef.current.textContent = `Success: awarded points to ${result.saved} students.`
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to award points.')
    } finally {
      setSaving(false)
    }
  }

  async function handleReferral() {
    if (!refStudent) { setRefError('Select a student.'); return }
    if (!refCategory) { setRefError('Select a behavior category.'); return }
    if (!refDescription.trim()) { setRefError('Description is required.'); return }
    setRefError(null)
    setRefSaving(true)
    setRefMsg(null)
    try {
      await fileBehaviorReferral({
        studentId: refStudent,
        categoryId: refCategory,
        location: refLocation.trim() || null,
        description: refDescription.trim(),
        response: refResponse.trim() || null,
      })
      setRefMsg('Referral filed successfully.')
      setRefStudent('')
      setRefCategory('')
      setRefDescription('')
      setRefLocation('')
      setRefResponse('')
      if (announceRef.current) {
        announceRef.current.textContent = 'Referral filed successfully.'
      }
    } catch (e) {
      setRefError(e instanceof Error ? e.message : 'Failed to file referral.')
    } finally {
      setRefSaving(false)
    }
  }

  const positiveCategories = categories.filter((c) => c.type === 'positive')
  const negativeCategories = categories.filter((c) => c.type === 'negative')

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <h1 className="text-2xl font-semibold mb-6">Behavior &amp; PBIS</h1>

      <div aria-live="polite" aria-atomic="true" className="sr-only" ref={announceRef} />

      <div role="tablist" className="flex gap-2 mb-6 border-b">
        <button
          role="tab"
          aria-selected={activeTab === 'award'}
          onClick={() => setActiveTab('award')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors cursor-pointer ${
            activeTab === 'award'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          Award PBIS Points
        </button>
        <button
          role="tab"
          aria-selected={activeTab === 'referral'}
          onClick={() => setActiveTab('referral')}
          className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors cursor-pointer ${
            activeTab === 'referral'
              ? 'border-blue-600 text-blue-600'
              : 'border-transparent text-gray-600 hover:text-gray-900'
          }`}
        >
          File Referral
        </button>
      </div>

      {activeTab === 'award' && (
        <section aria-label="Award PBIS points">
          {categories.length === 0 && (
            <p className="text-amber-700 bg-amber-50 border border-amber-200 rounded p-3 mb-4 text-sm">
              No behavior categories configured. Ask your administrator to set up PBIS categories.
            </p>
          )}

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1" htmlFor="pbis-category">
              Behavior Category <span aria-hidden="true">*</span>
            </label>
            <select
              id="pbis-category"
              value={selectedCategory}
              onChange={(e) => setSelectedCategory(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
            >
              <option value="">Select a positive category…</option>
              {positiveCategories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1" htmlFor="pbis-note">
              Note (optional)
            </label>
            <input
              id="pbis-note"
              type="text"
              value={awardNote}
              onChange={(e) => setAwardNote(e.target.value)}
              placeholder="e.g., Great participation today!"
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>

          <div className="mb-2 flex items-center justify-between">
            <span className="text-sm font-medium">Students ({selectedStudents.size} selected)</span>
            <span className="flex gap-2">
              <button
                onClick={selectAll}
                className="text-xs text-blue-600 hover:underline cursor-pointer"
              >
                Select all
              </button>
              <button
                onClick={clearAll}
                className="text-xs text-gray-500 hover:underline cursor-pointer"
              >
                Clear
              </button>
            </span>
          </div>

          {roster.length === 0 ? (
            <p className="text-gray-500 text-sm">No students enrolled in this course.</p>
          ) : (
            <ul
              className="border rounded divide-y max-h-64 overflow-y-auto mb-4"
              aria-label="Student roster"
            >
              {roster.map((s) => (
                <li key={s.userId}>
                  <label className="flex items-center gap-3 px-3 py-2 hover:bg-gray-50 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={selectedStudents.has(s.userId)}
                      onChange={() => toggleStudent(s.userId)}
                      className="rounded"
                    />
                    <span className="text-sm">{studentLabel(s)}</span>
                  </label>
                </li>
              ))}
            </ul>
          )}

          {error && (
            <p role="alert" className="text-red-600 text-sm mb-3">
              {error}
            </p>
          )}
          {saveMsg && (
            <p className="text-green-700 text-sm mb-3">{saveMsg}</p>
          )}

          <button
            onClick={handleAward}
            disabled={saving || selectedStudents.size === 0 || !selectedCategory}
            className="px-4 py-2 bg-blue-600 text-white rounded text-sm font-medium hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
          >
            {saving ? 'Awarding…' : 'Award Points'}
          </button>
        </section>
      )}

      {activeTab === 'referral' && (
        <section aria-label="File a behavior referral">
          <div className="mb-4">
            <label className="block text-sm font-medium mb-1" htmlFor="ref-student">
              Student <span aria-hidden="true">*</span>
            </label>
            <select
              id="ref-student"
              value={refStudent}
              onChange={(e) => setRefStudent(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
            >
              <option value="">Select a student…</option>
              {roster.map((s) => (
                <option key={s.userId} value={s.userId}>
                  {studentLabel(s)}
                </option>
              ))}
            </select>
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1" htmlFor="ref-category">
              Behavior Category <span aria-hidden="true">*</span>
            </label>
            <select
              id="ref-category"
              value={refCategory}
              onChange={(e) => setRefCategory(e.target.value)}
              className="w-full border rounded px-3 py-2 text-sm"
            >
              <option value="">Select a category…</option>
              <optgroup label="Negative">
                {negativeCategories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </optgroup>
              <optgroup label="Positive">
                {positiveCategories.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </optgroup>
            </select>
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1" htmlFor="ref-location">
              Location (optional)
            </label>
            <input
              id="ref-location"
              type="text"
              value={refLocation}
              onChange={(e) => setRefLocation(e.target.value)}
              placeholder="e.g., Classroom, Hallway"
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1" htmlFor="ref-description">
              Incident Description <span aria-hidden="true">*</span>
            </label>
            <textarea
              id="ref-description"
              value={refDescription}
              onChange={(e) => setRefDescription(e.target.value)}
              rows={3}
              placeholder="Describe the incident…"
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium mb-1" htmlFor="ref-response">
              Response Taken (optional)
            </label>
            <input
              id="ref-response"
              type="text"
              value={refResponse}
              onChange={(e) => setRefResponse(e.target.value)}
              placeholder="e.g., Verbal warning, parent contact"
              className="w-full border rounded px-3 py-2 text-sm"
            />
          </div>

          {refError && (
            <p role="alert" className="text-red-600 text-sm mb-3">
              {refError}
            </p>
          )}
          {refMsg && (
            <p className="text-green-700 text-sm mb-3">{refMsg}</p>
          )}

          <button
            onClick={handleReferral}
            disabled={refSaving || !refStudent || !refCategory || !refDescription.trim()}
            className="px-4 py-2 bg-red-600 text-white rounded text-sm font-medium hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer"
          >
            {refSaving ? 'Filing…' : 'File Referral'}
          </button>
        </section>
      )}
    </div>
  )
}
