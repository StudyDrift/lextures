import { useState } from 'react'
import { useTranslation } from 'react-i18next'

export function AnswerOrdering({
  options,
  locked,
  onSubmit,
}: {
  options: Array<{ id: string; text: string }>
  locked: boolean
  onSubmit: (order: string[]) => void
}) {
  const { t } = useTranslation('common')
  const [order, setOrder] = useState(() => options.map((o) => o.id))

  function move(id: string, dir: -1 | 1) {
    setOrder((prev) => {
      const i = prev.indexOf(id)
      const j = i + dir
      if (i < 0 || j < 0 || j >= prev.length) return prev
      const next = [...prev]
      const tmp = next[i]!
      next[i] = next[j]!
      next[j] = tmp
      return next
    })
  }

  const byId = new Map(options.map((o) => [o.id, o.text]))

  return (
    <div className="space-y-3">
      <ol className="space-y-2">
        {order.map((id, i) => (
          <li
            key={id}
            className="flex min-h-12 items-center gap-2 rounded-xl border border-border-default bg-surface-raised px-3 py-2 dark:border-border-default dark:bg-surface-raised"
          >
            <span className="w-6 tabular-nums text-fg-muted">{i + 1}</span>
            <span className="flex-1">{byId.get(id) ?? id}</span>
            {!locked && (
              <div className="flex gap-1">
                <button
                  type="button"
                  className="min-h-11 min-w-11 rounded-md bg-surface-sunken"
                  onClick={() => move(id, -1)}
                  aria-label={t('liveQuiz.answer.moveUp')}
                >
                  ↑
                </button>
                <button
                  type="button"
                  className="min-h-11 min-w-11 rounded-md bg-surface-sunken"
                  onClick={() => move(id, 1)}
                  aria-label={t('liveQuiz.answer.moveDown')}
                >
                  ↓
                </button>
              </div>
            )}
          </li>
        ))}
      </ol>
      <button
        type="button"
        disabled={locked}
        onClick={() => onSubmit(order)}
        className="min-h-12 w-full rounded-xl bg-accent-solid px-4 py-3 text-base font-semibold text-white disabled:opacity-40"
      >
        {t('liveQuiz.answer.submit')}
      </button>
    </div>
  )
}
