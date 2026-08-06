import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolTelemetry,
  type ContentToolTelemetryRow,
} from '../../../lib/content-tools-admin-api'

/** Platform admin telemetry table (CT.7 FR-15) — counts only, never free-text. */
export function AdminToolTelemetry() {
  const { t } = useTranslation('contentTools')
  const [rows, setRows] = useState<ContentToolTelemetryRow[]>([])
  const [error, setError] = useState<string | null>(null)
  const [from, setFrom] = useState(() => {
    const d = new Date()
    d.setDate(d.getDate() - 30)
    return d.toISOString().slice(0, 10)
  })
  const [to, setTo] = useState(() => new Date().toISOString().slice(0, 10))

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const raw = await fetchContentToolTelemetry({ from, to })
        if (!cancelled) setRows(raw.tools ?? [])
      } catch (e: unknown) {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load telemetry.')
      }
    })()
    return () => {
      cancelled = true
    }
  }, [from, to])

  return (
    <section className="space-y-3" data-testid="admin-tool-telemetry">
      <h2 className="text-lg font-semibold text-fg-default">
        {t('contentTools.analytics.adminTelemetryTitle')}
      </h2>
      <div className="flex flex-wrap gap-2 text-sm">
        <label>
          {t('contentTools.analytics.from')}
          <input
            type="date"
            className="ml-2 rounded border border-border-strong px-2 py-1 dark:border-border-default dark:bg-surface-base"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
        </label>
        <label>
          {t('contentTools.analytics.to')}
          <input
            type="date"
            className="ml-2 rounded border border-border-strong px-2 py-1 dark:border-border-default dark:bg-surface-base"
            value={to}
            onChange={(e) => setTo(e.target.value)}
          />
        </label>
      </div>
      {error ? (
        <p className="text-sm text-rose-600" role="alert">
          {error}
        </p>
      ) : null}
      <table className="w-full text-left text-sm">
        <thead>
          <tr className="border-b border-border-default">
            <th className="py-1 pr-2">{t('contentTools.analytics.colTool')}</th>
            <th className="py-1 pr-2">{t('contentTools.analytics.colInstances')}</th>
            <th className="py-1 pr-2">{t('contentTools.analytics.colLearners')}</th>
            <th className="py-1 pr-2">{t('contentTools.analytics.completed')}</th>
            <th className="py-1 pr-2">{t('contentTools.analytics.colErrors')}</th>
            <th className="py-1">{t('contentTools.analytics.colAiCost')}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.toolId} className="border-b border-border-subtle">
              <td className="py-1 pr-2 font-medium">{r.toolId}</td>
              <td className="py-1 pr-2 tabular-nums">{r.instances}</td>
              <td className="py-1 pr-2 tabular-nums">{r.learners}</td>
              <td className="py-1 pr-2 tabular-nums">{r.completions}</td>
              <td className="py-1 pr-2 tabular-nums">{r.renderErrors}</td>
              <td className="py-1 tabular-nums">${r.aiCostUsd.toFixed(2)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  )
}
