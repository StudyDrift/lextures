import { useId } from 'react'
import { useTranslation } from 'react-i18next'

export type SortSequenceEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
}

type Item = { id: string; text: string; imageUrl?: string; imageAlt?: string }
type Bucket = { id: string; label: string; description?: string }

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asItems(value: Record<string, unknown>): Item[] {
  return Array.isArray(value.items) ? (value.items as Item[]) : []
}

function asBuckets(value: Record<string, unknown>): Bucket[] {
  return Array.isArray(value.buckets) ? (value.buckets as Bucket[]) : []
}

function asCorrectBucket(value: Record<string, unknown>): Record<string, string | string[]> {
  if (value.correctBucketByItem && typeof value.correctBucketByItem === 'object') {
    return value.correctBucketByItem as Record<string, string | string[]>
  }
  return {}
}

function asCorrectOrder(value: Record<string, unknown>): string[] {
  return Array.isArray(value.correctOrder) ? (value.correctOrder as string[]) : []
}

function asTieGroups(value: Record<string, unknown>): string[][] {
  return Array.isArray(value.tieGroups) ? (value.tieGroups as string[][]) : []
}

function asFeedback(value: Record<string, unknown>): Record<string, string> {
  if (value.itemFeedback && typeof value.itemFeedback === 'object') {
    return value.itemFeedback as Record<string, string>
  }
  return {}
}

