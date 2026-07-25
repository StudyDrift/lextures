import { useCallback, useEffect, useState } from 'react'
import {
  fetchModuleRequirements,
  putModuleRequirements,
  type ModuleCompletionMode,
} from '../../lib/conditional-release-api'

export const MODULE_REQUIREMENTS_FORM_ID = 'module-requirements-form'

const MODES: { value: ModuleCompletionMode; label: string }[] = [
  { value: 'all_items', label: 'All items required' },
  { value: 'one_item', label: 'Complete any one item' },
  { value: 'sequential_order', label: 'Sequential order' },
]

function isoToDatetimeLocalValue(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
}

type Props = {
  courseCode: string
  moduleId: string
  allModules: { id: string; title: string }[]
  onSavingChange?: (saving: boolean) => void
  onLoadingChange?: (loading: boolean) => void
  onErrorChange?: (error: string | null) => void
  onSaved?: () => void
  /** Parent registers this to submit from a footer button outside the form. */
  registerSubmit?: (submit: (() => void) | null) => void
}

export function ModuleRequirementsPanel({
  courseCode,
  moduleId,
  allModules,
  onSavingChange,
  onLoadingChange,
  onErrorChange,
  onSaved,
  registerSubmit,
}: Props) {
  const [mode, setMode] = useState<ModuleCompletionMode>('all_items')
  const [prereqs, setPrereqs] = useState<string[]>([])
  const [unlockAt, setUnlockAt] = useState('')
  const [loading, setLoading] = useState(true)
  const [err, setErr] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    onLoadingChange?.(true)
    setErr(null)
    onErrorChange?.(null)
    setSaved(false)
    void (async () => {
      try {
        const req = await fetchModuleRequirements(courseCode, moduleId)
        if (cancelled) return
        if (req) {
          setMode(req.completionMode ?? 'all_items')
          setPrereqs(req.prerequisiteModuleIds ?? req.prerequisiteIds ?? [])
          setUnlockAt(isoToDatetimeLocalValue(req.unlockAt))
        } else {
          setMode('all_items')
          setPrereqs([])
          setUnlockAt('')
        }
      } catch (e) {
        if (cancelled) return
        // Missing config is fine (404 → null from fetch); only surface unexpected failures.
        const message = e instanceof Error ? e.message : 'Could not load requirements.'
        if (!message.toLowerCase().includes('not found')) {
          setErr(message)
          onErrorChange?.(message)
        }
      } finally {
        if (!cancelled) {
          setLoading(false)
          onLoadingChange?.(false)
        }
      }
    })()
    return () => {
      cancelled = true
      onLoadingChange?.(false)
    }
  }, [courseCode, moduleId, onErrorChange, onLoadingChange])

  const save = useCallback(async () => {
    if (loading) return
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
  }, [
    courseCode,
    loading,
    mode,
    moduleId,
    onErrorChange,
    onSaved,
    onSavingChange,
    prereqs,
    unlockAt,
  ])

  useEffect(() => {
    setSaved(false)
  }, [mode, prereqs, unlockAt])

  useEffect(() => {
    if (!registerSubmit) return
    registerSubmit(() => {
      void save()
    })
    return () => registerSubmit(null)
  }, [registerSubmit, save])

  const otherModules = allModules.filter((m) => m.id !== moduleId)

  return (
    <form
      id={MODULE_REQUIREMENTS_FORM_ID}
      className="space-y-3 text-start"
      noValidate
      onSubmit={(e) => {
        e.preventDefault()
        e.stopPropagation()
        void save()
      }}
    >
      <p className="text-xs text-slate-500">
        Choose how students complete this module and any modules they must finish first.
      </p>
      {loading ? <p className="text-xs text-slate-500">Loading requirements…</p> : null}
      {err ? (
        <p className="text-sm text-rose-700" role="alert">
          {err}
        </p>
      ) : null}
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
          disabled={loading}
          className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:cursor-not-allowed disabled:opacity-60"
        >
          {MODES.map((m) => (
            <option key={m.value} value={m.value}>
              {m.label}
            </option>
          ))}
        </select>
      </label>
      {otherModules.length > 0 ? (
        <fieldset disabled={loading}>
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
          disabled={loading}
          className="mt-1 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 disabled:cursor-not-allowed disabled:opacity-60"
        />
      </label>
      {/* Enables Enter-to-submit; footer Save calls registerSubmit instead. */}
      <button type="submit" className="sr-only" tabIndex={-1}>
        Save requirements
      </button>
    </form>
  )
}
