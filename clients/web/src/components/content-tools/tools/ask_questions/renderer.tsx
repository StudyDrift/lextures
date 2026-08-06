import { useEffect, useId, useRef, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'

type Citation = {
  kind: string
  id: string
  title: string
  url?: string
}

type Turn = {
  id: string
  role: 'user' | 'assistant'
  text: string
  citations?: Citation[]
  createdAt: string
  error?: string
}

type AskResult = {
  turn?: Turn
  questionsLeft?: number
  citationCount?: number
  error?: string
  message?: string
  preserveInput?: boolean
  askInstructor?: boolean
  resetAt?: string
  crisis?: boolean
}

function asTurns(state: Record<string, unknown>): Turn[] {
  const raw = state.turns
  if (!Array.isArray(raw)) return []
  return raw.filter((t): t is Turn => {
    if (!t || typeof t !== 'object') return false
    const row = t as Turn
    return (row.role === 'user' || row.role === 'assistant') && typeof row.text === 'string'
  })
}

export default function AskQuestionsRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const listId = useId()
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const turns = asTurns(state)
  const remoteDraft = typeof state.draft === 'string' ? state.draft : ''
  const [draft, setDraft] = useState(remoteDraft)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [askInstructor, setAskInstructor] = useState(false)

  const intro = typeof config.intro === 'string' ? config.intro.trim() : ''
  const placeholder =
    typeof config.placeholder === 'string' && config.placeholder.trim()
      ? config.placeholder
      : t('contentTools.tools.ask_questions.placeholder')
  const maxPerDay =
    typeof config.maxQuestionsPerDay === 'number' && config.maxQuestionsPerDay > 0
      ? config.maxQuestionsPerDay
      : 20
  const askedToday =
    state.askedToday && typeof state.askedToday === 'object'
      ? (state.askedToday as { date?: string; count?: number })
      : null
  const today = new Date().toISOString().slice(0, 10)
  const usedToday =
    askedToday && askedToday.date === today && typeof askedToday.count === 'number'
      ? askedToday.count
      : 0
  const questionsLeft = Math.max(0, maxPerDay - usedToday)
  const showCitations = config.showCitations !== false

  useEffect(() => {
    setDraft(remoteDraft)
  }, [remoteDraft])

  async function onAsk() {
    const question = draft.trim()
    if (!question || busy || readOnly) return
    setBusy(true)
    setError(null)
    setAskInstructor(false)
    try {
      const raw = await runAction('ask', { question })
      const result =
        raw && typeof raw === 'object' ? (raw as AskResult) : ({} as AskResult)
      if (result.error) {
        setError(result.message || result.error)
        setAskInstructor(Boolean(result.askInstructor))
        if (result.preserveInput) {
          // Keep draft; do not clear.
        } else {
          setDraft('')
        }
        return
      }
      setDraft('')
      void save({ draft: '' })
      const citeCount =
        typeof result.citationCount === 'number'
          ? result.citationCount
          : result.turn?.citations?.length ?? 0
      announce(
        t('contentTools.tools.ask_questions.answerReceived', {
          count: citeCount,
        }),
      )
      // Keep focus in the input (AC-11).
      inputRef.current?.focus()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.runtime.retry'))
    } finally {
      setBusy(false)
    }
  }

  async function onClear() {
    if (busy || readOnly) return
    setBusy(true)
    setError(null)
    try {
      await runAction('clear', {})
      setDraft('')
      announce(t('contentTools.tools.ask_questions.cleared'))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.runtime.retry'))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="space-y-3" data-content-tool="ask_questions" data-testid="ask-questions">
      {intro ? (
        <div className="prose prose-sm dark:prose-invert max-w-none text-fg-default">
          <p className="whitespace-pre-wrap text-sm">{intro}</p>
        </div>
      ) : null}

      <div
        id={listId}
        role="log"
        aria-live="polite"
        aria-relevant="additions"
        aria-label={t('contentTools.tools.ask_questions.messagesLabel')}
        className="max-h-72 space-y-2 overflow-y-auto rounded-md border border-border-default bg-slate-50/60 p-2 dark:border-border-default/40"
        data-testid="ask-questions-log"
      >
        {turns.length === 0 ? (
          <p className="px-1 py-2 text-xs text-fg-muted">
            {t('contentTools.tools.ask_questions.empty')}
          </p>
        ) : (
          turns.map((turn) => (
            <div
              key={turn.id}
              className={
                turn.role === 'user'
                  ? 'ms-8 rounded-md bg-slate-800 px-2.5 py-1.5 text-sm text-white dark:bg-neutral-200 dark:text-neutral-900'
                  : 'me-4 rounded-md border border-border-default bg-surface-raised px-2.5 py-1.5 text-sm text-fg-default dark:border-border-default dark:bg-surface-raised dark:text-fg-default'
              }
              data-role={turn.role}
            >
              {turn.role === 'assistant' ? (
                <span className="mb-1 inline-block text-[10px] font-semibold uppercase tracking-wide text-fg-muted">
                  {t('contentTools.tools.ask_questions.aiBadge')}
                </span>
              ) : null}
              <p className="whitespace-pre-wrap">{turn.text}</p>
              {showCitations && turn.role === 'assistant' && turn.citations && turn.citations.length > 0 ? (
                <div className="mt-2 flex flex-wrap gap-1.5" data-testid="ask-questions-sources">
                  <span className="text-[10px] font-medium uppercase text-fg-muted">
                    {t('contentTools.tools.ask_questions.sources')}
                  </span>
                  {turn.citations.map((c, i) => {
                    const label = t('contentTools.tools.ask_questions.sourceChip', {
                      n: i + 1,
                      title: c.title || c.id,
                    })
                    if (c.url) {
                      return (
                        <a
                          key={`${turn.id}-${c.id}`}
                          href={c.url}
                          target="_blank"
                          rel="noreferrer"
                          className="rounded bg-surface-sunken px-1.5 py-0.5 text-[11px] text-fg-muted underline dark:bg-surface-overlay dark:text-fg-default"
                          aria-label={label}
                        >
                          {i + 1}. {c.title || c.id}
                        </a>
                      )
                    }
                    return (
                      <span
                        key={`${turn.id}-${c.id}`}
                        className="rounded bg-surface-sunken px-1.5 py-0.5 text-[11px] text-fg-muted dark:bg-surface-overlay dark:text-fg-default"
                        aria-label={label}
                      >
                        {i + 1}. {c.title || c.id}
                      </span>
                    )
                  })}
                </div>
              ) : null}
            </div>
          ))
        )}
        {busy ? (
          <p className="text-xs text-fg-muted" role="status">
            {t('contentTools.tools.ask_questions.thinking')}
          </p>
        ) : null}
      </div>

      {!readOnly ? (
        <div className="space-y-2">
          <label className="block space-y-1">
            <span className="text-xs font-medium text-fg-muted">
              {t('contentTools.tools.ask_questions.inputLabel')}
            </span>
            <textarea
              ref={inputRef}
              value={draft}
              disabled={busy}
              rows={2}
              maxLength={4000}
              placeholder={placeholder}
              aria-controls={listId}
              data-testid="ask-questions-input"
              onChange={(e) => {
                const value = e.target.value
                setDraft(value)
                void save({ draft: value })
              }}
              onKeyDown={(e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
                  e.preventDefault()
                  void onAsk()
                }
              }}
              className="w-full rounded-md border border-border-default bg-surface-raised px-2.5 py-1.5 text-sm text-fg-default dark:border-border-default dark:bg-surface-base dark:text-fg-default"
            />
          </label>
          <div className="flex flex-wrap items-center gap-2">
            <button
              type="button"
              disabled={busy || !draft.trim()}
              onClick={() => void onAsk()}
              data-testid="ask-questions-submit"
              className="rounded-md bg-slate-800 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700 disabled:opacity-50 dark:bg-neutral-200 dark:text-neutral-900 dark:hover:bg-neutral-100"
            >
              {t('contentTools.tools.ask_questions.ask')}
            </button>
            <button
              type="button"
              disabled={busy || turns.length === 0}
              onClick={() => void onClear()}
              data-testid="ask-questions-clear"
              className="rounded-md border border-border-strong px-3 py-1.5 text-xs font-medium text-fg-muted hover:bg-surface-base disabled:opacity-50 dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
            >
              {t('contentTools.tools.ask_questions.clear')}
            </button>
            <span className="text-[11px] text-fg-muted" data-testid="ask-questions-remaining">
              {t('contentTools.tools.ask_questions.remaining', {
                left: questionsLeft,
                max: maxPerDay,
              })}
            </span>
            <span className="text-[11px] text-fg-subtle">
              {draft.length}/4000
            </span>
          </div>
        </div>
      ) : null}

      {error ? (
        <div
          className="rounded-md border border-rose-200 bg-rose-50 px-2.5 py-2 text-xs text-rose-800 dark:border-rose-900 dark:bg-rose-950/40 dark:text-rose-100"
          role="alert"
          data-testid="ask-questions-error"
        >
          <p>{error}</p>
          {askInstructor ? (
            <p className="mt-1">{t('contentTools.tools.ask_questions.askInstructor')}</p>
          ) : (
            <button
              type="button"
              className="mt-1 underline"
              onClick={() => void onAsk()}
              disabled={busy || !draft.trim()}
            >
              {t('contentTools.runtime.retry')}
            </button>
          )}
        </div>
      ) : null}
    </div>
  )
}
