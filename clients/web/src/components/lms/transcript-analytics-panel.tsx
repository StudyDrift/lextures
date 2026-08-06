import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Loader2 } from 'lucide-react'
import {
  downloadAdminTranscriptAnalyticsExport,
  fetchAdminTranscriptDashboard,
  fetchAdminTranscriptDashboardDrilldown,
  fetchAdminTranscriptHealth,
  type TranscriptDashboardSummary,
  type TranscriptDrillDownOrder,
  type TranscriptHealthSummary,
} from '../../lib/transcripts-api'

function isoDate(d: Date): string {
  return d.toISOString().slice(0, 10)
}

function defaultRange(): { from: string; to: string } {
  const to = new Date()
  const from = new Date()
  from.setUTCDate(from.getUTCDate() - 30)
  return { from: isoDate(from), to: isoDate(to) }
}

function formatMoney(minor: number, currency: string): string {
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: (currency || 'usd').toUpperCase(),
    }).format(minor / 100)
  } catch {
    return `${(minor / 100).toFixed(2)} ${currency}`
  }
}

function formatPct(rate: number): string {
  return `${(rate * 100).toFixed(1)}%`
}

function formatHours(h: number): string {
  if (!Number.isFinite(h) || h <= 0) return '—'
  if (h < 48) return `${h.toFixed(1)}h`
  return `${(h / 24).toFixed(1)}d`
}

type Props = {
  enabled: boolean
  onOpenQueue?: (orderId?: string) => void
}

