import { useEffect, useId, useRef, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'

type Rating = 'again' | 'hard' | 'good' | 'easy'
type DeckStatus = {
  newCount: number
  dueCount: number
  learningCount: number
  laterCount: number
  nextDueAt?: string
  srsEnabled: boolean
  totalCards: number
  ratedCards: number
}
type CurrentCard = {
  cardId: string
  side: string
  prompt: string
  answer: string
  index: number
  total: number
  hint?: string
  imageUrl?: string
  imageAlt?: string
  promptLang?: string
  answerLang?: string
}
type ActionResult = {
  error?: string
  message?: string
  caughtUp?: boolean
  srsEnabled?: boolean
  status?: DeckStatus
  current?: CurrentCard | null
  sessionComplete?: boolean
  summary?: { reviewed: number; endedAt?: string }
  nextDueAt?: string
  state?: Record<string, unknown>
}

type Config = {
  title?: string
  cards?: Array<{ id: string; front: string; back: string }>
  reversePractice?: boolean
}

const RATINGS: Array<{ id: Rating; shortcut: string }> = [
  { id: 'again', shortcut: '1' },
  { id: 'hard', shortcut: '2' },
  { id: 'good', shortcut: '3' },
  { id: 'easy', shortcut: '4' },
]

export default function FlashcardsRenderer({
  config,
  state: _state,
  readOnly,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const titleId = useId()
  const cfg = (config ?? {}) as Config
  const title =
    typeof cfg.title === 'string' && cfg.title.trim()
      ? cfg.title.trim()
      : t('contentTools.tools.flashcards.defaultTitle')
  const cardCount = Array.isArray(cfg.cards) ? cfg.cards.length : 0

  const [deckStatus, setDeckStatus] = useState<DeckStatus | null>(null)
  const [srsEnabled, setSrsEnabled] = useState(true)
  const [current, setCurrent] = useState<CurrentCard | null>(null)
  const [revealed, setRevealed] = useState(false)
  const [showHint, setShowHint] = useState(false)
  const [caughtUp, setCaughtUp] = useState(false)
  const [sessionSummary, setSessionSummary] = useState<{ reviewed: number; nextDueAt?: string } | null>(
    null,
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showHelp, setShowHelp] = useState(false)
  const inSession = Boolean(current) && !sessionSummary
  // Host passes a fresh runAction closure each render; keep a ref so the mount
  // status fetch is not cancelled by identity churn (and setBusy re-renders).
  const runActionRef = useRef(runAction)
  runActionRef.current = runAction

  useEffect(() => {
    let cancelled = false
    void runActionRef
      .current('status', {})
      .then((raw) => {
        if (cancelled) return
        const res = raw as ActionResult
        if (res.status) setDeckStatus(res.status)
        if (typeof res.srsEnabled === 'boolean') setSrsEnabled(res.srsEnabled)
        if (res.current) {
          setCurrent(res.current)
          setRevealed(false)
        }
      })
      .catch(() => {
        /* status is best-effort */
      })
    return () => {
      cancelled = true
    }
    // Mount-only: start/rate responses already carry deck status + srsEnabled.
    // Re-running on `state`/`runAction` identity caused a cancel loop when status
    // applied envelope state and host re-created runAction.
  }, [])

  useEffect(() => {
    if (!inSession) return
    function onKey(e: KeyboardEvent) {
      if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement) return
      if (e.key === 'Escape') {
        e.preventDefault()
        void endSession()
        return
      }
      if (e.key === ' ' || e.code === 'Space') {
        if (!revealed) {
          e.preventDefault()
          setRevealed(true)
          announce(t('contentTools.tools.flashcards.answerRevealed'))
        }
        return
      }
      if (!revealed || readOnly || busy) return
      const map: Record<string, Rating> = { '1': 'again', '2': 'hard', '3': 'good', '4': 'easy' }
      const rating = map[e.key]
      if (rating) {
        e.preventDefault()
        void rate(rating)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- endSession/rate stable enough via busy/revealed
  }, [inSession, revealed, readOnly, busy, announce, t])

  async function startSession() {
    if (readOnly || busy) return
    setBusy(true)
    setError(null)
    setSessionSummary(null)
    setCaughtUp(false)
    try {
      const res = (await runAction('start_session', {})) as ActionResult
      if (res.error) {
        setError(res.message || res.error)
        return
      }
      if (typeof res.srsEnabled === 'boolean') setSrsEnabled(res.srsEnabled)
      if (res.status) setDeckStatus(res.status)
      if (res.caughtUp) {
        setCaughtUp(true)
        setCurrent(null)
        announce(t('contentTools.tools.flashcards.caughtUpAnnounce'))
        return
      }
      if (res.current) {
        setCurrent(res.current)
        setRevealed(false)
        setShowHint(false)
        announce(
          t('contentTools.tools.flashcards.cardAnnounce', {
            index: String((res.current.index ?? 0) + 1),
            total: String(res.current.total ?? 0),
          }),
        )
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.flashcards.actionError'))
    } finally {
      setBusy(false)
    }
  }

  async function rate(rating: Rating) {
    if (!current || readOnly || busy || !revealed) return
    setBusy(true)
    setError(null)
    try {
      const res = (await runAction('rate', {
        cardId: current.cardId,
        rating,
        side: current.side,
        idempotencyKey: crypto.randomUUID(),
      })) as ActionResult
      if (res.error) {
        setError(res.message || res.error)
        return
      }
      if (typeof res.srsEnabled === 'boolean') setSrsEnabled(res.srsEnabled)
      if (res.status) setDeckStatus(res.status)
      announce(
        t('contentTools.tools.flashcards.ratedAnnounce', {
          rating: t(`contentTools.tools.flashcards.ratings.${rating}`),
        }),
      )
      if (res.sessionComplete) {
        setSessionSummary({
          reviewed: res.summary?.reviewed ?? 0,
          nextDueAt: res.nextDueAt,
        })
        setCurrent(null)
        setRevealed(false)
        announce(t('contentTools.tools.flashcards.sessionCompleteAnnounce'))
        return
      }
      if (res.current) {
        setCurrent(res.current)
        setRevealed(false)
        setShowHint(false)
        announce(
          t('contentTools.tools.flashcards.cardAnnounce', {
            index: String((res.current.index ?? 0) + 1),
            total: String(res.current.total ?? 0),
          }),
        )
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.flashcards.actionError'))
    } finally {
      setBusy(false)
    }
  }

  async function endSession() {
    if (busy) return
    setBusy(true)
    try {
      const res = (await runAction('end_session', {})) as ActionResult
      if (res.status) setDeckStatus(res.status)
      setSessionSummary({
        reviewed: res.summary?.reviewed ?? 0,
        nextDueAt: typeof res.nextDueAt === 'string' ? res.nextDueAt : undefined,
      })
      setCurrent(null)
      setRevealed(false)
      announce(t('contentTools.tools.flashcards.sessionEndedAnnounce'))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.flashcards.actionError'))
    } finally {
      setBusy(false)
    }
  }

  const statusChips = deckStatus
    ? t('contentTools.tools.flashcards.statusChips', {
        new: String(deckStatus.newCount),
        due: String(deckStatus.dueCount),
      })
    : t('contentTools.tools.flashcards.cardCount', { count: String(cardCount) })

  return (
    <section
      data-content-tool="flashcards"
      data-testid="flashcards-tool"
      aria-labelledby={titleId}
      className="space-y-4 rounded-lg border border-border-default bg-surface-raised p-4 dark:border-border-default dark:bg-surface-raised"
    >
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div className="space-y-1">
          <h2 id={titleId} className="text-base font-semibold text-fg-default">
            {title}
          </h2>
          <p className="text-xs text-fg-muted" data-testid="flashcards-status-chips">
            {statusChips}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className="rounded-md border border-border-default px-2 py-1 text-xs dark:border-border-default"
            onClick={() => setShowHelp((v) => !v)}
            aria-expanded={showHelp}
          >
            {t('contentTools.tools.flashcards.shortcutsHelp')}
          </button>
          {!inSession && !readOnly ? (
            <button
              type="button"
              data-testid="flashcards-start"
              className="rounded-md bg-slate-900 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
              disabled={busy || cardCount < 3}
              onClick={() => void startSession()}
            >
              {t('contentTools.tools.flashcards.startSession')}
            </button>
          ) : null}
          {inSession && !readOnly ? (
            <button
              type="button"
              data-testid="flashcards-end"
              className="rounded-md border border-border-default px-3 py-1.5 text-xs dark:border-border-default"
              disabled={busy}
              onClick={() => void endSession()}
            >
              {t('contentTools.tools.flashcards.endSession')}
            </button>
          ) : null}
        </div>
      </header>

      {showHelp ? (
        <p className="text-xs text-fg-muted" data-testid="flashcards-shortcuts">
          {t('contentTools.tools.flashcards.shortcutsBody')}
        </p>
      ) : null}

      {!srsEnabled ? (
        <p
          className="rounded-md bg-amber-50 px-3 py-2 text-xs text-amber-950 dark:bg-amber-950/30 dark:text-amber-100"
          data-testid="flashcards-srs-off"
          role="status"
        >
          {t('contentTools.tools.flashcards.srsOffNote')}
        </p>
      ) : null}

      {caughtUp ? (
        <p data-testid="flashcards-caught-up" className="text-sm text-fg-default">
          {t('contentTools.tools.flashcards.caughtUp')}
        </p>
      ) : null}

      {sessionSummary ? (
        <div data-testid="flashcards-summary" className="space-y-1 text-sm text-fg-default">
          <p>
            {t('contentTools.tools.flashcards.sessionSummary', {
              reviewed: String(sessionSummary.reviewed),
            })}
          </p>
          {srsEnabled && sessionSummary.nextDueAt ? (
            <p className="text-xs text-fg-muted">
              {t('contentTools.tools.flashcards.nextDue', { date: sessionSummary.nextDueAt })}
            </p>
          ) : null}
        </div>
      ) : null}

      {current ? (
        <div
          data-testid="flashcards-card"
          className="space-y-4"
          aria-label={t('contentTools.tools.flashcards.cardRegion', {
            index: String(current.index + 1),
            total: String(current.total),
          })}
        >
          <p className="text-xs text-fg-muted">
            {t('contentTools.tools.flashcards.progress', {
              index: String(current.index + 1),
              total: String(current.total),
            })}
          </p>

          <div
            lang={current.promptLang || undefined}
            className="min-h-[5rem] rounded-md bg-surface-base px-4 py-6 text-center text-lg font-medium text-fg-default dark:bg-surface-overlay"
            data-testid="flashcards-prompt"
          >
            {current.prompt}
          </div>

          {current.imageUrl ? (
            <img
              src={current.imageUrl}
              alt={current.imageAlt || t('contentTools.tools.flashcards.imageFallbackAlt')}
              className="mx-auto max-h-48 rounded-md"
            />
          ) : null}

          {current.hint ? (
            <button
              type="button"
              className="text-xs text-fg-muted underline dark:text-fg-muted"
              onClick={() => setShowHint(true)}
            >
              {showHint ? current.hint : t('contentTools.tools.flashcards.showHint')}
            </button>
          ) : null}

          {!revealed ? (
            <button
              type="button"
              data-testid="flashcards-reveal"
              className="w-full rounded-md border border-border-strong px-3 py-2 text-sm font-medium dark:border-border-default"
              onClick={() => {
                setRevealed(true)
                announce(t('contentTools.tools.flashcards.answerRevealed'))
              }}
            >
              {t('contentTools.tools.flashcards.showAnswer')}
            </button>
          ) : (
            <div className="space-y-3">
              <div
                lang={current.answerLang || undefined}
                data-testid="flashcards-answer"
                className="motion-safe:animate-[fadeIn_120ms_ease-out] rounded-md bg-emerald-50 px-4 py-4 text-center text-base text-fg-default dark:bg-emerald-950/40"
              >
                {current.answer}
              </div>
              <div
                className="grid grid-cols-2 gap-2"
                role="group"
                aria-label={t('contentTools.tools.flashcards.rateGroup')}
              >
                {RATINGS.map((r) => (
                  <button
                    key={r.id}
                    type="button"
                    data-testid={`flashcards-rate-${r.id}`}
                    disabled={readOnly || busy}
                    className="rounded-md border border-border-default px-2 py-3 text-start text-sm dark:border-border-default disabled:opacity-50"
                    onClick={() => void rate(r.id)}
                  >
                    <span className="block font-medium">
                      {t(`contentTools.tools.flashcards.ratings.${r.id}`)}
                    </span>
                    <span className="block text-xs text-fg-muted">
                      {t(`contentTools.tools.flashcards.ratingHints.${r.id}`)} · {r.shortcut}
                    </span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      ) : null}

      {error ? (
        <p className="text-xs text-rose-600" role="alert">
          {error}
        </p>
      ) : null}
    </section>
  )
}
