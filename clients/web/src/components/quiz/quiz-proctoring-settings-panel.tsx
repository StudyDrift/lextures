import { useCallback, useEffect, useState } from 'react'
import { Shield, Trash2 } from 'lucide-react'
import type { ProctoringConfig, ProctoringVendor } from '../../lib/courses-api'
import {
  deleteQuizProctoringConfig,
  fetchQuizProctoringConfig,
  saveQuizProctoringConfig,
} from '../../lib/courses-api'

type LTITool = {
  id: string
  name: string
}

export type QuizProctoringSettingsPanelProps = {
  courseCode: string
  itemId: string
  availableTools: LTITool[]
}

const VENDOR_OPTIONS: { value: ProctoringVendor; label: string }[] = [
  { value: 'honorlock', label: 'Honorlock' },
  { value: 'respondus', label: 'Respondus Monitor' },
  { value: 'proctu', label: 'ProctorU' },
  { value: 'examity', label: 'Examity' },
]

export function QuizProctoringSettingsPanel({
  courseCode,
  itemId,
  availableTools,
}: QuizProctoringSettingsPanelProps) {
  const [config, setConfig] = useState<ProctoringConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Form state
  const [selectedToolId, setSelectedToolId] = useState('')
  const [selectedVendor, setSelectedVendor] = useState<ProctoringVendor>('honorlock')
  const [required, setRequired] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const cfg = await fetchQuizProctoringConfig(courseCode, itemId)
      setConfig(cfg)
      if (cfg) {
        setSelectedToolId(cfg.externalToolId)
        setSelectedVendor(cfg.vendor)
        setRequired(cfg.required)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load proctoring config.')
    } finally {
      setLoading(false)
    }
  }, [courseCode, itemId])

  useEffect(() => {
    void load()
  }, [load])

  const handleSave = useCallback(async () => {
    if (!selectedToolId) return
    setSaving(true)
    setError(null)
    try {
      const saved = await saveQuizProctoringConfig(courseCode, itemId, {
        externalToolId: selectedToolId,
        vendor: selectedVendor,
        required,
      })
      setConfig(saved)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save proctoring config.')
    } finally {
      setSaving(false)
    }
  }, [courseCode, itemId, selectedToolId, selectedVendor, required])

  const handleRemove = useCallback(async () => {
    setSaving(true)
    setError(null)
    try {
      await deleteQuizProctoringConfig(courseCode, itemId)
      setConfig(null)
      setSelectedToolId('')
      setSelectedVendor('honorlock')
      setRequired(false)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to remove proctoring config.')
    } finally {
      setSaving(false)
    }
  }, [courseCode, itemId])

  if (loading) {
    return (
      <div className="flex items-center gap-2 py-2 text-xs text-slate-400 dark:text-neutral-500">
        <span className="h-3 w-3 motion-safe:animate-spin rounded-full border-2 border-slate-200 border-t-indigo-500" aria-hidden />
        Loading proctoring settings…
      </div>
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2">
        <Shield className="h-4 w-4 shrink-0 text-indigo-500 dark:text-indigo-400" aria-hidden />
        <span className="text-xs font-medium text-slate-700 dark:text-neutral-300">Proctoring</span>
        {config && (
          <span className="ms-auto rounded-full bg-emerald-100 px-2 py-0.5 text-[10px] font-semibold text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-400">
            Active
          </span>
        )}
      </div>

      {availableTools.length === 0 ? (
        <p className="text-xs text-slate-500 dark:text-neutral-400">
          No proctoring tools are registered. Ask your admin to add a proctoring vendor via LTI settings.
        </p>
      ) : (
        <div className="space-y-2">
          <div>
            <label className="block text-[11px] font-medium text-slate-600 dark:text-neutral-400">
              LTI Tool
            </label>
            <select
              value={selectedToolId}
              onChange={(e) => setSelectedToolId(e.target.value)}
              disabled={saving}
              className="mt-0.5 w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-xs text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
            >
              <option value="">Select a tool…</option>
              {availableTools.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
          </div>

          <div>
            <label className="block text-[11px] font-medium text-slate-600 dark:text-neutral-400">
              Vendor
            </label>
            <select
              value={selectedVendor}
              onChange={(e) => setSelectedVendor(e.target.value as ProctoringVendor)}
              disabled={saving}
              className="mt-0.5 w-full rounded-lg border border-slate-200 bg-white px-2 py-1.5 text-xs text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-400 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
            >
              {VENDOR_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
          </div>

          <label className="flex items-center gap-2 text-xs text-slate-700 dark:text-neutral-300">
            <input
              type="checkbox"
              checked={required}
              onChange={(e) => setRequired(e.target.checked)}
              disabled={saving}
              className="h-3.5 w-3.5 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
            />
            Require proctoring (block quiz if launch fails)
          </label>

          {error && (
            <p className="text-xs text-rose-700 dark:text-rose-400" role="alert">
              {error}
            </p>
          )}

          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void handleSave()}
              disabled={saving || !selectedToolId}
              className="rounded-lg bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white shadow-sm hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {saving ? 'Saving…' : config ? 'Update' : 'Attach Proctoring'}
            </button>
            {config && (
              <button
                type="button"
                onClick={() => void handleRemove()}
                disabled={saving}
                className="flex items-center gap-1 rounded-lg px-3 py-1.5 text-xs font-medium text-rose-700 hover:bg-rose-50 disabled:opacity-50 dark:text-rose-400 dark:hover:bg-rose-950/40"
              >
                <Trash2 className="h-3 w-3" aria-hidden />
                Remove
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
