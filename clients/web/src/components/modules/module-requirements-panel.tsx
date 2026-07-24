import { useCallback, useEffect, useState } from 'react'
import {
  putModuleRequirements,
  type ModuleCompletionMode,
} from '../../lib/conditional-release-api'

export const MODULE_REQUIREMENTS_FORM_ID = 'module-requirements-form'

const MODES: { value: ModuleCompletionMode; label: string }[] = [
  { value: 'all_items', label: 'All items required' },
  { value: 'one_item', label: 'Complete any one item' },
  { value: 'sequential_order', label: 'Sequential order' },
]

type Props = {
  courseCode: string
  moduleId: string
  allModules: { id: string; title: string }[]
  onSavingChange?: (saving: boolean) => void
  onErrorChange?: (error: string | null) => void
  onSaved?: () => void
}

export function ModuleRequirementsPanel({
  courseCode,
  moduleId,
  allModules,
  onSavingChange,
  onErrorChange,
  onSaved,
}: Props) {
  const [mode, setMode] = useState<ModuleCompletionMode>('all_items')
  const [prereqs, setPrereqs] = useState<string[]>([])
  const [unlockAt, setUnlockAt] = useState('')
  const [err, setErr] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const save = useCallback(async () => {
    onSavingChange?.(true)
    setErr(null)
    onErrorChange?.(null)
    setSaved(false)
    try {
      await putModuleRequirements(courseCode, moduleId, {
        completionMode: mode,
        prerequisiteModuleIds: prereqs,
        unlockAt: unlockAt.trim() ? new Date(unlockAt).toISOString() : null,
      })
      setSaved(true)
      onSaved?.()
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Could not save requirements.'
      setErr(message)
      onErrorChange?.(message)
    } finally {
      onSavingChange?.(false)
    }
  }, [courseCode, mode, moduleId, onErrorChange, onSaved, onSavingChange, prereqs, unlockAt])

  useEffect(() => {
    setSaved(false)
  }, [mode, prereqs, unlockAt])

  const otherModules = allModules.filter((m) => m.id !== moduleId)

  return (
    <form
      id={MODULE_REQUIREMENTS_FORM_ID}
      className="space-y-3 text-start"
      onSubmit={(e) => {
        e.preventDefault()
        void save()
      }}
    >
      <p className="text-xs text-slate-500">
        Choose how students complete this module and any modules they must finish first.
      </p>
      {err ? <p className="text-sm text-rose-700">{err}</p> : null}
      {saved ? (
        <p className="text-sm text-emerald-700" role="status">
          Requirements saved.
        </p>
      ) : null}
      <label className="block">
        <span className="text-xs font-medium text-slate-600">Completion mode</span>
        <select
          value={mode}
          onChange={(e) => setMode(e.target.value as ModuleCompletionMode)}
          className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2"
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
          <legend className="text-xs font-medium text-slate-600">Prerequisites</legend>
          <ul className="mt-1.5 max-h-36 space-y-1.5 overflow-y-auto rounded-xl border border-slate-200 bg-slate-50/70 px-3 py-2">
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
                    className="h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                  />
                  <span className="text-sm text-slate-700">{m.title}</span>
                </label>
              </li>
            ))}
          </ul>
        </fieldset>
      ) : (
        <p className="text-xs text-slate-500">No other modules available as prerequisites.</p>
      )}
      <label className="block">
        <span className="text-xs font-medium text-slate-600">Lock until (optional)</span>
        <input
          type="datetime-local"
          value={unlockAt}
          onChange={(e) => setUnlockAt(e.target.value)}
          className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2"
        />
      </label>
    </form>
  )
}
