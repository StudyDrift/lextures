import { useTranslation } from 'react-i18next'

export type NeedsAttentionListProps = {
  items: Array<{ enrollmentId: string; displayName: string; reason: string }>
}

export function NeedsAttentionList({ items }: NeedsAttentionListProps) {
  const { t } = useTranslation('contentTools')
  if (items.length === 0) {
    return (
      <p
        className="rounded-xl border border-dashed border-border-default bg-surface-base px-3 py-2.5 text-sm text-fg-muted dark:border-border-default dark:bg-surface-raised dark:text-fg-muted"
        data-testid="needs-attention-empty"
      >
        {t('contentTools.analytics.needsAttentionEmpty')}
      </p>
    )
  }
  return (
    <ul className="divide-y divide-slate-100 overflow-hidden rounded-xl border border-border-default dark:divide-neutral-800 dark:border-border-default" data-testid="needs-attention-list">
      {items.map((item) => (
        <li
          key={`${item.enrollmentId}-${item.reason}`}
          className="flex items-baseline justify-between gap-2 bg-surface-raised px-3 py-2 text-sm/40"
        >
          <span className="font-medium text-fg-default">
            {item.displayName || item.enrollmentId}
          </span>
          <span className="text-xs text-fg-muted">
            {t(`contentTools.analytics.reason.${item.reason}`, {
              defaultValue: item.reason,
            })}
          </span>
        </li>
      ))}
    </ul>
  )
}
