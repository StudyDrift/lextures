import { useTranslation } from 'react-i18next'
import {
  QUESTION_TYPES,
  defaultOptions,
  fieldClass,
  labelClass,
  newId,
  type Question,
} from './inline-questions-editor-helpers'

export type InlineQuestionsQuestionCardProps = {
  question: Question
  index: number
  baseId: string
  disabled?: boolean
  canRemove: boolean
  onChange: (next: Question) => void
  onRemove: () => void
}

export function InlineQuestionsQuestionCard({
  question: q,
  index: idx,
  baseId,
  disabled,
  canRemove,
  onChange,
  onRemove,
}: InlineQuestionsQuestionCardProps) {
  const { t } = useTranslation('contentTools')

  return (
    <div
      className="space-y-4 rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-950"
      data-editor-question={q.id}
    >
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {t('contentTools.tools.inline_questions.editor.questionN', { n: idx + 1 })}
        </p>
        <button
          type="button"
          disabled={disabled || !canRemove}
          onClick={onRemove}
          className="rounded-md px-2 py-1 text-xs font-medium text-rose-600 hover:bg-rose-50 disabled:opacity-40 dark:text-rose-400 dark:hover:bg-rose-950/40"
        >
          {t('contentTools.tools.inline_questions.editor.remove')}
        </button>
      </div>

      <div className="grid gap-4">
        <label className="block space-y-1.5">
          <span className={labelClass}>
            {t('contentTools.tools.inline_questions.editor.type')}
          </span>
          <select
            disabled={disabled}
            value={q.type}
            onChange={(e) => {
              const type = e.target.value
              const next: Question = { ...q, type }
              if (type === 'short_text') {
                next.options = undefined
                next.acceptedAnswers = next.acceptedAnswers ?? ['']
              } else if (type === 'numeric') {
                next.options = undefined
                next.correctValue = next.correctValue ?? 0
                next.tolerance = next.tolerance ?? { kind: 'absolute', value: 0 }
              } else {
                next.options = next.options?.length ? next.options : defaultOptions(type)
              }
              onChange(next)
            }}
            className={fieldClass}
          >
            {QUESTION_TYPES.map((type) => (
              <option key={type} value={type}>
                {t(`contentTools.tools.inline_questions.editor.types.${type}`)}
              </option>
            ))}
          </select>
        </label>

        <label className="block space-y-1.5">
          <span className={labelClass}>
            {t('contentTools.tools.inline_questions.editor.prompt')}
          </span>
          <textarea
            disabled={disabled}
            rows={3}
            value={q.prompt}
            onChange={(e) => onChange({ ...q, prompt: e.target.value })}
            className={`${fieldClass} resize-y`}
          />
        </label>
      </div>

      {(q.type === 'single' || q.type === 'multi' || q.type === 'true_false') && (
        <div className="space-y-3">
          <p className={labelClass}>
            {t('contentTools.tools.inline_questions.editor.options')}
          </p>
          <ul className="space-y-3">
            {(q.options ?? []).map((opt, oi) => (
              <li
                key={opt.id}
                className="space-y-2 rounded-md border border-slate-200 bg-slate-50/80 p-3 dark:border-neutral-700 dark:bg-neutral-900/50"
              >
                <div className="flex items-start gap-3">
                  <label className="mt-2.5 flex shrink-0 items-center gap-1.5">
                    <input
                      type={q.type === 'multi' ? 'checkbox' : 'radio'}
                      name={`${baseId}-correct-${q.id}`}
                      disabled={disabled}
                      checked={Boolean(opt.correct)}
                      onChange={() => {
                        const options = [...(q.options ?? [])]
                        if (q.type === 'multi') {
                          options[oi] = { ...opt, correct: !opt.correct }
                        } else {
                          for (let i = 0; i < options.length; i++) {
                            options[i] = { ...options[i], correct: i === oi }
                          }
                        }
                        onChange({ ...q, options })
                      }}
                      className="size-4 border-slate-300 text-slate-800 focus:ring-slate-400 dark:border-neutral-500"
                    />
                    <span className="sr-only">
                      {t('contentTools.tools.inline_questions.editor.correctOption')}
                    </span>
                  </label>
                  <div className="min-w-0 flex-1 space-y-2">
                    <input
                      type="text"
                      disabled={disabled}
                      placeholder={t('contentTools.tools.inline_questions.editor.optionText')}
                      value={opt.text}
                      onChange={(e) => {
                        const options = [...(q.options ?? [])]
                        options[oi] = { ...opt, text: e.target.value }
                        onChange({ ...q, options })
                      }}
                      className={fieldClass}
                      aria-label={t('contentTools.tools.inline_questions.editor.optionText')}
                    />
                    <input
                      type="text"
                      disabled={disabled}
                      placeholder={t('contentTools.tools.inline_questions.editor.feedback')}
                      value={opt.feedback ?? ''}
                      onChange={(e) => {
                        const options = [...(q.options ?? [])]
                        options[oi] = { ...opt, feedback: e.target.value }
                        onChange({ ...q, options })
                      }}
                      className={fieldClass}
                      aria-label={t('contentTools.tools.inline_questions.editor.feedback')}
                    />
                  </div>
                  {q.type !== 'true_false' ? (
                    <button
                      type="button"
                      disabled={disabled || (q.options?.length ?? 0) <= 2}
                      onClick={() =>
                        onChange({
                          ...q,
                          options: (q.options ?? []).filter((_, i) => i !== oi),
                        })
                      }
                      className="mt-2 shrink-0 rounded-md px-2 py-1 text-sm font-medium text-rose-600 hover:bg-rose-50 disabled:opacity-40 dark:text-rose-400 dark:hover:bg-rose-950/40"
                      aria-label={t('contentTools.tools.inline_questions.editor.remove')}
                    >
                      ×
                    </button>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
          {q.type !== 'true_false' ? (
            <button
              type="button"
              disabled={disabled}
              onClick={() =>
                onChange({
                  ...q,
                  options: [...(q.options ?? []), { id: newId('opt'), text: '', correct: false }],
                })
              }
              className="text-sm font-medium text-slate-700 underline decoration-slate-300 underline-offset-2 hover:text-slate-900 dark:text-neutral-300 dark:hover:text-neutral-100"
            >
              {t('contentTools.tools.inline_questions.editor.addOption')}
            </button>
          ) : null}
        </div>
      )}

      {q.type === 'short_text' && (
        <label className="block space-y-1.5">
          <span className={labelClass}>
            {t('contentTools.tools.inline_questions.editor.acceptedAnswers')}
          </span>
          <input
            type="text"
            disabled={disabled}
            value={(q.acceptedAnswers ?? []).join(', ')}
            onChange={(e) =>
              onChange({
                ...q,
                acceptedAnswers: e.target.value
                  .split(',')
                  .map((s) => s.trim())
                  .filter(Boolean),
              })
            }
            className={fieldClass}
          />
        </label>
      )}

      {q.type === 'numeric' && (
        <div className="grid gap-4 sm:grid-cols-3">
          <label className="block space-y-1.5">
            <span className={labelClass}>
              {t('contentTools.tools.inline_questions.editor.correctValue')}
            </span>
            <input
              type="number"
              disabled={disabled}
              value={q.correctValue ?? 0}
              onChange={(e) => onChange({ ...q, correctValue: Number(e.target.value) })}
              className={fieldClass}
            />
          </label>
          <label className="block space-y-1.5">
            <span className={labelClass}>
              {t('contentTools.tools.inline_questions.editor.tolerance')}
            </span>
            <input
              type="number"
              disabled={disabled}
              value={q.tolerance?.value ?? 0}
              onChange={(e) =>
                onChange({
                  ...q,
                  tolerance: {
                    kind: q.tolerance?.kind ?? 'absolute',
                    value: Number(e.target.value),
                  },
                })
              }
              className={fieldClass}
            />
          </label>
          <label className="block space-y-1.5">
            <span className={labelClass}>
              {t('contentTools.tools.inline_questions.editor.unit')}
            </span>
            <input
              type="text"
              disabled={disabled}
              value={q.unit ?? ''}
              onChange={(e) => onChange({ ...q, unit: e.target.value })}
              className={fieldClass}
            />
          </label>
        </div>
      )}

      <label className="block space-y-1.5">
        <span className={labelClass}>
          {t('contentTools.tools.inline_questions.editor.explanation')}
        </span>
        <textarea
          disabled={disabled}
          rows={2}
          value={q.explanation ?? ''}
          onChange={(e) => onChange({ ...q, explanation: e.target.value })}
          className={`${fieldClass} resize-y`}
        />
      </label>
    </div>
  )
}
