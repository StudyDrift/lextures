import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import type { LiveQuizOption } from '../../lib/live-quiz-api'

type Props = {
  options: LiveQuizOption[]
  onChange: (options: LiveQuizOption[]) => void
  multiCorrect?: boolean
  allowCorrect?: boolean
  disabled?: boolean
  min?: number
  max?: number
}

function newOptionId(options: LiveQuizOption[]): string {
  return `opt-${options.length + 1}-${Date.now().toString(36)}`
}

export function McOptionList({
  options,
  onChange,
  multiCorrect = false,
  allowCorrect = true,
  disabled,
  min = 2,
  max = 6,
}: Props) {
  const { t } = useTranslation('common')

  function setCorrect(index: number) {
    if (!allowCorrect) return
    onChange(
      options.map((o, i) => ({
        ...o,
        isCorrect: multiCorrect ? (i === index ? !o.isCorrect : o.isCorrect) : i === index,
      })),
    )
  }

  function updateText(index: number, text: string) {
    onChange(options.map((o, i) => (i === index ? { ...o, text } : o)))
  }

  function updateAlt(index: number, mediaAlt: string) {
    onChange(options.map((o, i) => (i === index ? { ...o, mediaAlt } : o)))
  }

  function remove(index: number) {
    if (options.length <= min) return
    onChange(options.filter((_, i) => i !== index))
  }

  function add() {
    if (options.length >= max) return
    onChange([...options, { id: newOptionId(options), text: '', isCorrect: false }])
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-slate-700 dark:text-neutral-200">
          {t('liveQuiz.editor.options')}
        </span>
        <button
          type="button"
          disabled={disabled || options.length >= max}
          onClick={add}
          className="inline-flex min-h-11 items-center gap-1 rounded-md px-2 text-sm text-indigo-600 disabled:opacity-40 dark:text-indigo-400"
        >
          <Plus className="h-4 w-4" aria-hidden />
          {t('liveQuiz.editor.addOption')}
        </button>
      </div>
      <ul className="space-y-2">
        {options.map((opt, index) => (
          <li
            key={opt.id}
            className="flex flex-wrap items-start gap-2 rounded-md border border-slate-200 p-2 dark:border-neutral-700"
          >
            {allowCorrect ? (
              <label className="mt-2 flex min-h-11 items-center gap-2 text-sm">
                <input
                  type={multiCorrect ? 'checkbox' : 'radio'}
                  name="correct-option"
                  checked={opt.isCorrect}
                  disabled={disabled}
                  onChange={() => setCorrect(index)}
                  aria-label={t('liveQuiz.editor.markCorrect', { index: index + 1 })}
                />
              </label>
            ) : null}
            <div className="min-w-[12rem] flex-1 space-y-1">
              <input
                value={opt.text}
                disabled={disabled}
                onChange={(e) => updateText(index, e.target.value)}
                placeholder={t('liveQuiz.editor.optionPlaceholder', { index: index + 1 })}
                className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
              {opt.mediaRef ? (
                <input
                  value={opt.mediaAlt ?? ''}
                  disabled={disabled}
                  onChange={(e) => updateAlt(index, e.target.value)}
                  placeholder={t('liveQuiz.editor.mediaAlt')}
                  className="w-full min-h-11 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                />
              ) : null}
            </div>
            <button
              type="button"
              disabled={disabled || options.length <= min}
              onClick={() => remove(index)}
              className="min-h-11 rounded-md px-2 text-slate-500 hover:text-red-600 disabled:opacity-40"
              aria-label={t('liveQuiz.editor.removeOption', { index: index + 1 })}
            >
              <Trash2 className="h-4 w-4" aria-hidden />
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}
