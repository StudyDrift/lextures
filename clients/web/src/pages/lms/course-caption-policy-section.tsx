import { useCallback, useState } from 'react'
import { patchCourseCaptionPolicy } from '../../lib/captions-api'
import { videoCaptionsFeatureEnabled } from '../../lib/platform-features'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import type { CoursePublic } from '../../lib/courses-api'

type Props = {
  courseCode: string
  course: CoursePublic
  onCourseUpdated: (c: CoursePublic) => void
}

export function CourseCaptionPolicySection({ courseCode, course, onCourseUpdated }: Props) {
  const [saving, setSaving] = useState(false)
  const enabled = course.requireCaptions === true

  const toggle = useCallback(async () => {
    setSaving(true)
    try {
      await patchCourseCaptionPolicy(courseCode, !enabled)
      onCourseUpdated({ ...course, requireCaptions: !enabled })
      toastSaveOk('Caption policy updated.')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not update caption policy.')
    } finally {
      setSaving(false)
    }
  }, [course, courseCode, enabled, onCourseUpdated])

  if (!videoCaptionsFeatureEnabled()) return null

  return (
    <section className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-neutral-700 dark:bg-neutral-900">
      <h2 className="text-lg font-semibold text-slate-900 dark:text-neutral-100">Video captions</h2>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Require captions before publishing module items that embed course videos (WCAG 1.2.2).
      </p>
      <div className="mt-4 flex flex-wrap items-center justify-between gap-4">
        <p className="text-sm text-slate-700 dark:text-neutral-200">
          Mandatory captions for this course
        </p>
        <button
          type="button"
          role="switch"
          aria-checked={enabled}
          disabled={saving}
          onClick={() => void toggle()}
          className={`relative inline-flex h-7 w-12 shrink-0 rounded-full border-2 border-transparent transition-colors ${
            enabled ? 'bg-indigo-600' : 'bg-slate-200 dark:bg-neutral-700'
          }`}
        >
          <span
            className={`inline-block h-6 w-6 transform rounded-full bg-white shadow transition ${
              enabled ? 'translate-x-5' : 'translate-x-0.5'
            }`}
          />
        </button>
      </div>
    </section>
  )
}
