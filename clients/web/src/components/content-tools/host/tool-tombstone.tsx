import { useTranslation } from 'react-i18next'

type Props = {
  toolName?: string
  developerName?: string
}

/** Read-only tombstone when a third-party tool is revoked (CT.9 FR-9 / AC-7). */
export function ToolTombstone({ toolName, developerName }: Props) {
  const { t } = useTranslation('contentTools')
  return (
    <div
      className="rounded border border-amber-300 bg-amber-50 p-4 text-sm text-amber-950 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100"
      role="status"
      data-testid="content-tool-tombstone"
    >
      <p className="font-medium">{t('contentTools.marketplace.tombstoneTitle')}</p>
      <p className="mt-1">
        {t('contentTools.marketplace.tombstoneBody', {
          name: toolName || t('contentTools.marketplace.tombstoneFallbackName'),
        })}
      </p>
      {developerName ? (
        <p className="mt-2 text-xs opacity-80">
          {t('contentTools.marketplace.providedBy', { developer: developerName })}
        </p>
      ) : null}
      <p className="mt-2 text-xs">{t('contentTools.marketplace.tombstoneExportHint')}</p>
    </div>
  )
}
