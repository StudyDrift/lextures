import { useEffect, useState } from 'react'
import { Loader2, Plus, Trash2 } from 'lucide-react'
import {
  bulkExtendAssignToDueDate,
  fetchAssignToTargets,
  fetchCourseEnrollmentsList,
  fetchCourseSections,
  fetchEnrollmentGroupsTree,
  putAssignToTargets,
  type AssignToTarget,
  type AssignToTargetType,
  type AssignToTargetWrite,
  type CourseEnrollmentRosterRow,
  type CourseSection,
} from '../../lib/courses-api'

export type AssignToEditorProps = {
  courseCode: string
  itemId: string
  disabled?: boolean
}

type DraftTarget = {
  key: string
  targetType: AssignToTargetType
  targetId: string | null
  dueLocal: string
  availableFromLocal: string
  availableUntilLocal: string
}

function isoToLocal(iso: string | null): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function localToIso(value: string): string | null {
  const t = value.trim()
  if (!t) return null
  const d = new Date(t)
  if (Number.isNaN(d.getTime())) return null
  return d.toISOString()
}

function newKey(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return `t-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

function targetFromApi(t: AssignToTarget): DraftTarget {
  return {
    key: t.id,
    targetType: t.targetType,
    targetId: t.targetId,
    dueLocal: isoToLocal(t.dueAt),
    availableFromLocal: isoToLocal(t.availableFrom),
    availableUntilLocal: isoToLocal(t.availableUntil),
  }
}

const selectClass =
  'rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100'

const inputClass = selectClass

const targetTypeLabels: Record<AssignToTargetType, string> = {
  everyone: 'Everyone',
  section: 'Section',
  group: 'Group',
  student: 'Student',
}

/** Plan 2.15 — "assign to" editor: targets an assignment/quiz at everyone, sections, groups, or
 * individual students, each with its own optional due/availability override. */
export function AssignToEditor({ courseCode, itemId, disabled }: AssignToEditorProps) {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [targets, setTargets] = useState<DraftTarget[]>([])
  const [orphaned, setOrphaned] = useState(false)
  const [sections, setSections] = useState<CourseSection[]>([])
  const [groups, setGroups] = useState<{ id: string; name: string }[]>([])
  const [enrollments, setEnrollments] = useState<CourseEnrollmentRosterRow[]>([])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    Promise.all([
      fetchAssignToTargets(courseCode, itemId),
      fetchCourseSections(courseCode).catch(() => []),
      fetchEnrollmentGroupsTree(courseCode).catch(() => ({ groupSets: [] })),
      fetchCourseEnrollmentsList(courseCode).catch(() => []),
    ])
      .then(([targetsRes, sectionsRes, groupsRes, enrollmentsRes]) => {
        if (cancelled) return
        setTargets(targetsRes.targets.map(targetFromApi))
        setOrphaned(targetsRes.orphaned)
        setSections(sectionsRes)
        setGroups(groupsRes.groupSets.flatMap((s) => s.groups.map((g) => ({ id: g.id, name: g.name }))))
        setEnrollments(enrollmentsRes.filter((e) => e.role === 'student'))
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load assign-to targets.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [courseCode, itemId])

  function addTarget() {
    const hasEveryone = targets.some((t) => t.targetType === 'everyone')
    setTargets((prev) => [
      ...prev,
      {
        key: newKey(),
        targetType: hasEveryone ? 'section' : 'everyone',
        targetId: null,
        dueLocal: '',
        availableFromLocal: '',
        availableUntilLocal: '',
      },
    ])
  }

  function updateTarget(key: string, patch: Partial<DraftTarget>) {
    setTargets((prev) => prev.map((t) => (t.key === key ? { ...t, ...patch } : t)))
  }

  function removeTarget(key: string) {
    setTargets((prev) => prev.filter((t) => t.key !== key))
  }

  async function save() {
    setSaving(true)
    setError(null)
    try {
      const writes: AssignToTargetWrite[] = targets.map((t) => ({
        targetType: t.targetType,
        targetId: t.targetType === 'everyone' ? null : t.targetId,
        dueAt: localToIso(t.dueLocal),
        availableFrom: localToIso(t.availableFromLocal),
        availableUntil: localToIso(t.availableUntilLocal),
      }))
      const res = await putAssignToTargets(courseCode, itemId, writes)
      setTargets(res.targets.map(targetFromApi))
      setOrphaned(res.orphaned)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not save assign-to targets.')
    } finally {
      setSaving(false)
    }
  }

  async function extendForStudent(enrollmentId: string, dueLocal: string) {
    const iso = localToIso(dueLocal)
    if (!iso) return
    setSaving(true)
    setError(null)
    try {
      await bulkExtendAssignToDueDate(courseCode, itemId, [enrollmentId], iso)
      const res = await fetchAssignToTargets(courseCode, itemId)
      setTargets(res.targets.map(targetFromApi))
      setOrphaned(res.orphaned)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not extend due date.')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-2 text-sm text-slate-500 dark:text-neutral-400">
        <Loader2 className="size-4 shrink-0 animate-spin" aria-hidden />
        Loading assign-to targets…
      </div>
    )
  }

  return (
    <div className="space-y-3 pt-1">
      <p className="text-[11px] leading-relaxed text-slate-500 dark:text-neutral-400">
        By default this item is assigned to everyone. Add a section, group, or student target to give it a
        different due date or availability window, or to hide it from everyone else.
      </p>
      {orphaned ? (
        <p className="rounded-md border border-amber-200 bg-amber-50 px-2.5 py-2 text-[11px] text-amber-950 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100">
          This item isn&apos;t assigned to anyone — no current student matches any target below.
        </p>
      ) : null}
      {error ? <p className="text-xs text-rose-600 dark:text-rose-400">{error}</p> : null}

      <div className="space-y-2">
        {targets.map((t) => (
          <div
            key={t.key}
            className="space-y-2 rounded-lg border border-slate-200 bg-slate-50/60 p-2.5 dark:border-neutral-700 dark:bg-neutral-900/40"
          >
            <div className="flex flex-wrap items-center gap-2">
              <select
                aria-label="Audience type"
                value={t.targetType}
                disabled={disabled || saving}
                onChange={(e) => updateTarget(t.key, { targetType: e.target.value as AssignToTargetType, targetId: null })}
                className={selectClass}
              >
                {(Object.keys(targetTypeLabels) as AssignToTargetType[]).map((tt) => (
                  <option key={tt} value={tt}>
                    {targetTypeLabels[tt]}
                  </option>
                ))}
              </select>

              {t.targetType === 'section' ? (
                <select
                  aria-label="Section"
                  value={t.targetId ?? ''}
                  disabled={disabled || saving}
                  onChange={(e) => updateTarget(t.key, { targetId: e.target.value || null })}
                  className={selectClass}
                >
                  <option value="">— Select section —</option>
                  {sections.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name ?? s.sectionCode}
                    </option>
                  ))}
                </select>
              ) : null}

              {t.targetType === 'group' ? (
                <select
                  aria-label="Group"
                  value={t.targetId ?? ''}
                  disabled={disabled || saving}
                  onChange={(e) => updateTarget(t.key, { targetId: e.target.value || null })}
                  className={selectClass}
                >
                  <option value="">— Select group —</option>
                  {groups.map((g) => (
                    <option key={g.id} value={g.id}>
                      {g.name}
                    </option>
                  ))}
                </select>
              ) : null}

              {t.targetType === 'student' ? (
                <select
                  aria-label="Student"
                  value={t.targetId ?? ''}
                  disabled={disabled || saving}
                  onChange={(e) => updateTarget(t.key, { targetId: e.target.value || null })}
                  className={selectClass}
                >
                  <option value="">— Select student —</option>
                  {enrollments.map((en) => (
                    <option key={en.id} value={en.id}>
                      {en.displayName ?? en.userId}
                    </option>
                  ))}
                </select>
              ) : null}

              <button
                type="button"
                disabled={disabled || saving}
                onClick={() => removeTarget(t.key)}
                aria-label="Remove target"
                className="ms-auto inline-flex items-center justify-center rounded-md p-1.5 text-rose-600 hover:bg-rose-50 disabled:opacity-40 dark:text-rose-400 dark:hover:bg-rose-950/40"
              >
                <Trash2 className="size-4" aria-hidden />
              </button>
            </div>

            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <label className="space-y-1 text-[11px] text-slate-500 dark:text-neutral-400">
                Due date
                <input
                  type="datetime-local"
                  value={t.dueLocal}
                  disabled={disabled || saving}
                  onChange={(e) => updateTarget(t.key, { dueLocal: e.target.value })}
                  className={`mt-1 block w-full ${inputClass}`}
                />
              </label>
              <label className="space-y-1 text-[11px] text-slate-500 dark:text-neutral-400">
                Available from
                <input
                  type="datetime-local"
                  value={t.availableFromLocal}
                  disabled={disabled || saving}
                  onChange={(e) => updateTarget(t.key, { availableFromLocal: e.target.value })}
                  className={`mt-1 block w-full ${inputClass}`}
                />
              </label>
              <label className="space-y-1 text-[11px] text-slate-500 dark:text-neutral-400">
                Available until
                <input
                  type="datetime-local"
                  value={t.availableUntilLocal}
                  disabled={disabled || saving}
                  onChange={(e) => updateTarget(t.key, { availableUntilLocal: e.target.value })}
                  className={`mt-1 block w-full ${inputClass}`}
                />
              </label>
            </div>

            {t.targetType === 'student' && t.targetId ? (
              <BulkExtendRow
                disabled={disabled || saving}
                onExtend={(local) => void extendForStudent(t.targetId as string, local)}
              />
            ) : null}
          </div>
        ))}
      </div>

      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          disabled={disabled || saving}
          onClick={addTarget}
          className="inline-flex items-center gap-1.5 rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-800 shadow-sm hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-40 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
        >
          <Plus className="size-3.5 shrink-0" aria-hidden />
          Add audience
        </button>
        <button
          type="button"
          disabled={disabled || saving}
          onClick={() => void save()}
          className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-3 py-2 text-xs font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-50 dark:bg-indigo-500 dark:hover:bg-indigo-400"
        >
          {saving ? <Loader2 className="size-3.5 shrink-0 animate-spin" aria-hidden /> : null}
          Save assign-to targets
        </button>
      </div>
    </div>
  )
}

function BulkExtendRow({ disabled, onExtend }: { disabled?: boolean; onExtend: (local: string) => void }) {
  const [local, setLocal] = useState('')
  return (
    <div className="flex flex-wrap items-end gap-2 border-t border-slate-200/70 pt-2 dark:border-neutral-700/60">
      <label className="space-y-1 text-[11px] text-slate-500 dark:text-neutral-400">
        Quick-extend due date for this student
        <input
          type="datetime-local"
          value={local}
          disabled={disabled}
          onChange={(e) => setLocal(e.target.value)}
          className={`mt-1 block ${inputClass}`}
        />
      </label>
      <button
        type="button"
        disabled={disabled || !local}
        onClick={() => onExtend(local)}
        className="rounded-lg border border-slate-200 bg-white px-2.5 py-1.5 text-xs font-medium text-slate-800 shadow-sm hover:bg-slate-50 disabled:opacity-40 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
      >
        Extend
      </button>
    </div>
  )
}
