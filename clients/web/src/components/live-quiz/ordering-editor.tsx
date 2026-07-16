import { useTranslation } from 'react-i18next'
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react'
import type { LiveQuizOption } from '../../lib/live-quiz-api'

type Props = {
  options: LiveQuizOption[]
  onChange: (options: LiveQuizOption[]) => void
  disabled?: boolean
}

export function OrderingEditor({ options, onChange, disabled }: Props) {
  const { t } = useTranslation('common')

  function move(index: number, dir: -1 | 1) {
    const next = index + dir
    if (next < 0 || next >= options.length) return
    const copy = [...options]
    const [item] = copy.splice(index, 1)
    copy.splice(next, 0, item)
    onChange(copy)
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-slate-700 dark:text-neutral-200">
          {t('liveQuiz.editor.orderingItems')}
        </span>
        <button
          type="button"
          disabled={disabled || options.length >= 8}
          onClick={() =>
            onChange([
              ...options,
              { id: `ord-${options.length + 1}-${Date.now().toString(36)}`, text: '', isCorrect: false },
            ])
          }
          className="inline-flex min-h-11 items-center gap-1 rounded-md px-2 text-sm text-indigo-600 disabled:opacity-40 dark:text-indigo-400"
        >
          <Plus className="h-4 w-4" aria-hidden />
          {t('liveQuiz.editor.addItem')}
        </button>
      </div>
      <p className="text-xs text-slate-500 dark:text-neutral-400">{t('liveQuiz.editor.orderingHint')}</p>
      <ol className="space-y-2">
        {options.map((opt, index) => (
          <li key={opt.id} className="flex items-center gap-2">
            <span className="w-6 text-sm text-slate-500">{index + 1}.</span>
            <input
              value={opt.text}
              disabled={disabled}
              onChange={(e) =>
                onChange(options.map((o, i) => (i === index ? { ...o, text: e.target.value } : o)))
              }
              className="min-h-11 min-w-0 flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
            />
            <button
              type="button"
              disabled={disabled || index === 0}
              onClick={() => move(index, -1)}
              className="min-h-11 rounded-md px-2 disabled:opacity-40"
              aria-label={t('liveQuiz.editor.moveUp')}
            >
              <ArrowUp className="h-4 w-4" aria-hidden />
            </button>
            <button
              type="button"
              disabled={disabled || index === options.length - 1}
              onClick={() => move(index, 1)}
              className="min-h-11 rounded-md px-2 disabled:opacity-40"
              aria-label={t('liveQuiz.editor.moveDown')}
            >
              <ArrowDown className="h-4 w-4" aria-hidden />
            </button>
            <button
              type="button"
              disabled={disabled || options.length <= 2}
              onClick={() => onChange(options.filter((_, i) => i !== index))}
              className="min-h-11 rounded-md px-2 text-slate-500 hover:text-red-600 disabled:opacity-40"
              aria-label={t('liveQuiz.editor.removeItem')}
            >
              <Trash2 className="h-4 w-4" aria-hidden />
            </button>
          </li>
        ))}
      </ol>
    </div>
  )
}
