import { useEffect, useId, useRef, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'

type Feedback = {
  covered: string[]
  missing: string[]
  strength: string
  suggestion: string
  probe?: string
  mode: 'ai' | 'review'
}

type Attempt = {
  at: string
  text: string
  feedback?: Feedback
}

type InstructorNote = {
  text: string
  at: string
  by: string
}

type SubmitResult = {
  feedback?: Feedback
  keyPointLabels?: Record<string, string>
  attemptsRemaining?: number
  wordCount?: number
  mode?: string
  error?: string
  message?: string
  preserveInput?: boolean
  crisis?: boolean
}

function asAttempts(state: Record<string, unknown>): Attempt[] {
  const raw = state.attempts
  if (!Array.isArray(raw)) return []
  return raw.filter((a): a is Attempt => {
    if (!a || typeof a !== 'object') return false
    return typeof (a as Attempt).text === 'string'
  })
}

function countWords(text: string): number {
  const parts = text.trim().match(/[\p{L}\p{N}]+/gu)
  return parts?.length ?? 0
}

export default function ExplainItBackRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const promptId = useId()
  const feedbackId = useId()
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  const announcedFeedbackAt = useRef<string | null>(null)

  const prompt = typeof config.prompt === 'string' ? config.prompt : ''
  const minWords =
    typeof config.minWords === 'number' && config.minWords > 0 ? config.minWords : 25
  const maxWords =
    typeof config.maxWords === 'number' && config.maxWords > 0 ? config.maxWords : 150
  const maxAttempts =
    typeof config.attempts === 'number' && config.attempts > 0 ? config.attempts : 3
  const aiFeedback = config.aiFeedback !== false
  const revealKeyPoints = config.revealKeyPointsAfterSubmit !== false

  const attempts = asAttempts(state)
  const remoteDraft = typeof state.draft === 'string' ? state.draft : ''
  const instructorNote =
    state.instructorNote && typeof state.instructorNote === 'object'
      ? (state.instructorNote as InstructorNote)
      : null

  const [draft, setDraft] = useState(remoteDraft)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [crisis, setCrisis] = useState(false)
  const [labels, setLabels] = useState<Record<string, string>>({})
  const [revising, setRevising] = useState(false)

  const wordCount = countWords(draft)
  const latest = attempts.length > 0 ? attempts[attempts.length - 1] : null
  const attemptsLeft = Math.max(0, maxAttempts - attempts.length)
  const showFeedback = Boolean(latest?.feedback) && !revising
  const canSubmit =
    !readOnly && !busy && wordCount >= minWords && wordCount <= maxWords && attemptsLeft > 0

  useEffect(() => {
    setDraft(remoteDraft)
  }, [remoteDraft])

  useEffect(() => {
    if (!latest?.feedback || !latest.at) return
    if (announcedFeedbackAt.current === latest.at) return
    announcedFeedbackAt.current = latest.at
    announce(t('contentTools.tools.explain_it_back.feedbackReceived'))
  }, [latest?.at, latest?.feedback, announce, t])

  async function onDraftChange(value: string) {
    setDraft(value)
    if (!readOnly) {
      void save({ draft: value })
    }
  }

  async function onSubmit() {
    if (!canSubmit) return
    setBusy(true)
    setError(null)
    setCrisis(false)
    try {
      const raw = await runAction('submit', { text: draft.trim() })
      const result =
        raw && typeof raw === 'object' ? (raw as SubmitResult) : ({} as SubmitResult)
      if (result.error) {
        setError(result.message || result.error)
        setCrisis(Boolean(result.crisis))
        return
      }
      if (result.keyPointLabels) {
        setLabels(result.keyPointLabels)
      }
      setDraft('')
      setRevising(false)
      void save({ draft: '' })
      // Keep focus in the textarea (AC-9) — do not steal focus to the feedback card.
      textareaRef.current?.focus()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.runtime.retry'))
    } finally {
      setBusy(false)
    }
  }

  function startRevise() {
    if (attemptsLeft <= 0 || readOnly) return
    setRevising(true)
    setDraft(latest?.text ?? '')
    textareaRef.current?.focus()
  }

  const submitLabel = aiFeedback
    ? t('contentTools.tools.explain_it_back.submitFeedback')
    : t('contentTools.tools.explain_it_back.submitReview')

  return (
    <div
      className="space-y-3"
      data-content-tool="explain_it_back"
      data-testid="explain-it-back"
    >
      <div className="prose prose-sm dark:prose-invert max-w-none text-slate-800 dark:text-neutral-200">
        <p id={promptId} className="whitespace-pre-wrap text-sm font-medium">
          {prompt || t('contentTools.tools.explain_it_back.defaultPrompt')}
        </p>
        <p className="text-xs text-slate-600 dark:text-neutral-400">
          {t('contentTools.tools.explain_it_back.lengthGuide', {
            min: minWords,
            max: maxWords,
          })}
        </p>
      </div>

      {instructorNote?.text ? (
        <div
          className="rounded border border-sky-200 bg-sky-50 px-3 py-2 text-sm text-sky-950 dark:border-sky-900 dark:bg-sky-950/40 dark:text-sky-100"
          data-testid="explain-it-back-instructor-note"
        >
          <div className="text-xs font-semibold uppercase tracking-wide">
            {t('contentTools.tools.explain_it_back.instructorNote')}
          </div>
          <p className="mt-1 whitespace-pre-wrap">{instructorNote.text}</p>
        </div>
      ) : null}

      {showFeedback && latest?.feedback ? (
        <section
          id={feedbackId}
          role="region"
          aria-labelledby={`${feedbackId}-heading`}
          className="space-y-2 rounded border border-slate-200 bg-slate-50 px-3 py-3 dark:border-neutral-700 dark:bg-neutral-900/60"
          data-testid="explain-it-back-feedback"
        >
          <h3
            id={`${feedbackId}-heading`}
            className="text-sm font-semibold text-slate-800 dark:text-neutral-100"
          >
            {latest.feedback.mode === 'review'
              ? t('contentTools.tools.explain_it_back.reviewTitle')
              : t('contentTools.tools.explain_it_back.feedbackTitle')}
          </h3>

          {latest.feedback.mode === 'ai' && revealKeyPoints ? (
            <div className="space-y-1">
              <div className="text-xs font-medium text-slate-700 dark:text-neutral-300">
                {t('contentTools.tools.explain_it_back.whatYouGot')}
              </div>
              <div className="flex flex-wrap gap-1.5">
                {(latest.feedback.covered ?? []).map((id) => (
                  <span
                    key={id}
                    className="rounded border border-emerald-300 bg-emerald-50 px-2 py-0.5 text-xs text-emerald-900 dark:border-emerald-800 dark:bg-emerald-950 dark:text-emerald-100"
                  >
                    {labels[id] || id}
                  </span>
                ))}
                {(latest.feedback.covered ?? []).length === 0 ? (
                  <span className="text-xs text-slate-500">
                    {t('contentTools.tools.explain_it_back.noneYet')}
                  </span>
                ) : null}
              </div>
              {attempts.length > 1 && (latest.feedback.missing ?? []).length > 0 ? (
                <>
                  <div className="pt-1 text-xs font-medium text-slate-700 dark:text-neutral-300">
                    {t('contentTools.tools.explain_it_back.whatsMissing')}
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {latest.feedback.missing.map((id) => (
                      <span
                        key={id}
                        className="rounded border border-amber-300 bg-amber-50 px-2 py-0.5 text-xs text-amber-950 dark:border-amber-800 dark:bg-amber-950 dark:text-amber-100"
                      >
                        {labels[id] || id}
                      </span>
                    ))}
                  </div>
                </>
              ) : null}
            </div>
          ) : null}

          <p className="text-sm">
            <span className="font-medium">
              {t('contentTools.tools.explain_it_back.strength')}:{' '}
            </span>
            {latest.feedback.strength}
          </p>
          <p className="text-sm">
            <span className="font-medium">
              {t('contentTools.tools.explain_it_back.suggestion')}:{' '}
            </span>
            {latest.feedback.suggestion}
          </p>
          {latest.feedback.probe ? (
            <p className="text-sm">
              <span className="font-medium">
                {t('contentTools.tools.explain_it_back.probe')}:{' '}
              </span>
              {latest.feedback.probe}
            </p>
          ) : null}

          {attemptsLeft > 0 && !readOnly ? (
            <button
              type="button"
              className="text-sm font-medium text-sky-700 underline-offset-2 hover:underline dark:text-sky-300"
              onClick={startRevise}
              data-testid="explain-it-back-revise"
            >
              {t('contentTools.tools.explain_it_back.revise', { left: attemptsLeft })}
            </button>
          ) : null}
        </section>
      ) : null}

      {(!showFeedback || revising) && (
        <div className="space-y-2">
          <label className="block space-y-1 text-sm" htmlFor={`${promptId}-input`}>
            <span className="sr-only">{t('contentTools.tools.explain_it_back.inputLabel')}</span>
            <textarea
              id={`${promptId}-input`}
              ref={textareaRef}
              className="min-h-[8rem] w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              disabled={readOnly || busy}
              value={draft}
              aria-describedby={`${promptId} ${promptId}-count`}
              data-testid="explain-it-back-input"
              onChange={(e) => void onDraftChange(e.target.value)}
            />
          </label>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <p
              id={`${promptId}-count`}
              className="text-xs text-slate-600 dark:text-neutral-400"
              aria-live="polite"
              data-testid="explain-it-back-word-count"
            >
              {t('contentTools.tools.explain_it_back.wordCount', {
                count: wordCount,
                min: minWords,
                max: maxWords,
              })}
            </p>
            <p className="text-xs text-slate-500">
              {t('contentTools.tools.explain_it_back.attemptsLeft', { count: attemptsLeft })}
            </p>
          </div>
          <button
            type="button"
            className="rounded bg-slate-900 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
            disabled={!canSubmit}
            onClick={() => void onSubmit()}
            data-testid="explain-it-back-submit"
          >
            {busy
              ? t('contentTools.tools.explain_it_back.submitting')
              : submitLabel}
          </button>
        </div>
      )}

      {error ? (
        <p
          className="text-sm text-rose-700 dark:text-rose-300"
          role="alert"
          data-testid="explain-it-back-error"
        >
          {crisis ? error : error}
        </p>
      ) : null}

      <p className="text-xs text-slate-500 dark:text-neutral-500">
        {t('contentTools.tools.explain_it_back.practiceNote')}
      </p>
    </div>
  )
}
