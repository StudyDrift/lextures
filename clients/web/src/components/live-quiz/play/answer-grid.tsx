import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { colorForIndex, shapeForIndex } from './answer-shape-meta'
import { ShapeIcon } from './answer-shapes'

export function AnswerGrid({
  options,
  locked,
  multi = false,
  selectedIds,
  onSelect,
  onSubmitMulti,
}: {
  options: Array<{ id: string; text: string }>
  locked: boolean
  multi?: boolean
  selectedIds: string[]
  onSelect: (id: string) => void
  onSubmitMulti?: () => void
}) {
  const { t } = useTranslation('common')

  useEffect(() => {
    if (locked) return
    function onKey(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
      const n = Number(e.key)
      if (n >= 1 && n <= options.length) {
        e.preventDefault()
        const opt = options[n - 1]
        if (opt) onSelect(opt.id)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [locked, onSelect, options])

  return (
    <div className="space-y-3">
      <ul className="grid gap-3 sm:grid-cols-2" role="list">
        {options.map((opt, i) => {
          const selected = selectedIds.includes(opt.id)
          const shape = shapeForIndex(i)
          return (
            <li key={opt.id}>
              <button
                type="button"
                disabled={locked && !multi}
                onClick={() => onSelect(opt.id)}
                className={`flex min-h-14 w-full items-center gap-3 rounded-xl px-4 py-4 text-start text-lg font-medium text-white shadow-sm transition focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 disabled:opacity-70 ${colorForIndex(i)} ${
                  selected ? 'ring-4 ring-white/80' : ''
                } ${locked ? 'cursor-default' : 'active:scale-[0.98]'}`}
                aria-pressed={selected}
                aria-label={`${t('liveQuiz.answer.optionN', { n: i + 1 })}: ${opt.text}`}
              >
                <ShapeIcon shape={shape} className="h-7 w-7 shrink-0 opacity-95" />
                <span className="min-w-0 flex-1 break-words">{opt.text}</span>
                <span className="shrink-0 text-sm opacity-80" aria-hidden="true">
                  {i + 1}
                </span>
              </button>
            </li>
          )
        })}
      </ul>
      {multi && !locked && (
        <button
          type="button"
          onClick={onSubmitMulti}
          disabled={selectedIds.length === 0}
          className="min-h-12 w-full rounded-xl bg-indigo-600 px-4 py-3 text-base font-semibold text-white disabled:opacity-40"
        >
          {t('liveQuiz.answer.submit')}
        </button>
      )}
      {!locked && (
        <p className="text-center text-xs text-slate-500 dark:text-neutral-400">
          {t('liveQuiz.answer.shortcutHint')}
        </p>
      )}
    </div>
  )
}