export function SortSequenceEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'ss-editor',
}: SortSequenceEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const mode = value.mode === 'order' ? 'order' : 'categorize'
  const items = asItems(value)
  const buckets = asBuckets(value)
  const correctBucket = asCorrectBucket(value)
  const correctOrder = asCorrectOrder(value)
  const tieGroups = asTieGroups(value)
  const feedback = asFeedback(value)

  function patch(partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setItems(next: Item[]) {
    const capped = next.slice(0, 30)
    const ids = new Set(capped.map((i) => i.id))
    const order = (correctOrder.length ? correctOrder : capped.map((i) => i.id)).filter((id) =>
      ids.has(id),
    )
    for (const it of capped) {
      if (!order.includes(it.id)) order.push(it.id)
    }
    const nextBuckets = asCorrectBucket(value)
    const cleaned: Record<string, string | string[]> = {}
    for (const id of ids) {
      if (nextBuckets[id] != null) cleaned[id] = nextBuckets[id]!
    }
    const nextFeedback: Record<string, string> = {}
    for (const id of ids) {
      if (feedback[id]) nextFeedback[id] = feedback[id]!
    }
    patch({
      items: capped,
      correctOrder: order,
      correctBucketByItem: cleaned,
      itemFeedback: nextFeedback,
    })
  }

  function pasteItems(text: string) {
    const lines = text
      .split(/\r?\n/)
      .map((l) => l.trim())
      .filter(Boolean)
    if (lines.length === 0) return
    const next = lines.slice(0, 30).map((line) => ({ id: newId('item'), text: line }))
    setItems(next)
  }

  return (
    <div className="space-y-4" data-testid="sort-sequence-editor">
      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.sort_sequence.editor.prompt')}
        </span>
        <textarea
          id={`${idPrefix}-${baseId}-prompt`}
          className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          rows={2}
          disabled={disabled}
          value={typeof value.prompt === 'string' ? value.prompt : ''}
          onChange={(e) => patch({ prompt: e.target.value })}
        />
      </label>

      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.sort_sequence.editor.mode')}
        </span>
        <select
          className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
          disabled={disabled}
          value={mode}
          onChange={(e) => {
            const next = e.target.value === 'order' ? 'order' : 'categorize'
            patch({
              mode: next,
              buckets:
                next === 'categorize' && buckets.length < 2
                  ? [
                      { id: newId('bucket'), label: 'Category A' },
                      { id: newId('bucket'), label: 'Category B' },
                    ]
                  : buckets,
            })
          }}
        >
          <option value="categorize">
            {t('contentTools.tools.sort_sequence.editor.modeCategorize')}
          </option>
          <option value="order">{t('contentTools.tools.sort_sequence.editor.modeOrder')}</option>
        </select>
      </label>

      <div className="space-y-2">
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.sort_sequence.editor.items')}
          </span>
          <button
            type="button"
            className="text-xs text-sky-700 underline dark:text-sky-300"
            disabled={disabled || items.length >= 30}
            onClick={() => setItems([...items, { id: newId('item'), text: 'New item' }])}
          >
            {t('contentTools.tools.sort_sequence.editor.addItem')}
          </button>
        </div>
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.sort_sequence.editor.pasteList')}
          </span>
          <textarea
            className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            rows={3}
            disabled={disabled}
            placeholder={t('contentTools.tools.sort_sequence.editor.pastePlaceholder')}
            onBlur={(e) => {
              if (e.target.value.trim()) {
                pasteItems(e.target.value)
                e.target.value = ''
              }
            }}
          />
        </label>
        <ul className="space-y-2">
          {items.map((item, idx) => (
            <li
              key={item.id}
              className="rounded border border-slate-200 p-2 dark:border-neutral-700"
            >
              <div className="flex gap-2">
                <input
                  className="min-w-0 flex-1 rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                  disabled={disabled}
                  value={item.text}
                  placeholder={t('contentTools.tools.sort_sequence.editor.itemText')}
                  onChange={(e) => {
                    const next = [...items]
                    next[idx] = { ...item, text: e.target.value }
                    setItems(next)
                  }}
                />
                <button
                  type="button"
                  className="text-xs text-rose-700"
                  disabled={disabled || items.length <= 2}
                  onClick={() => setItems(items.filter((i) => i.id !== item.id))}
                >
                  {t('contentTools.tools.sort_sequence.editor.remove')}
                </button>
              </div>
              {mode === 'categorize' ? (
                <label className="mt-1 block text-xs">
                  <span className="text-slate-600 dark:text-neutral-400">
                    {t('contentTools.tools.sort_sequence.editor.correctBucket')}
                  </span>
                  <select
                    className="mt-0.5 w-full rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                    disabled={disabled}
                    value={
                      typeof correctBucket[item.id] === 'string'
                        ? (correctBucket[item.id] as string)
                        : Array.isArray(correctBucket[item.id])
                          ? (correctBucket[item.id] as string[])[0] ?? ''
                          : ''
                    }
                    onChange={(e) => {
                      patch({
                        correctBucketByItem: {
                          ...correctBucket,
                          [item.id]: e.target.value,
                        },
                      })
                    }}
                  >
                    <option value="">—</option>
                    {buckets.map((b) => (
                      <option key={b.id} value={b.id}>
                        {b.label}
                      </option>
                    ))}
                  </select>
                </label>
              ) : null}
              <label className="mt-1 block text-xs">
                <span className="text-slate-600 dark:text-neutral-400">
                  {t('contentTools.tools.sort_sequence.editor.feedback')}
                </span>
                <input
                  className="mt-0.5 w-full rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                  disabled={disabled}
                  value={feedback[item.id] ?? ''}
                  onChange={(e) =>
                    patch({
                      itemFeedback: { ...feedback, [item.id]: e.target.value },
                    })
                  }
                />
              </label>
            </li>
          ))}
        </ul>
      </div>

      {mode === 'categorize' ? (
        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-slate-700 dark:text-neutral-300">
              {t('contentTools.tools.sort_sequence.editor.buckets')}
            </span>
            <button
              type="button"
              className="text-xs text-sky-700 underline dark:text-sky-300"
              disabled={disabled || buckets.length >= 6}
              onClick={() =>
                patch({
                  buckets: [...buckets, { id: newId('bucket'), label: 'New bucket' }].slice(0, 6),
                })
              }
            >
              {t('contentTools.tools.sort_sequence.editor.addBucket')}
            </button>
          </div>
          {buckets.map((b, idx) => (
            <div key={b.id} className="flex gap-2">
              <input
                className="min-w-0 flex-1 rounded border border-slate-300 bg-white px-2 py-1 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                disabled={disabled}
                value={b.label}
                onChange={(e) => {
                  const next = [...buckets]
                  next[idx] = { ...b, label: e.target.value }
                  patch({ buckets: next })
                }}
              />
              <button
                type="button"
                className="text-xs text-rose-700"
                disabled={disabled || buckets.length <= 2}
                onClick={() => patch({ buckets: buckets.filter((x) => x.id !== b.id) })}
              >
                {t('contentTools.tools.sort_sequence.editor.remove')}
              </button>
            </div>
          ))}
        </div>
      ) : (
        <div className="space-y-2">
          <span className="text-xs font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.sort_sequence.editor.correctOrder')}
          </span>
          <ol className="list-decimal space-y-1 ps-5 text-sm">
            {(correctOrder.length ? correctOrder : items.map((i) => i.id)).map((id) => {
              const item = items.find((i) => i.id === id)
              return <li key={id}>{item?.text ?? id}</li>
            })}
          </ol>
          <p className="text-xs text-slate-500">
            {t('contentTools.tools.sort_sequence.editor.correctOrderHelp')}
          </p>
          <label className="block space-y-1 text-xs">
            <span className="font-medium text-slate-700 dark:text-neutral-300">
              {t('contentTools.tools.sort_sequence.editor.tieGroups')}
            </span>
            <textarea
              className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              rows={2}
              disabled={disabled}
              placeholder={t('contentTools.tools.sort_sequence.editor.tieGroupsPlaceholder')}
              value={tieGroups.map((g) => g.join(',')).join('\n')}
              onChange={(e) => {
                const groups = e.target.value
                  .split(/\r?\n/)
                  .map((line) =>
                    line
                      .split(',')
                      .map((s) => s.trim())
                      .filter(Boolean),
                  )
                  .filter((g) => g.length >= 2)
                patch({ tieGroups: groups })
              }}
            />
          </label>
        </div>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.sort_sequence.editor.attempts')}
          </span>
          <select
            className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={value.attempts === 'unlimited' ? 'unlimited' : String(value.attempts ?? 3)}
            onChange={(e) =>
              patch({
                attempts:
                  e.target.value === 'unlimited' ? 'unlimited' : Number(e.target.value),
              })
            }
          >
            {[1, 2, 3, 4, 5].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
            <option value="unlimited">
              {t('contentTools.tools.sort_sequence.editor.unlimited')}
            </option>
          </select>
        </label>
        <label className="block space-y-1 text-xs">
          <span className="font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.sort_sequence.editor.scoreMode')}
          </span>
          <select
            className="w-full rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
            disabled={disabled}
            value={value.scoreMode === 'all_or_nothing' ? 'all_or_nothing' : 'per_item'}
            onChange={(e) => patch({ scoreMode: e.target.value })}
          >
            <option value="per_item">
              {t('contentTools.tools.sort_sequence.editor.scorePerItem')}
            </option>
            <option value="all_or_nothing">
              {t('contentTools.tools.sort_sequence.editor.scoreAllOrNothing')}
            </option>
          </select>
        </label>
      </div>

      <div className="flex flex-wrap gap-4 text-xs">
        <label className="inline-flex items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.showPerItemCorrectness !== false}
            onChange={(e) => patch({ showPerItemCorrectness: e.target.checked })}
          />
          {t('contentTools.tools.sort_sequence.editor.showPerItem')}
        </label>
        <label className="inline-flex items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.lockCorrect !== false}
            onChange={(e) => patch({ lockCorrect: e.target.checked })}
          />
          {t('contentTools.tools.sort_sequence.editor.lockCorrect')}
        </label>
        <label className="inline-flex items-center gap-2">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.shuffleItems !== false}
            onChange={(e) => patch({ shuffleItems: e.target.checked })}
          />
          {t('contentTools.tools.sort_sequence.editor.shuffle')}
        </label>
      </div>
    </div>
  )
}
