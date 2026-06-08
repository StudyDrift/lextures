import { type FormEvent, useCallback, useEffect, useState } from 'react'
import { Loader2, Save } from 'lucide-react'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import {
  fetchCoursePlagiarismSettings,
  patchCoursePlagiarismSettings,
  type CoursePlagiarismSettings,
} from '../../lib/courses-api'

type CoursePlagiarismSettingsSectionProps = {
  courseCode: string
}

const PROVIDERS = [
  { value: '', label: 'Use institution default' },
  { value: 'none', label: 'None (internal AI only)' },
  { value: 'turnitin', label: 'Turnitin' },
  { value: 'copyleaks', label: 'Copyleaks' },
  { value: 'gptzero', label: 'GPTZero' },
] as const

export function CoursePlagiarismSettingsSection({ courseCode }: CoursePlagiarismSettingsSectionProps) {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [enabled, setEnabled] = useState(true)
  const [provider, setProvider] = useState('')
  const [threshold, setThreshold] = useState('40')

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchCoursePlagiarismSettings(courseCode)
      if (data) {
        setEnabled(data.plagiarismChecksEnabled)
        setProvider(data.plagiarismProvider ?? '')
        setThreshold(String(data.plagiarismAlertThresholdPct ?? 40))
      }
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not load plagiarism settings.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  useEffect(() => {
    void reload()
  }, [reload])

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    try {
      const thresholdNum = Number.parseFloat(threshold)
      const body: Partial<CoursePlagiarismSettings> = {
        plagiarismChecksEnabled: enabled,
        plagiarismProvider: provider || null,
        plagiarismAlertThresholdPct: Number.isFinite(thresholdNum) ? thresholdNum : 40,
      }
      await patchCoursePlagiarismSettings(courseCode, body)
      toastSaveOk('Plagiarism settings saved.')
      await reload()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Save failed.')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <p className="flex items-center gap-2 text-sm text-slate-600 dark:text-neutral-300">
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
        Loading plagiarism settings…
      </p>
    )
  }

  return (
    <form onSubmit={(e) => void onSubmit(e)} className="max-w-xl space-y-6">
      <div>
        <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-50">Plagiarism &amp; AI checks</h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Configure course-wide originality scanning. Assignments can still opt out individually.
        </p>
      </div>
      <label className="flex items-center gap-3 text-sm text-slate-800 dark:text-neutral-100">
        <input
          type="checkbox"
          checked={enabled}
          onChange={(e) => setEnabled(e.target.checked)}
          className="h-4 w-4 rounded border-slate-300"
        />
        Enable plagiarism and AI-authorship checks for this course
      </label>
      <div className="space-y-1.5">
        <label htmlFor="plagiarism-provider" className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
          External provider
        </label>
        <select
          id="plagiarism-provider"
          value={provider}
          onChange={(e) => setProvider(e.target.value)}
          className="w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
        >
          {PROVIDERS.map((p) => (
            <option key={p.value || 'default'} value={p.value}>
              {p.label}
            </option>
          ))}
        </select>
      </div>
      <div className="space-y-1.5">
        <label htmlFor="plagiarism-threshold" className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
          Instructor alert threshold (% similarity)
        </label>
        <input
          id="plagiarism-threshold"
          type="number"
          min={0}
          max={100}
          step={1}
          value={threshold}
          onChange={(e) => setThreshold(e.target.value)}
          className="w-32 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
        />
      </div>
      <button
        type="submit"
        disabled={saving}
        className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
      >
        {saving ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <Save className="h-4 w-4" aria-hidden />}
        Save settings
      </button>
    </form>
  )
}
