import { useEffect, useState } from 'react'
import {
  fetchCourseSections,
  fetchEnrollmentGroupsTree,
  fetchCourseEnrollmentsList,
  fetchItemAssignToOverrides,
  putItemAssignToOverrides,
  type AssignToTarget,
  type AssignToTargetType,
  type CourseSection,
} from '../../lib/courses-api'
import { assignToOverridesFeatureEnabled } from '../../lib/platform-features'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

type Props = {
  courseCode: string
  itemId?: string
  disabled?: boolean
}

type GroupOption = { id: string; label: string }
type StudentOption = { id: string; label: string }

const inputClass =
  'w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:focus:border-indigo-500 dark:focus:ring-indigo-500'

function isoToLocal(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function localToIso(local: string): string | null {
  if (!local.trim()) return null
  const d = new Date(local)
  if (Number.isNaN(d.getTime())) return null
  return d.toISOString()
}

const targetTypeLabels: Record<AssignToTargetType, string> = {
  everyone: 'Everyone',
  section: 'Section',
  group: 'Group',
  student: 'Student',
}

/** Plan 2.15 — per-item "Assign To" editor: target everyone, sections, groups, or individual students,
 * each with optional due/availability overrides. */
export function AssignToEditor({ courseCode, itemId, disabled }: Props) {
  const [enabled] = useState(() => assignToOverridesFeatureEnabled())
  const [targets, setTargets] = useState<AssignToTarget[] | null>(null)
  const [orphaned, setOrphaned] = useState(false)
  const [sections, setSections] = useState<CourseSection[]>([])
  const [groups, setGroups] = useState<GroupOption[]>([])
  const [students, setStudents] = useState<StudentOption[]>([])
  const [busy, setBusy] = useState(false)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [newTargetType, setNewTargetType] = useState<AssignToTargetType>('section')
  const [newTargetId, setNewTargetId] = useState('')

  useEffect(() => {
    if (!enabled || !itemId) return
    let cancelled = false
    void (async () => {
      try {
        const [ov, secs] = await Promise.all([
          fetchItemAssignToOverrides(courseCode, itemId),
          fetchCourseSections(courseCode).catch(() => []),
        ])
        if (cancelled) return
        setTargets(ov.targets)
        setOrphaned(ov.orphaned)
        setSections(secs)
      } catch (e) {
        if (!cancelled) setLoadError(e instanceof Error ? e.message : 'Could not load assign-to targets.')
      }
      try {
        const tree = await fetchEnrollmentGroupsTree(courseCode)
        if (cancelled) return
        const opts: GroupOption[] = []
        for (const set of tree.groupSets) {
          for (const g of set.groups) {
            opts.push({ id: g.id, label: `${set.name} — ${g.name}` })
          }
        }
        setGroups(opts)
      } catch {
        if (!cancelled) setGroups([])
      }
      try {
        const roster = await fetchCourseEnrollmentsList(courseCode)
        if (cancelled) return
        setStudents(
          roster
            .filter((r) => r.role === 'student')
            .map((r) => ({ id: r.id, label: r.displayName ?? r.id })),
        )
      } catch {
        if (!cancelled) setStudents([])
      }
    })()
    return () => {
      cancelled = true
    }
  }, [enabled, courseCode, itemId])

  if (!enabled || !itemId) return null
  if (loadError) {
    return <p className="text-sm text-rose-600 dark:text-rose-400">{loadError}</p>
  }
  if (targets === null) {
    return <p className="text-sm text-slate-500 dark:text-neutral-400">Loading…</p>
  }

  function optionsForType(t: AssignToTargetType): { id: string; label: string }[] {
    if (t === 'section') return sections.map((s) => ({ id: s.id, label: s.name ?? s.sectionCode }))
    if (t === 'group') return groups
    if (t === 'student') return students
    return []
  }

  function addTarget() {
    if (newTargetType !== 'everyone' && !newTargetId) return
    const next: AssignToTarget = {
      targetType: newTargetType,
      targetId: newTargetType === 'everyone' ? null : newTargetId,
    }
    setTargets((prev) => [...(prev ?? []), next])
    setNewTargetId('')
  }

  function removeTarget(index: number) {
    setTargets((prev) => (prev ?? []).filter((_, i) => i !== index))
  }

  function updateTarget(index: number, patch: Partial<AssignToTarget>) {
    setTargets((prev) => (prev ?? []).map((t, i) => (i === index ? { ...t, ...patch } : t)))
  }

  function labelFor(t: AssignToTarget): string {
    if (t.targetType === 'everyone') return 'Everyone'
    const opts = optionsForType(t.targetType)
    const found = opts.find((o) => o.id === t.targetId)
    return found ? found.label : (t.targetId ?? '')
  }

  async function onSave() {
    if (!itemId) return
    setBusy(true)
    try {
      const result = await putItemAssignToOverrides(courseCode, itemId, targets ?? [])
      setTargets(result.targets)
      setOrphaned(result.orphaned)
      toastSaveOk('Assign-to targets saved')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not save assign-to targets.')
    } finally {
      setBusy(false)
    }
  }

  const isDisabled = Boolean(disabled) || busy

  return (
    <div className="space-y-3 pt-1">
      <p className="text-[11px] leading-relaxed text-slate-500 dark:text-neutral-400">
        Target this item to everyone, specific sections, groups, or individual students. Each audience can
        have its own due date and availability window. Save the page from the toolbar to apply changes; an
        item with no targets is visible to everyone (default).
      </p>
      {orphaned ? (
        <p
          role="alert"
          className="rounded-md border border-amber-200 bg-amber-50 px-2.5 py-2 text-[11px] text-amber-950 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100"
        >
          This item is targeted only at audiences with no students currently enrolled — no one will see it.
        </p>
      ) : null}
      <ul className="space-y-2">
        {(targets ?? []).map((t, i) => (
          <li
            key={`${t.targetType}-${t.targetId ?? 'everyone'}-${i}`}
            className="space-y-2 rounded-lg border border-slate-200 p-2.5 dark:border-neutral-700"
          >
            <div className="flex items-center justify-between gap-2">
              <span className="text-[13px] font-medium text-slate-800 dark:text-neutral-200">
                {targetTypeLabels[t.targetType]}
                {t.targetType !== 'everyone' ? <span className="text-slate-500"> — {labelFor(t)}</span> : null}
              </span>
              <button
                type="button"
                disabled={isDisabled}
                onClick={() => removeTarget(i)}
                className="text-xs font-medium text-rose-600 hover:text-rose-500 disabled:opacity-50"
              >
                Remove
              </button>
            </div>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
              <label className="block text-xs">
                <span className="text-slate-500 dark:text-neutral-400">Due</span>
                <input
                  type="datetime-local"
                  value={isoToLocal(t.dueAt)}
                  disabled={isDisabled}
                  onChange={(e) => updateTarget(i, { dueAt: localToIso(e.target.value) })}
                  className={`mt-0.5 ${inputClass}`}
                />
              </label>
              <label className="block text-xs">
                <span className="text-slate-500 dark:text-neutral-400">Available from</span>
                <input
                  type="datetime-local"
                  value={isoToLocal(t.availableFrom)}
                  disabled={isDisabled}
                  onChange={(e) => updateTarget(i, { availableFrom: localToIso(e.target.value) })}
                  className={`mt-0.5 ${inputClass}`}
                />
              </label>
              <label className="block text-xs">
                <span className="text-slate-500 dark:text-neutral-400">Available until</span>
                <input
                  type="datetime-local"
                  value={isoToLocal(t.availableUntil)}
                  disabled={isDisabled}
                  onChange={(e) => updateTarget(i, { availableUntil: localToIso(e.target.value) })}
                  className={`mt-0.5 ${inputClass}`}
                />
              </label>
            </div>
          </li>
        ))}
        {(targets ?? []).length === 0 ? (
          <li className="rounded-lg border border-dashed border-slate-200 p-3 text-center text-xs text-slate-400 dark:border-neutral-700 dark:text-neutral-500">
            No targets set — visible to everyone with the item&apos;s default dates.
          </li>
        ) : null}
      </ul>
      <div className="flex flex-wrap items-end gap-2 border-t border-slate-100 pt-3 dark:border-neutral-800">
        <label className="block text-xs">
          <span className="text-slate-500 dark:text-neutral-400">Add audience</span>
          <select
            value={newTargetType}
            disabled={isDisabled}
            onChange={(e) => {
              setNewTargetType(e.target.value as AssignToTargetType)
              setNewTargetId('')
            }}
            className={`mt-0.5 ${inputClass}`}
          >
            <option value="everyone">Everyone</option>
            <option value="section">Section</option>
            <option value="group">Group</option>
            <option value="student">Student</option>
          </select>
        </label>
        {newTargetType !== 'everyone' ? (
          <label className="block min-w-[12rem] flex-1 text-xs">
            <span className="text-slate-500 dark:text-neutral-400">{targetTypeLabels[newTargetType]}</span>
            <select
              value={newTargetId}
              disabled={isDisabled}
              onChange={(e) => setNewTargetId(e.target.value)}
              className={`mt-0.5 ${inputClass}`}
            >
              <option value="">Select…</option>
              {optionsForType(newTargetType).map((o) => (
                <option key={o.id} value={o.id}>
                  {o.label}
                </option>
              ))}
            </select>
          </label>
        ) : null}
        <button
          type="button"
          disabled={isDisabled || (newTargetType !== 'everyone' && !newTargetId)}
          onClick={addTarget}
          className="rounded-lg border border-slate-200 bg-white px-3 py-2 text-xs font-medium text-slate-800 shadow-sm hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
        >
          + Add
        </button>
        <button
          type="button"
          disabled={isDisabled}
          onClick={() => void onSave()}
          className="ms-auto rounded-lg bg-indigo-600 px-4 py-2 text-xs font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:opacity-50 dark:bg-indigo-500 dark:hover:bg-indigo-400"
        >
          Save assign-to targets
        </button>
      </div>
    </div>
  )
}
