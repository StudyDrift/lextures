import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolMyProgress,
  type ContentToolStudentProgress,
} from '../../../lib/courses-api'

export type StudentProgressStripProps = {
  courseCode: string
  itemId: string
}

export function StudentProgressStrip({ courseCode, itemId }: StudentProgressStripProps) {
  const { t } = useTranslation('contentTools')
  const [progress, setProgress] = useState<ContentToolStudentProgress | null>(null)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const p = await fetchContentToolMyProgress(courseCode, itemId)
        if (!cancelled) setProgress(p)
      } catch {
        if (!cancelled) setProgress(null)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, itemId])

  if (!progress || progress.total === 0) return null
  const firstIncomplete = progress.tools.find((x) => !x.completed)

  return (
    <div
      className="mb-3 flex flex-wrap items-center gap-3 rounded-md border border-slate-200 bg-slate-50 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
      data-testid="student-progress-strip"
    >
      <span className="font-medium text-slate-800 dark:text-neutral-100">
        {t('contentTools.analytics.progressSummary', {
          completed: progress.completed,
          total: progress.total,
        })}
      </span>
      {progress.tools.some((x) => x.countsForGrade) ? (
        <span className="text-xs font-medium text-amber-800 dark:text-amber-200">
          {t('contentTools.grading.countsBadge')}
        </span>
      ) : null}
      {firstIncomplete ? (
        <a
          href={`#lex-tool-${firstIncomplete.instanceId}`}
          className="text-xs text-sky-700 underline dark:text-sky-300"
        >
          {t('contentTools.analytics.continueNext')}
        </a>
      ) : null}
    </div>
  )
}
