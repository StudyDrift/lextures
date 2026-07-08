import { Circle, CircleDot, Disc } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { confidenceLevel } from '../../../lib/learner-profile-format'

type Props = {
  score: number
  className?: string
}

export function ConfidenceIndicator({ score, className = '' }: Props) {
  const { t } = useTranslation('learnerProfile')
  const level = confidenceLevel(score)
  const label =
    level === 'high'
      ? t('learnerProfile.confidence.high')
      : level === 'medium'
        ? t('learnerProfile.confidence.medium')
        : t('learnerProfile.confidence.low')

  const Icon = level === 'high' ? Disc : level === 'medium' ? CircleDot : Circle

  return (
    <span
      className={`inline-flex items-center gap-1.5 text-xs font-medium text-slate-600 dark:text-neutral-300 ${className}`}
      title={t('learnerProfile.confidence.label', { level: label })}
    >
      <Icon className="h-3.5 w-3.5 shrink-0" aria-hidden />
      <span>{label}</span>
    </span>
  )
}