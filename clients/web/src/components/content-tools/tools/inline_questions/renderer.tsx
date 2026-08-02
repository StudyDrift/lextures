import { useEffect, useId, useRef, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'

type QuestionType = 'single' | 'multi' | 'true_false' | 'short_text' | 'numeric'

type Option = { id: string; text: string }

type Question = {
  id: string
  type: QuestionType
  prompt: string
  options?: Option[]
  unit?: string
}

type Attempt = { value: unknown; correct: boolean; at: string; points?: number }
type QuestionAnswer = { attempts?: Attempt[]; revealed?: boolean }

type SubmitResult = {
  correct?: boolean
  feedback?: string
  explanation?: string
  correctAnswer?: unknown
  attemptsRemaining?: number
  error?: string
  message?: string
}

function shuffleStable<T extends { id: string }>(items: T[], seed: string): T[] {
  if (items.length <= 1) return items
  const out = [...items]
  let h = 0
  for (let i = 0; i < seed.length; i++) h = (h * 31 + seed.charCodeAt(i)) >>> 0
  for (let i = out.length - 1; i > 0; i--) {
    h = (h * 1103515245 + 12345) >>> 0
    const j = h % (i + 1)
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}

function attemptsUsed(answers: Record<string, QuestionAnswer>, qid: string): number {
  return answers[qid]?.attempts?.length ?? 0
}

/** Returns page size 1–3, or null for “show all”. */
function parseQuestionsAtATime(raw: unknown): number | null {
  if (raw === undefined || raw === null || raw === 'all') return null
  if (typeof raw === 'number' && Number.isFinite(raw)) {
    const n = Math.floor(raw)
    if (n >= 1 && n <= 3) return n
  }
  if (typeof raw === 'string') {
    const n = Number(raw)
    if (Number.isFinite(n)) {
      const i = Math.floor(n)
      if (i >= 1 && i <= 3) return i
    }
  }
  return null
}

function initialPageIndex(
  questions: Question[],
  answers: Record<string, QuestionAnswer>,
  pageSize: number | null,
): number {
  if (!pageSize || pageSize <= 0 || questions.length === 0) return 0
  const firstIncomplete = questions.findIndex((q) => attemptsUsed(answers, q.id) === 0)
  const idx = firstIncomplete === -1 ? questions.length - 1 : firstIncomplete
  return Math.floor(idx / pageSize)
}

export default function InlineQuestionsRenderer({
  instanceId,
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const labelId = useId()
  const feedbackRef = useRef<HTMLDivElement | null>(null)
  const questions = Array.isArray(config.questions)
    ? (config.questions as Question[]).slice(0, 3)
    : []
  const sequential = config.sequential === true
  const shuffleOptions = config.shuffleOptions === true
  const questionsAtATime = parseQuestionsAtATime(config.questionsAtATime)
  const label =
    typeof config.label === 'string' && config.label.trim()
      ? config.label
      : t('contentTools.tools.inline_questions.checkLabel')

  const answers = (state.answers && typeof state.answers === 'object'
    ? (state.answers as Record<string, QuestionAnswer>)
    : {}) as Record<string, QuestionAnswer>
  const drafts = (state.drafts && typeof state.drafts === 'object'
    ? (state.drafts as Record<string, unknown>)
    : {}) as Record<string, unknown>

  const [localDrafts, setLocalDrafts] = useState<Record<string, unknown>>(drafts)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [lastResult, setLastResult] = useState<Record<string, SubmitResult>>({})
  const [pageIndex, setPageIndex] = useState(() =>
    initialPageIndex(questions, answers, questionsAtATime),
  )

  useEffect(() => {
    setLocalDrafts(drafts)
  }, [JSON.stringify(drafts)])

  const pagingEnabled =
    questionsAtATime != null && questionsAtATime > 0 && questions.length > questionsAtATime
  const pageCount = pagingEnabled
    ? Math.ceil(questions.length / (questionsAtATime as number))
    : 1
  const safePage = Math.max(0, Math.min(pageIndex, pageCount - 1))
  const pageStart = pagingEnabled ? safePage * (questionsAtATime as number) : 0
  const pageEnd = pagingEnabled
    ? Math.min(questions.length, pageStart + (questionsAtATime as number))
    : questions.length
  const visibleQuestions = questions.slice(pageStart, pageEnd)

  function isUnlocked(qid: string): boolean {
    if (!sequential) return true
    for (const q of questions) {
      if (q.id === qid) return true
      if (attemptsUsed(answers, q.id) === 0) return false
    }
    return false
  }

  function canAdvancePage(): boolean {
    if (!pagingEnabled || safePage >= pageCount - 1) return false
    if (!sequential) return true
    // In sequential mode, require every visible question on this page to have been attempted.
    return visibleQuestions.every((q) => attemptsUsed(answers, q.id) > 0)
  }

  function optionsFor(q: Question): Option[] {
    const opts = q.options ?? []
    if (!shuffleOptions) return opts
    return shuffleStable(opts, `${instanceId}:${q.id}`)
  }

  function setDraft(qid: string, value: unknown) {
    setLocalDrafts((prev) => {
      const next = { ...prev, [qid]: value }
      void save({ drafts: next, v: 1, answers })
      return next
    })
  }

  async function onSubmit(q: Question) {
    if (readOnly || busyId) return
    const value = localDrafts[q.id]
    if (value === undefined || value === null || value === '') return
    setBusyId(q.id)
    try {
      const raw = await runAction('submit', { questionId: q.id, value })
      const result =
        raw && typeof raw === 'object' ? (raw as SubmitResult) : ({ correct: false } as SubmitResult)
      setLastResult((prev) => ({ ...prev, [q.id]: result }))
      if (result.error) {
        announce(result.message || result.error)
      } else if (result.correct) {
        announce(t('contentTools.tools.inline_questions.correctAnnounce'))
      } else {
        announce(t('contentTools.tools.inline_questions.incorrectAnnounce'))
      }
      requestAnimationFrame(() => feedbackRef.current?.focus())
    } catch {
      setLastResult((prev) => ({
        ...prev,
        [q.id]: { correct: false, error: 'error', message: t('contentTools.runtime.retry') },
      }))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <div className="space-y-4" data-content-tool="inline_questions">
      <p id={labelId} className="text-sm font-semibold text-slate-800 dark:text-neutral-100">
        {label}
      </p>
      {questions.length > 1 ? (
        <div
          className="flex items-center gap-1.5"
          role="group"
          aria-label={t('contentTools.tools.inline_questions.progress')}
        >
          {questions.map((q) => {
            const used = attemptsUsed(answers, q.id)
            const last = answers[q.id]?.attempts?.[used - 1]
            const done = Boolean(last?.correct) || (used > 0 && lastResult[q.id]?.attemptsRemaining === 0)
            return (
              <span
                key={q.id}
                className={
                  done
                    ? 'h-2 w-2 rounded-full bg-emerald-600 dark:bg-emerald-400'
                    : used > 0
                      ? 'h-2 w-2 rounded-full bg-amber-500'
                      : 'h-2 w-2 rounded-full bg-slate-300 dark:bg-neutral-600'
                }
                aria-hidden
              />
            )
          })}
        </div>
      ) : null}

      {visibleQuestions.map((q) => {
        const idx = questions.findIndex((item) => item.id === q.id)
        const unlocked = isUnlocked(q.id)
        const used = attemptsUsed(answers, q.id)
        const ans = answers[q.id]
        const last = ans?.attempts?.[used - 1]
        const result = lastResult[q.id]
        const exhausted =
          typeof result?.attemptsRemaining === 'number'
            ? result.attemptsRemaining === 0
            : false
        const locked = !unlocked
        const reviewed = Boolean(last) && (Boolean(last?.correct) || exhausted || Boolean(ans?.revealed))
        const feedbackId = `iq-feedback-${instanceId}-${q.id}`
        const legend = `${idx + 1}. ${q.prompt}`

        return (
          <fieldset
            key={q.id}
            disabled={readOnly || locked || busyId === q.id}
            aria-disabled={locked || undefined}
            aria-describedby={last || result ? feedbackId : undefined}
            className={
              locked
                ? 'space-y-2 opacity-50'
                : 'space-y-2'
            }
            data-question-id={q.id}
          >
            <legend className="text-sm font-medium text-slate-800 dark:text-neutral-100">
              {legend}
            </legend>
            {locked ? (
              <p className="text-xs text-slate-500 dark:text-neutral-400">
                {t('contentTools.tools.inline_questions.sequentialLocked')}
              </p>
            ) : null}

            {(q.type === 'single' || q.type === 'true_false') && (
              <div className="space-y-1.5" role="radiogroup" aria-label={q.prompt}>
                {optionsFor(q).map((opt) => {
                  const selected =
                    (localDrafts[q.id] as string | undefined) === opt.id ||
                    (last && !localDrafts[q.id] && last.value === opt.id)
                  return (
                    <label
                      key={opt.id}
                      className="flex min-h-11 cursor-pointer items-center gap-2 rounded-md px-1 text-sm text-slate-800 dark:text-neutral-100"
                    >
                      <input
                        type="radio"
                        name={`${instanceId}-${q.id}`}
                        value={opt.id}
                        checked={Boolean(selected)}
                        disabled={readOnly || locked || (reviewed && exhausted)}
                        onChange={() => setDraft(q.id, opt.id)}
                      />
                      <span>{opt.text}</span>
                    </label>
                  )
                })}
              </div>
            )}

            {q.type === 'multi' && (
              <div className="space-y-1.5" role="group" aria-label={q.prompt}>
                {optionsFor(q).map((opt) => {
                  const current = Array.isArray(localDrafts[q.id])
                    ? (localDrafts[q.id] as string[])
                    : Array.isArray(last?.value)
                      ? (last?.value as string[])
                      : []
                  const checked = current.includes(opt.id)
                  return (
                    <label
                      key={opt.id}
                      className="flex min-h-11 cursor-pointer items-center gap-2 rounded-md px-1 text-sm text-slate-800 dark:text-neutral-100"
                    >
                      <input
                        type="checkbox"
                        value={opt.id}
                        checked={checked}
                        disabled={readOnly || locked}
                        onChange={() => {
                          const next = checked
                            ? current.filter((id) => id !== opt.id)
                            : [...current, opt.id]
                          setDraft(q.id, next)
                        }}
                      />
                      <span>{opt.text}</span>
                    </label>
                  )
                })}
              </div>
            )}

            {q.type === 'short_text' && (
              <label className="block space-y-1">
                <span className="sr-only">{t('contentTools.runtime.yourAnswer')}</span>
                <input
                  type="text"
                  value={
                    typeof localDrafts[q.id] === 'string'
                      ? (localDrafts[q.id] as string)
                      : typeof last?.value === 'string'
                        ? (last.value as string)
                        : ''
                  }
                  disabled={readOnly || locked}
                  onChange={(e) => setDraft(q.id, e.target.value)}
                  className="w-full min-h-11 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-sm text-slate-900 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
                />
              </label>
            )}

            {q.type === 'numeric' && (
              <label className="flex min-h-11 items-center gap-2">
                <span className="sr-only">{t('contentTools.runtime.yourAnswer')}</span>
                <input
                  type="text"
                  inputMode="decimal"
                  value={
                    localDrafts[q.id] !== undefined && localDrafts[q.id] !== null
                      ? String(localDrafts[q.id])
                      : last?.value !== undefined
                        ? String(last.value)
                        : ''
                  }
                  disabled={readOnly || locked}
                  onChange={(e) => setDraft(q.id, e.target.value)}
                  className="w-40 rounded-md border border-slate-200 bg-white px-2.5 py-1.5 text-sm text-slate-900 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
                />
                {q.unit ? (
                  <span className="text-xs text-slate-500 dark:text-neutral-400">{q.unit}</span>
                ) : null}
              </label>
            )}

            <div className="flex flex-wrap items-center gap-2">
              {!readOnly && unlocked && !(last?.correct && reviewed) ? (
                <button
                  type="button"
                  disabled={busyId === q.id || localDrafts[q.id] === undefined || localDrafts[q.id] === ''}
                  onClick={() => void onSubmit(q)}
                  className="rounded-md bg-slate-800 px-3 py-2 text-xs font-medium text-white hover:bg-slate-700 disabled:opacity-50 dark:bg-neutral-200 dark:text-neutral-900"
                >
                  {last && !last.correct && !exhausted
                    ? t('contentTools.tools.inline_questions.tryAgain')
                    : t('contentTools.runtime.checkAnswer')}
                </button>
              ) : null}
            </div>

            {(result || last) && (
              <div
                id={feedbackId}
                ref={result ? feedbackRef : undefined}
                tabIndex={-1}
                role="status"
                aria-live="polite"
                className="space-y-1 text-sm"
                data-correct={
                  result?.correct === true || last?.correct === true ? 'true' : 'false'
                }
              >
                <p
                  className={
                    (result?.correct ?? last?.correct)
                      ? 'font-medium text-emerald-700 dark:text-emerald-300'
                      : 'font-medium text-rose-700 dark:text-rose-300'
                  }
                >
                  <span aria-hidden>
                    {(result?.correct ?? last?.correct) ? '✓ ' : '✗ '}
                  </span>
                  {(result?.correct ?? last?.correct)
                    ? t('contentTools.tools.inline_questions.correct')
                    : exhausted
                      ? t('contentTools.tools.inline_questions.exhausted')
                      : t('contentTools.tools.inline_questions.incorrect')}
                </p>
                {result?.feedback ? (
                  <p className="text-slate-700 dark:text-neutral-300">{result.feedback}</p>
                ) : null}
                {result?.explanation ? (
                  <p className="text-slate-600 dark:text-neutral-400">{result.explanation}</p>
                ) : null}
                {typeof result?.attemptsRemaining === 'number' && result.attemptsRemaining >= 0 ? (
                  <p className="text-xs text-slate-500 dark:text-neutral-400">
                    {t('contentTools.tools.inline_questions.attemptsLeft', {
                      count: result.attemptsRemaining,
                    })}
                  </p>
                ) : null}
                {result?.error === 'max_attempts' ? (
                  <p className="text-xs text-rose-600 dark:text-rose-400">
                    {result.message || t('contentTools.tools.inline_questions.exhausted')}
                  </p>
                ) : null}
              </div>
            )}
          </fieldset>
        )
      })}

      {pagingEnabled ? (
        <div className="flex flex-wrap items-center justify-between gap-2 border-t border-slate-200 pt-3 dark:border-neutral-700">
          <p className="text-xs text-slate-500 dark:text-neutral-400">
            {t('contentTools.tools.inline_questions.pageOf', {
              from: pageStart + 1,
              to: pageEnd,
              total: questions.length,
            })}
          </p>
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={safePage <= 0}
              onClick={() => setPageIndex((p) => Math.max(0, p - 1))}
              className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-800 hover:bg-slate-50 disabled:opacity-40 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
            >
              {t('contentTools.tools.inline_questions.previous')}
            </button>
            <button
              type="button"
              disabled={!canAdvancePage()}
              onClick={() => setPageIndex((p) => Math.min(pageCount - 1, p + 1))}
              className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-800 hover:bg-slate-50 disabled:opacity-40 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:hover:bg-neutral-800"
            >
              {t('contentTools.tools.inline_questions.next')}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
