import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'

export type AcceptedAnswerDraft = {
  text: string
  matchMode: 'exact' | 'case_insensitive' | 'trim' | 'fuzzy'
  fuzzyMax?: number
}

type Props = {
  accepted: AcceptedAnswerDraft[]
  onChange: (accepted: AcceptedAnswerDraft[]) => void
  disabled?: boolean
}

export function TypeAnswerEditor({ accepted, onChange, disabled }: Props) {
  const { t } = useTranslation('common')

  function update(index: number, patch: Partial<AcceptedAnswerDraft>) {
    onChange(accepted.map((a, i) => (i === index ? { ...a, ...patch } : a)))
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-slate-700 dark:text-neutral-200">
          {t('liveQuiz.editor.acceptedAnswers')}
        </span>
        <button
          type="button"
          disabled={disabled || accepted.length >= 20}
          onClick={() => onChange([...accepted, { text: '', matchMode: 'case_insensitive' }])}
          className="inline-flex min-h-11 items-center gap-1 rounded-md px-2 text-sm text-indigo-600 disabled:opacity-40 dark:text-indigo-400"
        >
          <Plus className="h-4 w-4" aria-hidden />
          {t('liveQuiz.editor.addAccepted')}
        </button>
      </div>
      <ul className="space-y-2">
        {accepted.map((a, index) => (
          <li key={index} className="flex flex-wrap items-center gap-2">
            <input
              value={a.text}
              disabled={disabled}
              onChange={(e) => update(index, { text: e.target.value })}
              placeholder={t('liveQuiz.editor.acceptedPlaceholder')}
              className="min-h-11 min-w-[12rem] flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
            />
            <select
              value={a.matchMode}
              disabled={disabled}
              onChange={(e) =>
                update(index, { matchMode: e.target.value as AcceptedAnswerDraft['matchMode'] })
              }
              className="min-h-11 rounded-md border border-slate-300 px-2 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              aria-label={t('liveQuiz.editor.matchMode')}
            >
              <option value="exact">{t('liveQuiz.editor.match.exact')}</option>
              <option value="case_insensitive">{t('liveQuiz.editor.match.caseInsensitive')}</option>
              <option value="trim">{t('liveQuiz.editor.match.trim')}</option>
              <option value="fuzzy">{t('liveQuiz.editor.match.fuzzy')}</option>
            </select>
            {a.matchMode === 'fuzzy' ? (
              <input
                type="number"
                min={0}
                max={10}
                value={a.fuzzyMax ?? 1}
                disabled={disabled}
                onChange={(e) => update(index, { fuzzyMax: Number(e.target.value) })}
                className="w-20 min-h-11 rounded-md border border-slate-300 px-2 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
                aria-label={t('liveQuiz.editor.fuzzyMax')}
              />
            ) : null}
            <button
              type="button"
              disabled={disabled || accepted.length <= 1}
              onClick={() => onChange(accepted.filter((_, i) => i !== index))}
              className="min-h-11 rounded-md px-2 text-slate-500 hover:text-red-600 disabled:opacity-40"
              aria-label={t('liveQuiz.editor.removeAccepted')}
            >
              <Trash2 className="h-4 w-4" aria-hidden />
            </button>
          </li>
        ))}
      </ul>
    </div>
  )
}
