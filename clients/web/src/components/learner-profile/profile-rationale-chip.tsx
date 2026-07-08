import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { profileRationaleFacetPath } from './profile-rationale-utils'

export type ProfileRationale = {
  text: string
  facetKey: string
  insightKey: string
}

type ProfileRationaleChipProps = {
  rationale: ProfileRationale
  className?: string
}

export function ProfileRationaleChip({ rationale, className = '' }: ProfileRationaleChipProps) {
  const { t } = useTranslation('learnerProfile')
  const facetPath = profileRationaleFacetPath(rationale.facetKey)
  return (
    <p
      className={`text-xs text-violet-700 dark:text-violet-300 ${className}`.trim()}
      role="note"
      aria-label={rationale.text}
    >
      <span>{rationale.text}</span>{' '}
      <Link
        to={`/lms/settings/learner-profile/${facetPath}`}
        className="font-medium underline underline-offset-2 hover:text-violet-900 dark:hover:text-violet-100"
      >
        {t('learnerProfile.adaptivity.learnMore')}
      </Link>
    </p>
  )
}