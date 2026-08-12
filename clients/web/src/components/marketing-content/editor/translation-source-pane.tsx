import { useTranslation } from 'react-i18next'
import type { MarketingArticle } from '../../../lib/marketing-content-api'
import { Badge, Button } from '../../ui'
import { ArticlePreview } from './article-preview'

type Props = {
  source: MarketingArticle
  stale: boolean
  canAuthor: boolean
  marking: boolean
  onMarkSynced: () => void
}

export function TranslationSourcePane({ source, stale, canAuthor, marking, onMarkSynced }: Props) {
  const { t } = useTranslation('common')
  return (
    <section className="min-w-0 overflow-hidden rounded-2xl border border-border-default bg-surface-sunken" aria-label={t('marketingContent.translations.sourcePane.title', { defaultValue: 'Source article' })}>
      <div className="flex flex-wrap items-center gap-2 border-b border-border-subtle px-4 py-3">
        <h2 className="text-sm font-semibold text-fg-default">
          {t('marketingContent.translations.sourcePane.title', {
            defaultValue: 'Source: {{locale}}, revision {{revision}}',
            locale: source.locale.toUpperCase(),
            revision: source.revisionNo,
          })}
        </h2>
        {stale ? <Badge tone="warning">{t('marketingContent.translations.stale', { defaultValue: 'Stale' })}</Badge> : null}
        {canAuthor ? (
          <Button className="ms-auto" size="sm" variant="secondary" loading={marking} onClick={onMarkSynced}>
            {t('marketingContent.translations.markSynced', { defaultValue: 'Mark synced' })}
          </Button>
        ) : null}
      </div>
      <div className="max-h-[52vh] overflow-auto">
        <ArticlePreview title={source.title} body={source.bodyMd} dir="ltr" />
      </div>
    </section>
  )
}
