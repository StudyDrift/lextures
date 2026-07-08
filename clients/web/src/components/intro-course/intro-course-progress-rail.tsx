import { useTranslation } from 'react-i18next'
import { CheckCircle2, Circle, CircleDot } from 'lucide-react'
import { useIntroCourseProgress } from '../../hooks/use-intro-course-progress'
import { INTRO_COURSE_CODE } from '../../lib/intro-course-api'
import { IntroCourseProgressBar } from './intro-course-progress-bar'

function ModuleStatusIcon({ status }: { status: 'done' | 'current' | 'upcoming' }) {
  if (status === 'done') {
    return <CheckCircle2 className="h-4 w-4 shrink-0 text-emerald-600 dark:text-emerald-400" aria-hidden />
  }
  if (status === 'current') {
    return <CircleDot className="h-4 w-4 shrink-0 text-sky-600 dark:text-sky-400" aria-hidden />
  }
  return <Circle className="h-4 w-4 shrink-0 text-slate-300 dark:text-neutral-600" aria-hidden />
}

export function IntroCourseProgressRail({ courseCode }: { courseCode: string }) {
  const { t } = useTranslation('introCourse')
  const isIntroCourse = courseCode === INTRO_COURSE_CODE
  const { progress, loading, error } = useIntroCourseProgress(isIntroCourse)

  if (!isIntroCourse || loading || error || !progress?.enrolled) return null

  const modules = progress.modules ?? []

  return (
    <section
      aria-label={t('introCourse.rail.ariaLabel')}
      className="rounded-xl border border-sky-100 bg-sky-50/50 p-4 dark:border-sky-900/40 dark:bg-sky-950/20"
    >
      <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-50">
        {t('introCourse.rail.title')}
      </h2>
      <div className="mt-3">
        <IntroCourseProgressBar
          percent={progress.percent}
          modulesComplete={progress.modulesComplete}
          modulesTotal={progress.modulesTotal}
        />
      </div>
      {progress.nextItem?.title && !progress.completedAt ? (
        <p className="mt-3 text-xs text-slate-600 dark:text-neutral-400">
          {t('introCourse.rail.nextUp', { title: progress.nextItem.title })}
        </p>
      ) : null}
      {modules.length > 0 ? (
        <ol className="mt-4 space-y-2" aria-label={t('introCourse.rail.modulesAria')}>
          {modules.map((mod) => (
            <li
              key={mod.slug}
              className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-200"
            >
              <ModuleStatusIcon status={mod.status} />
              <span
                className={
                  mod.status === 'current'
                    ? 'font-semibold text-slate-900 dark:text-neutral-50'
                    : mod.status === 'done'
                      ? 'text-slate-600 dark:text-neutral-400'
                      : 'text-slate-500 dark:text-neutral-500'
                }
              >
                {mod.title}
              </span>
              <span className="sr-only">
                {mod.status === 'done'
                  ? t('introCourse.rail.statusDone')
                  : mod.status === 'current'
                    ? t('introCourse.rail.statusCurrent')
                    : t('introCourse.rail.statusUpcoming')}
              </span>
            </li>
          ))}
        </ol>
      ) : null}
    </section>
  )
}