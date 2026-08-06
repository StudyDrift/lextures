import { useId } from 'react'
import { useTranslation } from 'react-i18next'
import { InlineQuestionsBuildAiButton } from './inline-questions-build-ai-button'
import {
  REVEAL_OPTIONS,
  asQuestions,
  defaultOptions,
  fieldClass,
  labelClass,
  newId,
  sectionClass,
  type Question,
} from './inline-questions-editor-helpers'
import { InlineQuestionsQuestionCard } from './inline-questions-question-card'

export type InlineQuestionsEditorProps = {
  value: Record<string, unknown>
  onChange: (next: Record<string, unknown>) => void
  disabled?: boolean
  idPrefix?: string
  courseCode?: string
  instanceId?: string
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
          className="text-xs font-semibold uppercase tracking-wide text-fg-muted"
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

          <label className="block space-y-1.5">
            <span className={labelClass}>
              {t('contentTools.tools.inline_questions.editor.questionsAtATime')}
            </span>
            <select
              disabled={disabled}
              value={
                value.questionsAtATime === 'all' ||
                value.questionsAtATime === undefined ||
                value.questionsAtATime === null
                  ? 'all'
                  : String(value.questionsAtATime)
              }
              onChange={(e) =>
                patch({
                  questionsAtATime: e.target.value === 'all' ? 'all' : Number(e.target.value),
                })
              }
              className={fieldClass}
            >
              <option value="all">
                {t('contentTools.tools.inline_questions.editor.questionsAtATimeAll')}
              </option>
              {[1, 2, 3].map((n) => (
                <option key={n} value={n}>
                  {n}
                </option>
              ))}
            </select>
          </label>
        </div>

        <div className="flex flex-col gap-3 border-t border-border-default pt-4 sm:flex-row sm:flex-wrap sm:gap-x-6 sm:gap-y-3 dark:border-border-default">
          <label className="flex min-h-10 cursor-pointer items-center gap-2.5 text-sm text-fg-default">
            <input
              type="checkbox"
              disabled={disabled}
              checked={value.sequential === true}
              onChange={(e) => patch({ sequential: e.target.checked })}
              className="size-4 rounded border-border-strong text-fg-default focus:ring-slate-400 dark:border-neutral-500"
            />
            <span>{t('contentTools.tools.inline_questions.editor.sequential')}</span>
          </label>
          <label className="flex min-h-10 cursor-pointer items-center gap-2.5 text-sm text-fg-default">
            <input
              type="checkbox"
              disabled={disabled}
              checked={value.shuffleOptions === true}
              onChange={(e) => patch({ shuffleOptions: e.target.checked })}
              className="size-4 rounded border-border-strong text-fg-default focus:ring-slate-400 dark:border-neutral-500"
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
        <h4 className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
          {t('contentTools.tools.inline_questions.editor.questions')}
        </h4>

        {questions.map((q, idx) => (
          <InlineQuestionsQuestionCard
            key={q.id}
            question={q}
            index={idx}
            baseId={baseId}
            disabled={disabled}
            canRemove={questions.length > 1}
            onChange={(next) => updateQuestion(idx, next)}
            onRemove={() => setQuestions(questions.filter((_, i) => i !== idx))}
          />
        ))}
      </div>

      <button
        type="button"
        disabled={disabled || questions.length >= 3}
        onClick={addQuestion}
        className="w-full rounded-md border border-dashed border-border-strong px-4 py-2.5 text-sm font-medium text-fg-muted hover:border-slate-400 hover:bg-surface-base disabled:opacity-40 dark:border-border-default dark:text-fg-default dark:hover:border-neutral-500 dark:hover:bg-surface-raised"
      >
        {t('contentTools.tools.inline_questions.editor.addQuestion')}
      </button>
    </div>
  )
}

export default InlineQuestionsEditor
