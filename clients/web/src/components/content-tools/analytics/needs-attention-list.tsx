import { useTranslation } from 'react-i18next'

export type NeedsAttentionListProps = {
  items: Array<{ enrollmentId: string; displayName: string; reason: string }>
}

export function NeedsAttentionList({ items }: NeedsAttentionListProps) {
  const { t } = useTranslation('contentTools')
  if (items.length === 0) {
    return (
      <p
        className="rounded-xl border border-dashed border-slate-200 bg-slate-50 px-3 py-2.5 text-sm text-slate-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-400"
        data-testid="needs-attention-empty"
      >
        {t('contentTools.analytics.needsAttentionEmpty')}
      </p>
    )
  }
  return (
    <ul className="divide-y divide-slate-100 overflow-hidden rounded-xl border border-slate-200 dark:divide-neutral-800 dark:border-neutral-700" data-testid="needs-attention-list">
      {items.map((item) => (
        <li
          key={`${item.enrollmentId}-${item.reason}`}
          className="flex items-baseline justify-between gap-2 bg-white px-3 py-2 text-sm dark:bg-neutral-950/40"
        >
          <span className="font-medium text-slate-800 dark:text-neutral-100">
            {item.displayName || item.enrollmentId}
          </span>
          <span className="text-xs text-slate-500 dark:text-neutral-400">
            {t(`contentTools.analytics.reason.${item.reason}`, {
              defaultValue: item.reason,
            })}
          </span>
        </li>
      ))}
    </ul>
  )
}
