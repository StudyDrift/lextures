import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolPageAnalytics,
  type ContentToolInstanceAnalytics,
} from '../../../lib/courses-api'

export type PageToolsOverviewProps = {
  courseCode: string
  itemId: string
  onOpenInstance?: (instanceId: string) => void
}

export function PageToolsOverview({ courseCode, itemId, onOpenInstance }: PageToolsOverviewProps) {
  const { t } = useTranslation('contentTools')
  const [instances, setInstances] = useState<ContentToolInstanceAnalytics[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    void (async () => {
      try {
        const res = await fetchContentToolPageAnalytics(courseCode, itemId)
        if (!cancelled) setInstances(res.instances)
      } catch {
        if (!cancelled) setInstances([])
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [courseCode, itemId])

  if (loading) {
    return <p className="text-sm text-fg-muted">{t('contentTools.runtime.loading')}</p>
  }
  if (instances.length === 0) return null

  return (
    <section className="space-y-3" data-testid="page-tools-overview">
      <h3 className="text-sm font-semibold text-fg-default">
        {t('contentTools.analytics.pageOverviewTitle')}
      </h3>
      <ul className="space-y-2">
        {instances.map((inst) => {
          const pct =
            inst.learners > 0 ? Math.round((inst.completed / inst.learners) * 100) : 0
          return (
            <li key={inst.instanceId}>
              <button
                type="button"
                className="flex w-full items-center gap-3 rounded border border-border-default px-3 py-2 text-left text-sm hover:bg-surface-base dark:border-border-default dark:hover:bg-surface-raised"
                onClick={() => onOpenInstance?.(inst.instanceId)}
              >
                <span className="min-w-0 flex-1 truncate font-medium text-fg-default">
                  {inst.title || inst.toolId}
                </span>
                <span className="w-24 shrink-0">
                  <span className="mb-0.5 block text-xs text-fg-muted">{pct}%</span>
                  <span className="block h-1.5 rounded bg-surface-sunken">
                    <span
                      className="block h-1.5 rounded bg-sky-600"
                      style={{ width: `${pct}%` }}
                    />
                  </span>
                </span>
              </button>
            </li>
          )
        })}
      </ul>
    </section>
  )
}
