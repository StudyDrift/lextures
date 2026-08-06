import { useEffect, useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  browseToolMarketplace,
  type MarketplaceToolListing,
} from '../../lib/content-tool-marketplace-api'

export default function ToolMarketplacePage() {
  const { t } = useTranslation('contentTools')
  const titleId = useId()
  const { ffContentToolMarketplace, loading: featuresLoading } = usePlatformFeatures()
  const [tools, setTools] = useState<MarketplaceToolListing[]>([])
  const [q, setQ] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (featuresLoading || !ffContentToolMarketplace) return
    let cancelled = false
    setLoading(true)
    void browseToolMarketplace({ q: q.trim() || undefined })
      .then((list) => {
        if (!cancelled) setTools(list)
      })
      .catch((e: unknown) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [ffContentToolMarketplace, featuresLoading, q])

  if (!ffContentToolMarketplace && !featuresLoading) {
    return (
      <div className="mx-auto max-w-3xl p-6" data-testid="tool-marketplace-disabled">
        <p className="text-sm text-fg-muted">{t('contentTools.marketplace.disabled')}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl p-6" data-testid="tool-marketplace-page">
      <h1 id={titleId} className="text-2xl font-semibold text-fg-default dark:text-slate-100">
        {t('contentTools.marketplace.title')}
      </h1>
      <p className="mt-1 text-sm text-fg-muted dark:text-fg-subtle">
        {t('contentTools.marketplace.help')}
      </p>
      <label className="mt-4 block text-sm">
        <span className="sr-only">{t('contentTools.marketplace.search')}</span>
        <input
          data-testid="tool-marketplace-search"
          className="mt-1 w-full rounded border border-border-strong bg-surface-raised px-3 py-2 dark:border-slate-600 dark:bg-slate-900"
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder={t('contentTools.marketplace.search')}
        />
      </label>
      {loading ? <p className="mt-4 text-sm">{t('contentTools.marketplace.loading')}</p> : null}
      {error ? (
        <p className="mt-4 text-sm text-rose-700" role="alert">
          {error}
        </p>
      ) : null}
      <ul className="mt-6 space-y-4" aria-labelledby={titleId}>
        {tools.map((tool) => (
          <li key={tool.toolId} className="border-b border-border-default pb-4 dark:border-slate-700">
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <h2 className="text-lg font-medium text-fg-default dark:text-slate-100">
                {tool.displayName}
              </h2>
              <span className="text-xs text-fg-muted">
                WCAG {tool.wcagLevel} · v{tool.version}
              </span>
            </div>
            <p className="mt-1 text-sm text-fg-muted dark:text-slate-300">{tool.summary}</p>
            <p className="mt-2 text-xs text-fg-muted">{tool.toolId}</p>
            <Link
              className="mt-2 inline-block text-sm font-medium text-sky-700 underline dark:text-sky-300"
              to={`/tool-marketplace/${encodeURIComponent(tool.toolId)}`}
              data-testid={`tool-marketplace-link-${tool.toolId}`}
            >
              {t('contentTools.marketplace.viewDetails')}
            </Link>
          </li>
        ))}
      </ul>
      {!loading && tools.length === 0 ? (
        <p className="mt-4 text-sm text-fg-muted">{t('contentTools.marketplace.empty')}</p>
      ) : null}
    </div>
  )
}
