import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import { InlineQuestionsBuildAiButton } from './inline-questions-build-ai-button'

export type InlineQuestionsEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
  courseCode?: string
  instanceId?: string
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

const QUESTION_TYPES = ['single', 'multi', 'true_false', 'short_text', 'numeric'] as const
const REVEAL_OPTIONS = ['first_attempt', 'last_attempt', 'never'] as const
const fieldClass =
  'w-full rounded-md border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 shadow-sm focus:border-slate-400 focus:outline-none focus:ring-2 focus:ring-slate-200 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:focus:border-neutral-500 dark:focus:ring-neutral-700'
const labelClass = 'block text-xs font-medium text-slate-700 dark:text-neutral-300'
const sectionClass =
  'space-y-4 rounded-lg border border-slate-200 bg-slate-50/50 p-4 dark:border-neutral-700 dark:bg-neutral-900/40'

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
  courseCode,
  instanceId,
}: InlineQuestionsEditorProps) {
  const { t } = useTranslation('contentTools')
  const baseId = useId()
  const questions = asQuestions(value)

  function patch(partial: Record<string, unknown>) {
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
    <div className="space-y-6" data-testid="inline-questions-editor">
      <section className={sectionClass} aria-labelledby={`${baseId}-settings`}>
        <h4
          id={`${baseId}-settings`}
          className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400"
        >
          {t('contentTools.tools.inline_questions.editor.settings')}
        </h4>

        <label className="block space-y-1.5">
          <span className={labelClass}>
            {t('contentTools.tools.inline_questions.editor.label')}
          </span>
          <input
            id={`${idPrefix}-${baseId}-label`}
            type="text"
            disabled={disabled}
            value={typeof value.label === 'string' ? value.label : ''}
            onChange={(e) => patch({ label: e.target.value })}
            className={fieldClass}
          />
        </label>

        <div className="grid gap-4 sm:grid-cols-2">
          <label className="block space-y-1.5">
            <span className={labelClass}>
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
              className={fieldClass}
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

          <label className="block space-y-1.5">
            <span className={labelClass}>
              {t('contentTools.tools.inline_questions.editor.reveal')}
            </span>
            <select
              disabled={disabled}
              value={
                typeof value.revealCorrectAfter === 'string'
                  ? value.revealCorrectAfter
                  : 'last_attempt'
              }
              onChange={(e) => patch({ revealCorrectAfter: e.target.value })}
              className={fieldClass}
            >
              {REVEAL_OPTIONS.map((opt) => (
                <option key={opt} value={opt}>
                  {t(`contentTools.tools.inline_questions.editor.revealOptions.${opt}`)}
                </option>
              ))}
            </select>
          </label>
        </div>

        <div className="flex flex-col gap-3 border-t border-slate-200 pt-4 sm:flex-row sm:flex-wrap sm:gap-x-6 sm:gap-y-3 dark:border-neutral-700">
          <label className="flex min-h-10 cursor-pointer items-center gap-2.5 text-sm text-slate-700 dark:text-neutral-200">
            <input
              type="checkbox"
              disabled={disabled}
              checked={value.sequential === true}
              onChange={(e) => patch({ sequential: e.target.checked })}
              className="size-4 rounded border-slate-300 text-slate-800 focus:ring-slate-400 dark:border-neutral-500"
            />
            <span>{t('contentTools.tools.inline_questions.editor.sequential')}</span>
          </label>
          <label className="flex min-h-10 cursor-pointer items-center gap-2.5 text-sm text-slate-700 dark:text-neutral-200">
            <input
              type="checkbox"
              disabled={disabled}
              checked={value.shuffleOptions === true}
              onChange={(e) => patch({ shuffleOptions: e.target.checked })}
              className="size-4 rounded border-slate-300 text-slate-800 focus:ring-slate-400 dark:border-neutral-500"
            />
            <span>{t('contentTools.tools.inline_questions.editor.shuffle')}</span>
          </label>
        </div>

        <InlineQuestionsBuildAiButton
          courseCode={courseCode}
          instanceId={instanceId}
          disabled={disabled}
          hasExistingQuestions={questions.some((q) => (q.prompt ?? '').trim().length > 0)}
          onBuilt={({ label, questions: built }) => {
            const next: Record<string, unknown> = { ...value, questions: built.slice(0, 3) }
            if (label) next.label = label
            onChange(next)
          }}
        />
      </section>

      <div className="space-y-4">
        <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
          {t('contentTools.tools.inline_questions.editor.questions')}
        </h4>

        {questions.map((q, idx) => (
          <div
            key={q.id}
            className="space-y-4 rounded-lg border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-950"
            data-editor-question={q.id}
          >
            <div className="flex items-center justify-between gap-3">
              <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
                {t('contentTools.tools.inline_questions.editor.questionN', { n: idx + 1 })}
              </p>
              <button
                type="button"
                disabled={disabled || questions.length <= 1}
                onClick={() => setQuestions(questions.filter((_, i) => i !== idx))}
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
                    updateQuestion(idx, next)
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
                  onChange={(e) => updateQuestion(idx, { ...q, prompt: e.target.value })}
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
                              updateQuestion(idx, { ...q, options })
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
                            placeholder={t(
                              'contentTools.tools.inline_questions.editor.optionText',
                            )}
                            value={opt.text}
                            onChange={(e) => {
                              const options = [...(q.options ?? [])]
                              options[oi] = { ...opt, text: e.target.value }
                              updateQuestion(idx, { ...q, options })
                            }}
                            className={fieldClass}
                            aria-label={t(
                              'contentTools.tools.inline_questions.editor.optionText',
                            )}
                          />
                          <input
                            type="text"
                            disabled={disabled}
                            placeholder={t(
                              'contentTools.tools.inline_questions.editor.feedback',
                            )}
                            value={opt.feedback ?? ''}
                            onChange={(e) => {
                              const options = [...(q.options ?? [])]
                              options[oi] = { ...opt, feedback: e.target.value }
                              updateQuestion(idx, { ...q, options })
                            }}
                            className={fieldClass}
                            aria-label={t(
                              'contentTools.tools.inline_questions.editor.feedback',
                            )}
                          />
                        </div>
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
                      updateQuestion(idx, {
                        ...q,
                        options: [
                          ...(q.options ?? []),
                          { id: newId('opt'), text: '', correct: false },
                        ],
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
                    updateQuestion(idx, {
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
                    onChange={(e) =>
                      updateQuestion(idx, { ...q, correctValue: Number(e.target.value) })
                    }
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
                      updateQuestion(idx, {
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
                    onChange={(e) => updateQuestion(idx, { ...q, unit: e.target.value })}
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
                onChange={(e) => updateQuestion(idx, { ...q, explanation: e.target.value })}
                className={`${fieldClass} resize-y`}
              />
            </label>
          </div>
        ))}
      </div>

      <button
        type="button"
        disabled={disabled || questions.length >= 3}
        onClick={addQuestion}
        className="w-full rounded-md border border-dashed border-slate-300 px-4 py-2.5 text-sm font-medium text-slate-700 hover:border-slate-400 hover:bg-slate-50 disabled:opacity-40 dark:border-neutral-600 dark:text-neutral-200 dark:hover:border-neutral-500 dark:hover:bg-neutral-900"
      >
        {t('contentTools.tools.inline_questions.editor.addQuestion')}
      </button>
    </div>
  )
}

export default InlineQuestionsEditor