export function TranscriptAnalyticsPanel({ enabled, onOpenQueue }: Props) {
  const { t } = useTranslation('common')
  const titleId = useId()
  const [range, setRange] = useState(defaultRange)
  const [data, setData] = useState<TranscriptDashboardSummary | null>(null)
  const [health, setHealth] = useState<TranscriptHealthSummary | null>(null)
  const [drillMetric, setDrillMetric] = useState<string | null>(null)
  const [drillOrders, setDrillOrders] = useState<TranscriptDrillDownOrder[]>([])
  const [loading, setLoading] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!enabled) return
    setLoading(true)
    setError(null)
    void Promise.all([
      fetchAdminTranscriptDashboard({ from: range.from, to: range.to }),
      fetchAdminTranscriptHealth(),
    ])
      .then(([dash, h]) => {
        setData(dash)
        setHealth(h)
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : t('transcripts.analytics.loadError'))
      })
      .finally(() => setLoading(false))
  }, [enabled, range.from, range.to, t])

  async function openDrill(metric: string) {
    setDrillMetric(metric)
    setDrillOrders([])
    try {
      const orders = await fetchAdminTranscriptDashboardDrilldown({
        metric,
        from: range.from,
        to: range.to,
      })
      setDrillOrders(orders)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('transcripts.analytics.drillError'))
    }
  }

  if (!enabled) return null

  const maxOrders = Math.max(1, ...(data?.daily.map((d) => d.orders) ?? [1]))
  const maxMethod = Math.max(1, ...(data?.methodMix.map((m) => m.count) ?? [1]))

  return (
    <section aria-labelledby={titleId} className="mt-6 space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div>
          <h2 id={titleId} className="text-lg font-semibold text-fg-default">
            {t('transcripts.analytics.title')}
          </h2>
          <p className="mt-1 text-sm text-fg-muted">
            {t('transcripts.analytics.subtitle')}
          </p>
        </div>
        <div className="flex flex-wrap items-end gap-2">
          <label className="text-sm">
            <span className="block text-fg-muted">{t('transcripts.analytics.from')}</span>
            <input
              type="date"
              value={range.from}
              onChange={(e) => setRange((r) => ({ ...r, from: e.target.value }))}
              className="mt-1 rounded-md border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-raised"
            />
          </label>
          <label className="text-sm">
            <span className="block text-fg-muted">{t('transcripts.analytics.to')}</span>
            <input
              type="date"
              value={range.to}
              onChange={(e) => setRange((r) => ({ ...r, to: e.target.value }))}
              className="mt-1 rounded-md border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-raised"
            />
          </label>
          {data?.panels.export !== false ? (
            <button
              type="button"
              disabled={exporting || loading}
              onClick={() => {
                setExporting(true)
                void downloadAdminTranscriptAnalyticsExport({
                  from: range.from,
                  to: range.to,
                })
                  .catch((err: unknown) =>
                    setError(err instanceof Error ? err.message : t('transcripts.analytics.exportError')),
                  )
                  .finally(() => setExporting(false))
              }}
              className="rounded-md border border-border-strong px-3 py-1.5 text-sm font-medium hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:hover:bg-surface-raised"
            >
              {t('transcripts.analytics.exportCsv')}
            </button>
          ) : null}
        </div>
      </div>

      {error ? (
        <p className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-100" role="alert">
          {error}
        </p>
      ) : null}

      {loading ? (
        <p className="flex items-center gap-2 text-sm text-fg-muted" role="status">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
          {t('common.loading')}
        </p>
      ) : null}

      {data?.stale ? (
        <p className="rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-100" role="status">
          {t('transcripts.analytics.staleWarning')}
          {data.lastRefreshedAt
            ? ` (${new Date(data.lastRefreshedAt).toLocaleString()})`
            : ''}
        </p>
      ) : null}

      {health ? (
        <div
          className={`rounded-lg border p-4 ${ health.anyAlert ? 'border-amber-300 bg-amber-50 dark:border-amber-800 dark:bg-amber-950' : 'border-border-default dark:border-border-subtle' }`}
          role="region"
          aria-label={t('transcripts.analytics.healthTitle')}
        >
          <h3 className="text-sm font-semibold text-fg-default">
            {t('transcripts.analytics.healthTitle')}
            {health.anyAlert ? (
              <span className="ml-2 text-amber-800 dark:text-amber-200">
                {t('transcripts.analytics.healthAlert')}
              </span>
            ) : null}
          </h3>
          <dl className="mt-3 grid grid-cols-2 gap-3 sm:grid-cols-4">
            <div>
              <dt className="text-xs text-fg-muted">{t('transcripts.analytics.backlog')}</dt>
              <dd className="text-lg font-semibold">
                <button
                  type="button"
                  className="underline-offset-2 hover:underline"
                  onClick={() => onOpenQueue?.()}
                >
                  {health.backlogCount}
                </button>
              </dd>
            </div>
            <div>
              <dt className="text-xs text-fg-muted">{t('transcripts.analytics.oldestPending')}</dt>
              <dd className="text-lg font-semibold">{formatHours(health.oldestPendingAgeHours)}</dd>
            </div>
            <div>
              <dt className="text-xs text-fg-muted">{t('transcripts.analytics.failureRate')}</dt>
              <dd className="text-lg font-semibold">{formatPct(health.deliveryFailureRate)}</dd>
            </div>
            <div>
              <dt className="text-xs text-fg-muted">{t('transcripts.analytics.deadLetters')}</dt>
              <dd className="text-lg font-semibold">{health.deadLetterCount}</dd>
            </div>
          </dl>
        </div>
      ) : null}

      {data && !loading ? (
        data.orders === 0 ? (
          <p className="text-sm text-fg-muted">{t('transcripts.analytics.empty')}</p>
        ) : (
          <div className="space-y-6">
            <dl className="grid grid-cols-2 gap-3 sm:grid-cols-4">
              {(
                [
                  { key: 'orders', label: t('transcripts.analytics.orders'), value: String(data.orders), metric: 'orders' },
                  {
                    key: 'delivered',
                    label: t('transcripts.analytics.delivered'),
                    value: String(data.delivered),
                    metric: 'delivered',
                  },
                  {
                    key: 'turnaround',
                    label: t('transcripts.analytics.avgTurnaround'),
                    value: formatHours(data.turnaround.avgHours),
                    metric: null,
                  },
                  {
                    key: 'revenue',
                    label: t('transcripts.analytics.netRevenue'),
                    value: data.panels.finance
                      ? formatMoney(data.netRevenueMinor, data.currency)
                      : t('transcripts.analytics.financeHidden'),
                    metric: data.panels.finance ? 'refunded' : null,
                  },
                ] as const
              ).map((kpi) => (
                <div key={kpi.key} className="rounded-md bg-surface-base p-3 dark:bg-surface-overlay">
                  <dt className="text-xs text-fg-muted">{kpi.label}</dt>
                  <dd className="text-xl font-semibold text-fg-default">
                    {kpi.metric ? (
                      <button
                        type="button"
                        className="underline-offset-2 hover:underline"
                        onClick={() => {
                          const metric = kpi.metric
                          if (metric) void openDrill(metric)
                        }}
                      >
                        {kpi.value}
                      </button>
                    ) : (
                      kpi.value
                    )}
                  </dd>
                </div>
              ))}
            </dl>

            <div className="grid gap-3 sm:grid-cols-3">
              <p className="text-sm text-fg-muted">
                {t('transcripts.analytics.holdRate')}:{' '}
                <button type="button" className="font-medium underline-offset-2 hover:underline" onClick={() => void openDrill('on_hold')}>
                  {formatPct(data.holdRate)}
                </button>
              </p>
              <p className="text-sm text-fg-muted">
                {t('transcripts.analytics.rejectionRate')}:{' '}
                <button type="button" className="font-medium underline-offset-2 hover:underline" onClick={() => void openDrill('rejected')}>
                  {formatPct(data.rejectionRate)}
                </button>
              </p>
              <p className="text-sm text-fg-muted">
                {t('transcripts.analytics.refundRate')}:{' '}
                {data.panels.finance ? (
                  <button type="button" className="font-medium underline-offset-2 hover:underline" onClick={() => void openDrill('refunded')}>
                    {formatPct(data.refundRate)}
                  </button>
                ) : (
                  t('transcripts.analytics.financeHidden')
                )}
              </p>
            </div>

            <p className="text-sm text-fg-muted">
              {t('transcripts.analytics.turnaroundPercentiles', {
                p50: formatHours(data.turnaround.p50Hours),
                p90: formatHours(data.turnaround.p90Hours),
                p95: formatHours(data.turnaround.p95Hours),
                n: data.turnaround.sampleSize,
              })}
            </p>

            {data.daily.length > 0 ? (
              <section aria-labelledby={`${titleId}-volume`}>
                <h3 id={`${titleId}-volume`} className="text-sm font-medium text-fg-default">
                  {t('transcripts.analytics.volume')}
                </h3>
                <div
                  className="mt-2 flex h-20 items-end gap-1"
                  role="img"
                  aria-label={t('transcripts.analytics.volumeChartAria')}
                >
                  {data.daily.map((d) => (
                    <div
                      key={d.day}
                      className="flex-1 rounded-t bg-sky-600/80 dark:bg-sky-400/70"
                      style={{ height: `${Math.max(8, (d.orders / maxOrders) * 100)}%` }}
                      title={`${d.day}: ${d.orders}`}
                    />
                  ))}
                </div>
                <table className="mt-3 w-full text-left text-xs text-fg-muted">
                  <caption className="sr-only">{t('transcripts.analytics.volumeTableCaption')}</caption>
                  <thead>
                    <tr>
                      <th scope="col">{t('transcripts.analytics.day')}</th>
                      <th scope="col">{t('transcripts.analytics.orders')}</th>
                      <th scope="col">{t('transcripts.analytics.delivered')}</th>
                      {data.panels.finance ? (
                        <th scope="col">{t('transcripts.analytics.netRevenue')}</th>
                      ) : null}
                    </tr>
                  </thead>
                  <tbody>
                    {data.daily.map((d) => (
                      <tr key={d.day} className="border-t border-border-subtle">
                        <td className="py-1">{new Date(d.day).toLocaleDateString()}</td>
                        <td>{d.orders}</td>
                        <td>{d.delivered}</td>
                        {data.panels.finance ? (
                          <td>{formatMoney(d.netRevenueMinor, data.currency)}</td>
                        ) : null}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </section>
            ) : null}

            <div className="grid gap-6 lg:grid-cols-2">
              <section aria-labelledby={`${titleId}-methods`}>
                <h3 id={`${titleId}-methods`} className="text-sm font-medium text-fg-default">
                  {t('transcripts.analytics.methodMix')}
                </h3>
                <ul className="mt-2 space-y-2" aria-label={t('transcripts.analytics.methodMix')}>
                  {data.methodMix.map((m) => (
                    <li key={m.method} className="text-sm">
                      <div className="flex justify-between gap-2">
                        <span>{m.method}</span>
                        <span className="font-medium">{m.count}</span>
                      </div>
                      <div className="mt-1 h-2 rounded bg-surface-sunken" aria-hidden>
                        <div
                          className="h-2 rounded bg-emerald-600 dark:bg-emerald-400"
                          style={{ width: `${(m.count / maxMethod) * 100}%` }}
                        />
                      </div>
                    </li>
                  ))}
                </ul>
                <table className="sr-only">
                  <caption>{t('transcripts.analytics.methodMix')}</caption>
                  <thead>
                    <tr>
                      <th>{t('transcripts.analytics.method')}</th>
                      <th>{t('transcripts.analytics.count')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.methodMix.map((m) => (
                      <tr key={m.method}>
                        <td>{m.method}</td>
                        <td>{m.count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </section>

              <section aria-labelledby={`${titleId}-dest`}>
                <h3 id={`${titleId}-dest`} className="text-sm font-medium text-fg-default">
                  {t('transcripts.analytics.topDestinations')}
                </h3>
                <table className="mt-2 w-full text-left text-sm">
                  <caption className="sr-only">{t('transcripts.analytics.topDestinations')}</caption>
                  <thead>
                    <tr className="text-xs text-fg-muted">
                      <th scope="col">{t('transcripts.analytics.destination')}</th>
                      <th scope="col">{t('transcripts.analytics.count')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.topDestinations.map((d) => (
                      <tr key={`${d.recipientId ?? d.recipientName}`} className="border-t border-border-subtle">
                        <td className="py-1.5">{d.recipientName}</td>
                        <td>{d.count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </section>
            </div>

            {drillMetric ? (
              <section aria-labelledby={`${titleId}-drill`} className="rounded-lg border border-border-default p-4 dark:border-border-subtle">
                <div className="flex items-center justify-between gap-2">
                  <h3 id={`${titleId}-drill`} className="text-sm font-semibold">
                    {t('transcripts.analytics.drillTitle', { metric: drillMetric })}
                  </h3>
                  <button
                    type="button"
                    className="text-xs text-fg-muted hover:underline"
                    onClick={() => setDrillMetric(null)}
                  >
                    {t('dialogs.close')}
                  </button>
                </div>
                {drillOrders.length === 0 ? (
                  <p className="mt-2 text-sm text-fg-muted">{t('transcripts.analytics.drillEmpty')}</p>
                ) : (
                  <ul className="mt-2 divide-y divide-slate-100 dark:divide-neutral-800">
                    {drillOrders.map((o) => (
                      <li key={o.id}>
                        <button
                          type="button"
                          className="flex w-full items-center justify-between gap-2 py-2 text-left text-sm hover:bg-surface-base dark:hover:bg-surface-raised"
                          onClick={() => onOpenQueue?.(o.id)}
                        >
                          <span>
                            {o.userEmail || o.id}
                            <span className="ml-2 text-xs text-fg-muted">{o.status}</span>
                          </span>
                          <span className="text-xs text-fg-subtle">
                            {new Date(o.submittedAt ?? o.createdAt).toLocaleString()}
                          </span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            ) : null}
          </div>
        )
      ) : null}
    </section>
  )
}
