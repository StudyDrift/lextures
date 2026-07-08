import { useTranslation } from 'react-i18next'

export function IntroCourseProgressBar({
  percent,
  modulesComplete,
  modulesTotal,
  label,
}: {
  percent: number
  modulesComplete: number
  modulesTotal: number
  label?: string
}) {
  const { t } = useTranslation('introCourse')
  const clamped = Math.max(0, Math.min(100, Math.round(percent)))
  const ariaLabel =
    label ??
    t('introCourse.progress.ariaLabel', {
      complete: modulesComplete,
      total: modulesTotal,
      percent: clamped,
    })

  return (
    <div className="space-y-1">
      <p className="text-xs font-medium text-slate-600 dark:text-neutral-400">
        {t('introCourse.progress.modules', { complete: modulesComplete, total: modulesTotal })}
        <span className="mx-1.5 text-slate-300 dark:text-neutral-600" aria-hidden>
          ·
        </span>
        <span className="tabular-nums">{clamped}%</span>
      </p>
      <div
        role="progressbar"
        aria-valuenow={clamped}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-label={ariaLabel}
        className="h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-neutral-700"
      >
        <div
          className="h-full rounded-full bg-sky-500 motion-safe:transition-[width] motion-safe:duration-300"
          style={{ width: `${clamped}%` }}
        />
      </div>
    </div>
  )
}