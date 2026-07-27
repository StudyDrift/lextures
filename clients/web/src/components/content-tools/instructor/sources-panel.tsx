import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolsContextSources,
  patchContentToolsContextSource,
  reingestContentToolsContextSource,
  type ContentToolsContextSource,
} from '../../../lib/courses-api'

type Props = {
  courseCode: string
  itemId: string
  instanceId?: string
}

export function SourcesPanel({ courseCode, itemId, instanceId }: Props) {
  const { t } = useTranslation('contentTools')
  const [items, setItems] = useState<ContentToolsContextSource[]>([])
  const [totalTokens, setTotalTokens] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [drawer, setDrawer] = useState<ContentToolsContextSource | null>(null)
  const [busyId, setBusyId] = useState<string | null>(null)

  const load = async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetchContentToolsContextSources(courseCode, itemId, instanceId)
      setItems(res.items)
      setTotalTokens(res.totalTokens)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.context.loadError'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps -- reload when item/instance changes
  }, [courseCode, itemId, instanceId])

  const onExclude = async (row: ContentToolsContextSource, excluded: boolean) => {
    setBusyId(row.id)
    try {
      const updated = await patchContentToolsContextSource(courseCode, row.id, excluded)
      setItems((prev) => prev.map((it) => (it.id === row.id ? { ...it, ...updated } : it)))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.context.actionError'))
    } finally {
      setBusyId(null)
    }
  }

  const onReingest = async (row: ContentToolsContextSource) => {
    setBusyId(row.id)
    try {
      const updated = await reingestContentToolsContextSource(courseCode, row.id)
      setItems((prev) => prev.map((it) => (it.id === row.id ? { ...it, ...updated } : it)))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.context.actionError'))
    } finally {
      setBusyId(null)
    }
  }

  const statusLabel = (status: string) => {
    switch (status) {
      case 'ready':
        return t('contentTools.context.status.ready')
      case 'pending':
        return t('contentTools.context.status.pending')
      case 'blocked':
        return t('contentTools.context.status.blocked')
      case 'failed':
        return t('contentTools.context.status.failed')
      case 'unsupported':
        return t('contentTools.context.status.unsupported')
      default:
        return status
    }
  }

  return (
    <section className="mt-6" data-testid="content-tools-sources-panel" aria-labelledby="ct-sources-heading">
      <div className="mb-2 flex flex-wrap items-end justify-between gap-2">
        <div>
          <h2 id="ct-sources-heading" className="text-base font-semibold text-slate-900 dark:text-neutral-100">
            {t('contentTools.context.sourcesTitle')}
          </h2>
          <p className="mt-0.5 text-sm text-slate-600 dark:text-neutral-300">
            {t('contentTools.context.sourcesHelp')}
          </p>
        </div>
        <p className="text-xs text-slate-500 dark:text-neutral-400" data-testid="ct-sources-token-budget">
          {t('contentTools.context.tokenBudget', { count: totalTokens })}
        </p>
      </div>
      {loading ? <p className="text-sm text-slate-500">{t('contentTools.context.loading')}</p> : null}
      {error ? (
        <p className="text-sm text-rose-600" role="alert">
          {error}
        </p>
      ) : null}
      {!loading && items.length === 0 ? (
        <p className="text-sm text-slate-500" data-testid="ct-sources-empty">
          {t('contentTools.context.empty')}
        </p>
      ) : null}
      {items.length > 0 ? (
        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm" aria-label={t('contentTools.context.tableLabel')}>
            <thead>
              <tr className="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500 dark:border-neutral-700">
                <th className="px-2 py-2 font-medium">{t('contentTools.context.colTitle')}</th>
                <th className="px-2 py-2 font-medium">{t('contentTools.context.colHost')}</th>
                <th className="px-2 py-2 font-medium">{t('contentTools.context.colOrigin')}</th>
                <th className="px-2 py-2 font-medium">{t('contentTools.context.colStatus')}</th>
                <th className="px-2 py-2 font-medium">{t('contentTools.context.colActions')}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((row) => (
                <tr
                  key={row.id}
                  className="border-b border-slate-100 dark:border-neutral-800"
                  data-testid={`ct-source-row-${row.id}`}
                  data-status={row.status}
                  data-excluded={row.excluded ? '1' : '0'}
                >
                  <td className="px-2 py-2">
                    <div className="font-medium text-slate-800 dark:text-neutral-100">
                      {row.title || row.url}
                    </div>
                    {row.error ? (
                      <div className="mt-0.5 text-xs text-rose-600" data-testid="ct-source-error">
                        {row.error}
                      </div>
                    ) : null}
                  </td>
                  <td className="px-2 py-2 text-slate-600 dark:text-neutral-300">{row.host || '—'}</td>
                  <td className="px-2 py-2 text-slate-600 dark:text-neutral-300">{row.origin}</td>
                  <td className="px-2 py-2">
                    <span className="inline-flex items-center gap-1">
                      <span aria-hidden="true">
                        {row.status === 'ready' ? '●' : row.status === 'pending' ? '◌' : '!'}
                      </span>
                      <span>{statusLabel(row.status)}</span>
                      {row.excluded ? (
                        <span className="ml-1 text-xs text-slate-500">
                          ({t('contentTools.context.excluded')})
                        </span>
                      ) : null}
                    </span>
                  </td>
                  <td className="px-2 py-2">
                    <div className="flex flex-wrap gap-2">
                      <a
                        href={row.url}
                        target="_blank"
                        rel="noreferrer"
                        className="text-xs text-sky-700 underline dark:text-sky-300"
                      >
                        {t('contentTools.context.openOriginal')}
                      </a>
                      <button
                        type="button"
                        className="text-xs text-sky-700 underline dark:text-sky-300 disabled:opacity-50"
                        disabled={busyId === row.id}
                        onClick={() => void onReingest(row)}
                      >
                        {t('contentTools.context.reingest')}
                      </button>
                      <button
                        type="button"
                        className="text-xs text-sky-700 underline dark:text-sky-300 disabled:opacity-50"
                        disabled={busyId === row.id}
                        onClick={() => void onExclude(row, !row.excluded)}
                        data-testid={`ct-source-exclude-${row.id}`}
                      >
                        {row.excluded
                          ? t('contentTools.context.include')
                          : t('contentTools.context.exclude')}
                      </button>
                      <button
                        type="button"
                        className="text-xs text-sky-700 underline dark:text-sky-300"
                        onClick={() => setDrawer(row)}
                      >
                        {t('contentTools.context.viewExtracted')}
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {drawer ? (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="ct-extracted-title"
          className="fixed inset-0 z-40 flex items-end justify-center bg-black/40 p-0 sm:items-center sm:p-4"
          data-testid="ct-extracted-drawer"
        >
          <div className="max-h-[90vh] w-full max-w-2xl overflow-auto rounded-t-xl bg-white p-4 shadow-lg sm:rounded-xl dark:bg-neutral-900">
            <div className="mb-3 flex items-start justify-between gap-2">
              <h3 id="ct-extracted-title" className="text-base font-semibold">
                {t('contentTools.context.extractedTitle')}
              </h3>
              <button
                type="button"
                className="text-sm text-slate-600 underline dark:text-neutral-300"
                onClick={() => setDrawer(null)}
              >
                {t('contentTools.context.close')}
              </button>
            </div>
            <pre className="whitespace-pre-wrap text-sm text-slate-800 dark:text-neutral-100">
              {drawer.extractedText || t('contentTools.context.noExtracted')}
            </pre>
          </div>
        </div>
      ) : null}
    </section>
  )
}
