import { useCallback, useEffect, useState } from 'react'
import {
  putModuleRequirements,
  type ModuleCompletionMode,
} from '../../lib/conditional-release-api'

const MODES: { value: ModuleCompletionMode; label: string }[] = [
  { value: 'all_items', label: 'All items required' },
  { value: 'one_item', label: 'Complete any one item' },
  { value: 'sequential_order', label: 'Sequential order' },
]

type Props = {
  courseCode: string
  moduleId: string
  allModules: { id: string; title: string }[]
}

export function ModuleRequirementsPanel({ courseCode, moduleId, allModules }: Props) {
  const [open, setOpen] = useState(false)
  const [mode, setMode] = useState<ModuleCompletionMode>('all_items')
  const [prereqs, setPrereqs] = useState<string[]>([])
  const [unlockAt, setUnlockAt] = useState('')
  const [saving, setSaving] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const save = useCallback(async () => {
    setSaving(true)
    setErr(null)
    setSaved(false)
    try {
      await putModuleRequirements(courseCode, moduleId, {
        completionMode: mode,
        prerequisiteModuleIds: prereqs,
        unlockAt: unlockAt.trim() ? new Date(unlockAt).toISOString() : null,
      })
      setSaved(true)
    } catch (e) {
      setErr(e instanceof Error ? e.message : 'Could not save requirements.')
    } finally {
      setSaving(false)
    }
  }, [courseCode, mode, moduleId, prereqs, unlockAt])

  useEffect(() => {
    if (!open) return
    setSaved(false)
  }, [open, mode, prereqs, unlockAt])

  const otherModules = allModules.filter((m) => m.id !== moduleId)

  return (
    <div className="mt-3 rounded-xl border border-sky-100 bg-sky-50/40 px-3 py-2 text-start dark:border-sky-900/40 dark:bg-sky-950/30">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="text-xs font-semibold text-sky-800 hover:text-sky-600 dark:text-sky-200 dark:hover:text-sky-100"
      >
        {open ? '▼' : '▶'} Module requirements
      </button>
      {open ? (
        <div className="mt-2 space-y-2 text-xs text-slate-700 dark:text-neutral-200">
          {err ? <p className="text-rose-700 dark:text-rose-300">{err}</p> : null}
          {saved ? (
            <p className="text-emerald-700 dark:text-emerald-300" role="status">
              Requirements saved.
            </p>
          ) : null}
          <label className="block">
            <span className="font-medium text-slate-600 dark:text-neutral-300">Completion mode</span>
            <select
              value={mode}
              onChange={(e) => setMode(e.target.value as ModuleCompletionMode)}
              className="mt-1 w-full rounded border border-slate-200 bg-white px-2 py-1 text-xs dark:border-neutral-600 dark:bg-neutral-900"
            >
              {MODES.map((m) => (
                <option key={m.value} value={m.value}>
                  {m.label}
                </option>
              ))}
            </select>
          </label>
          {otherModules.length > 0 ? (
            <fieldset>
              <legend className="font-medium text-slate-600 dark:text-neutral-300">Prerequisites</legend>
              <ul className="mt-1 space-y-1">
                {otherModules.map((m) => (
                  <li key={m.id}>
                    <label className="inline-flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={prereqs.includes(m.id)}
                        onChange={(e) => {
                          setPrereqs((prev) =>
                            e.target.checked ? [...prev, m.id] : prev.filter((id) => id !== m.id),
                          )
                        }}
                      />
                      <span>{m.title}</span>
                    </label>
                  </li>
                ))}
              </ul>
            </fieldset>
          ) : null}
          <label className="block">
            <span className="font-medium text-slate-600 dark:text-neutral-300">Lock until (optional)</span>
            <input
              type="datetime-local"
              value={unlockAt}
              onChange={(e) => setUnlockAt(e.target.value)}
              className="mt-1 w-full rounded border border-slate-200 bg-white px-2 py-1 text-xs dark:border-neutral-600 dark:bg-neutral-900"
            />
          </label>
          <button
            type="button"
            disabled={saving}
            onClick={() => void save()}
            className="rounded bg-sky-700 px-3 py-1.5 text-xs font-semibold text-white hover:bg-sky-600 disabled:opacity-50 dark:bg-sky-600"
          >
            {saving ? 'Saving…' : 'Save requirements'}
          </button>
        </div>
      ) : null}
    </div>
  )
}
