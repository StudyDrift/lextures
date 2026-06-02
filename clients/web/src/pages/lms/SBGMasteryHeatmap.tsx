import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams } from 'react-router-dom'
import { authorizedFetch } from '../../lib/api'
import {
  getSBGHeatmap,
  listCourseStandards,
  getMasteryScale,
  recordMasteryScore,
  type HeatmapCell,
  type Standard,
  type MasteryScaleEntry,
} from '../../lib/sbg-api'

type RosterEntry = { userId: string; email: string; displayName?: string | null }

function studentLabel(s: RosterEntry): string {
  return s.displayName?.trim() || s.email
}

function currentQuarter(): string {
  const now = new Date()
  const q = Math.ceil((now.getMonth() + 1) / 3)
  return `Q${q}-${now.getFullYear()}`
}

export default function SBGMasteryHeatmap() {
  const { courseCode } = useParams<{ courseCode: string }>()
  const [period, setPeriod] = useState<string>(currentQuarter())
  const [standards, setStandards] = useState<Standard[]>([])
  const [scale, setScale] = useState<MasteryScaleEntry[]>([])
  const [cells, setCells] = useState<HeatmapCell[]>([])
  const [roster, setRoster] = useState<RosterEntry[]>([])
  const [orgId, setOrgId] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState<string | null>(null)
  const announceRef = useRef<HTMLDivElement>(null)

  function announce(msg: string) {
    if (announceRef.current) announceRef.current.textContent = msg
  }

  // Load org and roster
  useEffect(() => {
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/me')
        if (!res.ok) return
        const me = (await res.json()) as { orgId?: string }
        if (me.orgId) setOrgId(me.orgId)
      } catch { /* ignore */ }
    })()
  }, [])

  useEffect(() => {
    if (!courseCode) return
    void (async () => {
      try {
        const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/enrollments`)
        if (!res.ok) return
        const body = (await res.json()) as { enrollments?: Array<{ userId: string; email: string; displayName?: string; role: string }> }
        setRoster((body.enrollments ?? []).filter((e) => e.role === 'student'))
      } catch { /* ignore */ }
    })()
  }, [courseCode])

  const loadData = useCallback(async () => {
    if (!courseCode || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const [stds, sc, hm] = await Promise.all([
        listCourseStandards(courseCode),
        getMasteryScale(orgId),
        getSBGHeatmap(courseCode, period),
      ])
      setStandards(stds)
      setScale(sc)
      setCells(hm)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load heatmap.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, orgId, period])

  useEffect(() => {
    void loadData()
  }, [loadData])

  function getCell(studentId: string, standardId: string): HeatmapCell | undefined {
    return cells.find((c) => c.studentId === studentId && c.standardId === standardId)
  }

  function getScaleEntry(value: number): MasteryScaleEntry | undefined {
    return scale.find((s) => s.value === value)
  }

  function cellLabel(cell: HeatmapCell | undefined): string {
    if (!cell) return '—'
    const entry = getScaleEntry(cell.scoreValue)
    return entry ? `${cell.scoreValue} – ${entry.label}` : String(cell.scoreValue)
  }

  function cellColor(cell: HeatmapCell | undefined): string {
    if (!cell) return '#f3f4f6'
    const entry = getScaleEntry(cell.scoreValue)
    return entry?.color ?? '#d1d5db'
  }

  async function handleScoreChange(studentId: string, standardId: string, value: number) {
    if (!courseCode) return
    const key = `${studentId}:${standardId}`
    setSaving(key)
    try {
      await recordMasteryScore({
        studentId,
        standardId,
        courseCode,
        gradingPeriod: period,
        scoreValue: value,
        source: 'observation',
      })
      announce(`Score updated to ${value}`)
      await loadData()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save score.')
    } finally {
      setSaving(null)
    }
  }

  const meetsThreshold = scale.length > 0 ? Math.max(...scale.map((s) => s.value)) - 1 : 2

  if (!courseCode) return null

  return (
    <div style={{ padding: '24px', maxWidth: '1200px', margin: '0 auto' }}>
      <div ref={announceRef} role="status" aria-live="polite" style={{ position: 'absolute', left: '-9999px' }} />

      <h1 style={{ fontSize: '1.5rem', fontWeight: 700, marginBottom: '8px' }}>
        Standards-Based Grading — Mastery Heatmap
      </h1>
      <p style={{ color: '#6b7280', marginBottom: '20px', fontSize: '0.875rem' }}>
        Course: <strong>{courseCode}</strong>
      </p>

      <div style={{ display: 'flex', gap: '12px', alignItems: 'center', marginBottom: '20px', flexWrap: 'wrap' }}>
        <label htmlFor="sbg-period" style={{ fontWeight: 600 }}>
          Grading Period:
        </label>
        <input
          id="sbg-period"
          type="text"
          value={period}
          onChange={(e) => setPeriod(e.target.value)}
          placeholder="e.g. Q1-2026"
          style={{ border: '1px solid #d1d5db', borderRadius: '6px', padding: '6px 12px', fontSize: '0.9rem' }}
        />
        <button
          onClick={() => { void loadData() }}
          disabled={loading}
          style={{ padding: '6px 16px', background: '#2563eb', color: '#fff', border: 'none', borderRadius: '6px', cursor: 'pointer' }}
        >
          {loading ? 'Loading…' : 'Load'}
        </button>
      </div>

      {error && (
        <p role="alert" style={{ color: '#dc2626', marginBottom: '12px' }}>
          {error}
        </p>
      )}

      {/* Mastery scale legend */}
      {scale.length > 0 && (
        <div style={{ display: 'flex', gap: '12px', marginBottom: '16px', flexWrap: 'wrap' }}>
          {scale.map((s) => (
            <span
              key={s.value}
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: '6px',
                padding: '4px 10px',
                borderRadius: '9999px',
                background: s.color ?? '#d1d5db',
                color: '#fff',
                fontSize: '0.8rem',
                fontWeight: 600,
              }}
            >
              <span aria-hidden="true">■</span>
              {s.value} – {s.label}
            </span>
          ))}
        </div>
      )}

      {standards.length === 0 && !loading && (
        <p style={{ color: '#6b7280' }}>
          No standards found for this org. Ask your admin to import standards first.
        </p>
      )}

      {roster.length === 0 && !loading && standards.length > 0 && (
        <p style={{ color: '#6b7280' }}>No students enrolled in this course.</p>
      )}

      {standards.length > 0 && roster.length > 0 && (
        <div style={{ overflowX: 'auto' }}>
          <table
            role="grid"
            aria-label={`Mastery heatmap for ${courseCode}, period ${period}`}
            style={{ borderCollapse: 'collapse', width: '100%', fontSize: '0.8rem' }}
          >
            <thead>
              <tr>
                <th scope="col" style={{ padding: '8px 12px', textAlign: 'left', background: '#f9fafb', border: '1px solid #e5e7eb', minWidth: '140px' }}>
                  Student
                </th>
                {standards.map((std) => (
                  <th
                    key={std.id}
                    scope="col"
                    title={std.description}
                    style={{
                      padding: '8px',
                      textAlign: 'center',
                      background: '#f9fafb',
                      border: '1px solid #e5e7eb',
                      minWidth: '80px',
                      maxWidth: '120px',
                      wordBreak: 'break-word',
                    }}
                  >
                    {std.code}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {roster.map((student) => (
                <tr key={student.userId}>
                  <th scope="row" style={{ padding: '8px 12px', textAlign: 'left', border: '1px solid #e5e7eb', fontWeight: 500, whiteSpace: 'nowrap' }}>
                    {studentLabel(student)}
                  </th>
                  {standards.map((std) => {
                    const cell = getCell(student.userId, std.id)
                    const key = `${student.userId}:${std.id}`
                    const isSaving = saving === key
                    const isBelowThreshold = cell && cell.scoreValue <= meetsThreshold - 1
                    return (
                      <td
                        key={std.id}
                        role="gridcell"
                        tabIndex={0}
                        aria-label={`${studentLabel(student)}: ${std.code}: ${cellLabel(cell)}`}
                        style={{
                          padding: '4px',
                          border: '1px solid #e5e7eb',
                          textAlign: 'center',
                          background: isBelowThreshold ? '#fef2f2' : '#fff',
                          outline: 'none',
                        }}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.currentTarget.querySelector('select')?.focus()
                          }
                        }}
                      >
                        <select
                          aria-label={`Set mastery for ${studentLabel(student)} on ${std.code}`}
                          disabled={isSaving}
                          value={cell?.scoreValue ?? ''}
                          onChange={(e) => {
                            const v = parseInt(e.target.value, 10)
                            if (!isNaN(v) && v >= 1) {
                              void handleScoreChange(student.userId, std.id, v)
                            }
                          }}
                          style={{
                            border: 'none',
                            background: cellColor(cell),
                            color: cell ? '#fff' : '#374151',
                            borderRadius: '4px',
                            padding: '2px 4px',
                            width: '100%',
                            cursor: 'pointer',
                            fontWeight: 600,
                          }}
                        >
                          <option value="">—</option>
                          {scale.map((s) => (
                            <option key={s.value} value={s.value}>
                              {s.value} – {s.label}
                            </option>
                          ))}
                        </select>
                        {isSaving && <span aria-hidden="true" style={{ fontSize: '0.7rem', color: '#6b7280' }}>…</span>}
                      </td>
                    )
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
