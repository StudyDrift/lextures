import { useCallback, useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  decideToolReview,
  listToolReviews,
  type ToolReview,
} from '../../lib/content-tool-marketplace-api'

export default function ToolReviewsPage() {
  const { t } = useTranslation('contentTools')
  const titleId = useId()
  const { ffContentToolMarketplace } = usePlatformFeatures()
  const [reviews, setReviews] = useState<ToolReview[]>([])
  const [error, setError] = useState<string | null>(null)
  const [notes, setNotes] = useState<Record<string, string>>({})

  const refresh = useCallback(async () => {
    const list = await listToolReviews('pending')
    setReviews(list)
  }, [])

  useEffect(() => {
    if (!ffContentToolMarketplace) return
    void refresh().catch((e: unknown) =>
      setError(e instanceof Error ? e.message : 'Failed to load'),
    )
  }, [ffContentToolMarketplace, refresh])

  if (!ffContentToolMarketplace) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <p className="text-sm">{t('contentTools.marketplace.disabled')}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-4xl p-6" data-testid="tool-reviews-page">
      <h1 id={titleId} className="text-2xl font-semibold">
        {t('contentTools.review.title')}
      </h1>
      <p className="mt-1 text-sm text-fg-muted">{t('contentTools.review.help')}</p>
      {error ? (
        <p className="mt-4 text-sm text-rose-700" role="alert">
          {error}
        </p>
      ) : null}
      <ul className="mt-6 space-y-6">
        {reviews.map((r) => (
          <li key={r.id} className="border-b pb-4" data-testid={`tool-review-${r.id}`}>
            <div className="font-medium">
              {r.displayName || r.toolId} <span className="text-sm text-fg-muted">v{r.version}</span>
            </div>
            <label className="mt-2 block text-sm">
              {t('contentTools.review.notes')}
              <textarea
                className="mt-1 w-full rounded border px-3 py-2"
                data-testid={`tool-review-notes-${r.id}`}
                value={notes[r.id] ?? ''}
                onChange={(e) => setNotes((prev) => ({ ...prev, [r.id]: e.target.value }))}
              />
            </label>
            <div className="mt-2 flex gap-2">
              <button
                type="button"
                className="rounded bg-emerald-700 px-3 py-1.5 text-sm text-white"
                data-testid={`tool-review-approve-${r.id}`}
                onClick={() => {
                  void decideToolReview(r.id, {
                    approve: true,
                    notes: notes[r.id] || 'approved',
                  })
                    .then(() => refresh())
                    .catch((e: unknown) =>
                      setError(e instanceof Error ? e.message : 'Decision failed'),
                    )
                }}
              >
                {t('contentTools.review.approve')}
              </button>
              <button
                type="button"
                className="rounded bg-rose-700 px-3 py-1.5 text-sm text-white"
                data-testid={`tool-review-reject-${r.id}`}
                onClick={() => {
                  void decideToolReview(r.id, {
                    approve: false,
                    notes: notes[r.id] || 'rejected',
                  })
                    .then(() => refresh())
                    .catch((e: unknown) =>
                      setError(e instanceof Error ? e.message : 'Decision failed'),
                    )
                }}
              >
                {t('contentTools.review.reject')}
              </button>
            </div>
          </li>
        ))}
      </ul>
      {reviews.length === 0 ? (
        <p className="mt-4 text-sm text-fg-muted">{t('contentTools.review.empty')}</p>
      ) : null}
    </div>
  )
}
