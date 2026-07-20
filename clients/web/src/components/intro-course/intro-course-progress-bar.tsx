import { useTranslation } from 'react-i18next'
import { AnimatedProgress } from '../ui/animated-progress'

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
      <AnimatedProgress
        value={clamped}
        label={ariaLabel}
        fillClassName="h-full rounded-full bg-sky-500"
      />
    </div>
  )
}
