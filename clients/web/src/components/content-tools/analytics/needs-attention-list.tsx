import { useTranslation } from 'react-i18next'

export type NeedsAttentionListProps = {
  items: Array<{ enrollmentId: string; displayName: string; reason: string }>
}

export function NeedsAttentionList({ items }: NeedsAttentionListProps) {
  const { t } = useTranslation('contentTools')
  if (items.length === 0) {
    return (
      <p className="text-sm text-slate-500" data-testid="needs-attention-empty">
        {t('contentTools.analytics.needsAttentionEmpty')}
      </p>
    )
  }
  return (
    <ul className="space-y-1" data-testid="needs-attention-list">
      {items.map((item) => (
        <li
          key={`${item.enrollmentId}-${item.reason}`}
          className="flex items-baseline justify-between gap-2 text-sm"
        >
          <span className="font-medium text-slate-800 dark:text-neutral-100">
            {item.displayName || item.enrollmentId}
          </span>
          <span className="text-xs text-slate-500">
            {t(`contentTools.analytics.reason.${item.reason}`, {
              defaultValue: item.reason,
            })}
          </span>
        </li>
      ))}
    </ul>
  )
}
