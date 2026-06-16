import { type FormEvent, useCallback, useEffect, useState } from 'react'
import { Loader2, Save } from 'lucide-react'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import {
  fetchCourseConsortiumSettings,
  patchCourseConsortiumSettings,
} from '../../lib/consortium-api'

type CourseConsortiumSettingsSectionProps = {
  courseCode: string
}

export function CourseConsortiumSettingsSection({ courseCode }: CourseConsortiumSettingsSectionProps) {
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [shareable, setShareable] = useState(false)

  const reload = useCallback(async () => {
    setLoading(true)
    try {
      const data = await fetchCourseConsortiumSettings(courseCode)
      if (data) setShareable(data.consortiumShareable)
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not load consortium settings.')
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
      await patchCourseConsortiumSettings(courseCode, shareable)
      toastSaveOk('Consortium settings saved.')
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
        Loading consortium settings…
      </p>
    )
  }

  return (
    <form onSubmit={(e) => void onSubmit(e)} className="max-w-xl space-y-4 rounded-2xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
      <div>
        <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-50">Consortium enrollment</h2>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Allow students from partner institutions with an active sharing agreement to enroll in this course.
        </p>
      </div>
      <label className="flex items-center gap-3 text-sm text-slate-800 dark:text-neutral-100">
        <input
          type="checkbox"
          checked={shareable}
          onChange={(e) => setShareable(e.target.checked)}
          className="h-4 w-4 rounded border-slate-300"
        />
        Allow consortium enrollment
      </label>
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
