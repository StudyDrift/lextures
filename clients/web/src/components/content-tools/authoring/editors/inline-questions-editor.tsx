import { useId } from 'react'
import { useTranslation } from 'react-i18next'

export type InlineQuestionsEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
}

type Option = { id: string; text: string; correct?: boolean; feedback?: string }
type Question = {
  id: string
  type: string
  prompt: string
  options?: Option[]
  acceptedAnswers?: string[]
  correctValue?: number
  tolerance?: { kind: 'absolute' | 'relative'; value: number }
  unit?: string
  explanation?: string
  points?: number
  caseSensitive?: boolean
  normalizePunctuation?: boolean
}

const QUESTION_TYPES = [
  'single',
  'multi',
  'true_false',
  'short_text',
  'numeric',
] as const

function newId(prefix: string): string {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function asQuestions(value: Record<string, unknown>): Question[] {
  if (!Array.isArray(value.questions)) return []
  return value.questions as Question[]
}

function defaultOptions(type: string): Option[] {
  if (type === 'true_false') {
    return [
      { id: 'true', text: 'True', correct: true },
      { id: 'false', text: 'False', correct: false },
    ]
  }
  return [
    { id: newId('opt'), text: '', correct: true },
    { id: newId('opt'), text: '', correct: false },
  ]
}

export function InlineQuestionsEditor({
  value,
  onChange,
  disabled,
  idPrefix = 'iq-editor',
}: InlineQuestionsEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const questions = asQuestions(value)

  function patch( partial: Record<string, unknown>) {
    onChange({ ...value, ...partial })
  }

  function setQuestions(next: Question[]) {
    patch({ questions: next.slice(0, 3) })
  }

  function updateQuestion(idx: number, next: Question) {
    const copy = [...questions]
    copy[idx] = next
    setQuestions(copy)
  }

  function addQuestion() {
    if (questions.length >= 3) return
    setQuestions([
      ...questions,
      {
        id: newId('q'),
        type: 'single',
        prompt: '',
        options: defaultOptions('single'),
        points: 1,
      },
    ])
  }

  return (
    <div className="space-y-4" data-testid="inline-questions-editor">
      <label className="block space-y-1 text-xs">
        <span className="font-medium text-slate-700 dark:text-neutral-300">
          {t('contentTools.tools.inline_questions.editor.label')}
        </span>
        <input
          id={`${idPrefix}-${baseId}-label`}
          type="text"
          disabled={disabled}
          value={typeof value.label === 'string' ? value.label : ''}
          onChange={(e) => patch({ label: e.target.value })}
          className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
        />
      </label>

      <div className="grid grid-cols-2 gap-2 text-xs sm:grid-cols-4">
        <label className="space-y-1">
          <span className="font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.inline_questions.editor.attempts')}
          </span>
          <select
            disabled={disabled}
            value={value.attempts === 'unlimited' ? 'unlimited' : String(value.attempts ?? 2)}
            onChange={(e) =>
              patch({
                attempts: e.target.value === 'unlimited' ? 'unlimited' : Number(e.target.value),
              })
            }
            className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
          >
            {[1, 2, 3, 4, 5].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
            <option value="unlimited">
              {t('contentTools.tools.inline_questions.editor.unlimited')}
            </option>
          </select>
        </label>
        <label className="space-y-1">
          <span className="font-medium text-slate-700 dark:text-neutral-300">
            {t('contentTools.tools.inline_questions.editor.reveal')}
          </span>
          <select
            disabled={disabled}
            value={typeof value.revealCorrectAfter === 'string' ? value.revealCorrectAfter : 'last_attempt'}
            onChange={(e) => patch({ revealCorrectAfter: e.target.value })}
            className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
          >
            <option value="first_attempt">first_attempt</option>
            <option value="last_attempt">last_attempt</option>
            <option value="never">never</option>
          </select>
        </label>
        <label className="flex items-end gap-2 pb-1">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.sequential === true}
            onChange={(e) => patch({ sequential: e.target.checked })}
          />
          <span>{t('contentTools.tools.inline_questions.editor.sequential')}</span>
        </label>
        <label className="flex items-end gap-2 pb-1">
          <input
            type="checkbox"
            disabled={disabled}
            checked={value.shuffleOptions === true}
            onChange={(e) => patch({ shuffleOptions: e.target.checked })}
          />
          <span>{t('contentTools.tools.inline_questions.editor.shuffle')}</span>
        </label>
      </div>

      {questions.map((q, idx) => (
        <div
          key={q.id}
          className="space-y-2 border-t border-slate-200 pt-3 dark:border-neutral-700"
          data-editor-question={q.id}
        >
          <div className="flex items-center justify-between gap-2">
            <p className="text-xs font-semibold text-slate-800 dark:text-neutral-100">
              {t('contentTools.tools.inline_questions.editor.questionN', { n: idx + 1 })}
            </p>
            <button
              type="button"
              disabled={disabled || questions.length <= 1}
              onClick={() => setQuestions(questions.filter((_, i) => i !== idx))}
              className="text-xs text-rose-600 hover:underline disabled:opacity-40"
            >
              {t('contentTools.tools.inline_questions.editor.remove')}
            </button>
          </div>
          <div className="grid gap-2 sm:grid-cols-[10rem_1fr]">
            <label className="space-y-1 text-xs">
              <span className="font-medium">{t('contentTools.tools.inline_questions.editor.type')}</span>
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
                  updateQuestion(idx, next)
                }}
                className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
              >
                {QUESTION_TYPES.map((type) => (
                  <option key={type} value={type}>
                    {type}
                  </option>
                ))}
              </select>
            </label>
            <label className="space-y-1 text-xs">
              <span className="font-medium">{t('contentTools.tools.inline_questions.editor.prompt')}</span>
              <textarea
                disabled={disabled}
                rows={2}
                value={q.prompt}
                onChange={(e) => updateQuestion(idx, { ...q, prompt: e.target.value })}
                className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              />
            </label>
          </div>

          {(q.type === 'single' || q.type === 'multi' || q.type === 'true_false') && (
            <div className="space-y-2">
              {(q.options ?? []).map((opt, oi) => (
                <div key={opt.id} className="flex flex-wrap items-start gap-2 text-xs">
                  <label className="mt-2 flex items-center gap-1">
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
                        updateQuestion(idx, { ...q, options })
                      }}
                    />
                    <span className="sr-only">
                      {t('contentTools.tools.inline_questions.editor.correctOption')}
                    </span>
                  </label>
                  <input
                    type="text"
                    disabled={disabled}
                    placeholder={t('contentTools.tools.inline_questions.editor.optionText')}
                    value={opt.text}
                    onChange={(e) => {
                      const options = [...(q.options ?? [])]
                      options[oi] = { ...opt, text: e.target.value }
                      updateQuestion(idx, { ...q, options })
                    }}
                    className="min-w-[10rem] flex-1 rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                  />
                  <input
                    type="text"
                    disabled={disabled}
                    placeholder={t('contentTools.tools.inline_questions.editor.feedback')}
                    value={opt.feedback ?? ''}
                    onChange={(e) => {
                      const options = [...(q.options ?? [])]
                      options[oi] = { ...opt, feedback: e.target.value }
                      updateQuestion(idx, { ...q, options })
                    }}
                    className="min-w-[10rem] flex-1 rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                  />
                  {q.type !== 'true_false' ? (
                    <button
                      type="button"
                      disabled={disabled || (q.options?.length ?? 0) <= 2}
                      onClick={() =>
                        updateQuestion(idx, {
                          ...q,
                          options: (q.options ?? []).filter((_, i) => i !== oi),
                        })
                      }
                      className="text-rose-600 disabled:opacity-40"
                    >
                      ×
                    </button>
                  ) : null}
                </div>
              ))}
              {q.type !== 'true_false' ? (
                <button
                  type="button"
                  disabled={disabled}
                  onClick={() =>
                    updateQuestion(idx, {
                      ...q,
                      options: [...(q.options ?? []), { id: newId('opt'), text: '', correct: false }],
                    })
                  }
                  className="text-xs font-medium text-slate-700 underline dark:text-neutral-300"
                >
                  {t('contentTools.tools.inline_questions.editor.addOption')}
                </button>
              ) : null}
            </div>
          )}

          {q.type === 'short_text' && (
            <label className="block space-y-1 text-xs">
              <span className="font-medium">
                {t('contentTools.tools.inline_questions.editor.acceptedAnswers')}
              </span>
              <input
                type="text"
                disabled={disabled}
                value={(q.acceptedAnswers ?? []).join(', ')}
                onChange={(e) =>
                  updateQuestion(idx, {
                    ...q,
                    acceptedAnswers: e.target.value
                      .split(',')
                      .map((s) => s.trim())
                      .filter(Boolean),
                  })
                }
                className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
              />
            </label>
          )}

          {q.type === 'numeric' && (
            <div className="grid gap-2 sm:grid-cols-3 text-xs">
              <label className="space-y-1">
                <span className="font-medium">
                  {t('contentTools.tools.inline_questions.editor.correctValue')}
                </span>
                <input
                  type="number"
                  disabled={disabled}
                  value={q.correctValue ?? 0}
                  onChange={(e) =>
                    updateQuestion(idx, { ...q, correctValue: Number(e.target.value) })
                  }
                  className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              <label className="space-y-1">
                <span className="font-medium">
                  {t('contentTools.tools.inline_questions.editor.tolerance')}
                </span>
                <input
                  type="number"
                  disabled={disabled}
                  value={q.tolerance?.value ?? 0}
                  onChange={(e) =>
                    updateQuestion(idx, {
                      ...q,
                      tolerance: {
                        kind: q.tolerance?.kind ?? 'absolute',
                        value: Number(e.target.value),
                      },
                    })
                  }
                  className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
              <label className="space-y-1">
                <span className="font-medium">{t('contentTools.tools.inline_questions.editor.unit')}</span>
                <input
                  type="text"
                  disabled={disabled}
                  value={q.unit ?? ''}
                  onChange={(e) => updateQuestion(idx, { ...q, unit: e.target.value })}
                  className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
                />
              </label>
            </div>
          )}

          <label className="block space-y-1 text-xs">
            <span className="font-medium">
              {t('contentTools.tools.inline_questions.editor.explanation')}
            </span>
            <textarea
              disabled={disabled}
              rows={2}
              value={q.explanation ?? ''}
              onChange={(e) => updateQuestion(idx, { ...q, explanation: e.target.value })}
              className="w-full rounded-md border border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-950"
            />
          </label>
        </div>
      ))}

      <button
        type="button"
        disabled={disabled || questions.length >= 3}
        onClick={addQuestion}
        className="rounded-md border border-slate-300 px-3 py-1.5 text-xs font-medium text-slate-800 hover:bg-slate-50 disabled:opacity-40 dark:border-neutral-600 dark:text-neutral-100 dark:hover:bg-neutral-800"
      >
        {t('contentTools.tools.inline_questions.editor.addQuestion')}
      </button>
    </div>
  )
}

export default InlineQuestionsEditor
