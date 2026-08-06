import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  fetchContentToolModeration,
  moderateContentToolContent,
  type ContentToolModerationAction,
} from '../../../lib/content-tools-governance-api'

type Props = {
  courseCode: string
  instanceId: string
}

export function ModerationQueue({ courseCode, instanceId }: Props) {
  const { t } = useTranslation('contentTools')
  const [items, setItems] = useState<ContentToolModerationAction[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    const rows = await fetchContentToolModeration(courseCode, instanceId)
    setItems(rows)
  }, [courseCode, instanceId])

  useEffect(() => {
    let cancelled = false
    void refresh().catch((e: unknown) => {
      if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load')
    })
    return () => {
      cancelled = true
    }
  }, [refresh])

  async function act(id: string, action: string, contentPath?: string) {
    setBusy(id + action)
    setError(null)
    try {
      await moderateContentToolContent(courseCode, instanceId, {
        action,
        contentPath,
      })
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed')
    } finally {
      setBusy(null)
    }
  }

  return (
    <section className="mt-4" aria-labelledby="ct-moderation-title" data-testid="content-tool-moderation-queue">
      <h3 id="ct-moderation-title" className="text-sm font-semibold">
        {t('contentTools.safety.moderationTitle')}
      </h3>
      <p className="mt-1 text-xs text-fg-muted dark:text-fg-subtle">
        {t('contentTools.safety.moderationHelp')}
      </p>
      {error ? (
        <p className="mt-2 text-sm text-rose-700" role="alert">
          {error}
        </p>
      ) : null}
      {items.length === 0 ? (
        <p className="mt-2 text-sm text-fg-muted">{t('contentTools.safety.moderationEmpty')}</p>
      ) : (
        <ul className="mt-2 space-y-2" role="list">
          {items.map((item) => (
            <li
              key={item.id}
              className="rounded border border-border-default p-2 text-sm dark:border-border-default"
              data-moderation-id={item.id}
            >
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span>
                  <strong>{item.action}</strong>
                  {item.category ? ` · ${item.category}` : ''}
                  <span className="ms-2 text-xs text-fg-muted">{item.createdAt}</span>
                </span>
                {item.action === 'reported' ? (
                  <span className="flex gap-1">
                    <button
                      type="button"
                      className="rounded border px-2 py-0.5 text-xs"
                      disabled={busy !== null}
                      onClick={() => void act(item.id, 'hidden', item.contentPath)}
                    >
                      {t('contentTools.safety.hide')}
                    </button>
                    <button
                      type="button"
                      className="rounded border px-2 py-0.5 text-xs"
                      disabled={busy !== null}
                      onClick={() => void act(item.id, 'removed', item.contentPath)}
                    >
                      {t('contentTools.safety.remove')}
                    </button>
                  </span>
                ) : null}
                {item.action === 'hidden' || item.action === 'removed' ? (
                  <button
                    type="button"
                    className="rounded border px-2 py-0.5 text-xs"
                    disabled={busy !== null}
                    onClick={() => void act(item.id, 'restored', item.contentPath)}
                  >
                    {t('contentTools.safety.restore')}
                  </button>
                ) : null}
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
