import { useEffect, useId, useRef, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'

type Outcome = { id: string; text: string }
type Reveal = { markdown: string; imageUrl?: string }
type Prediction = { outcomeId?: string; text?: string }
type PeerResults = {
  suppressed: boolean
  reason?: string
  learners: number
  outcomes?: Array<{ outcomeId: string; count: number }>
  confidenceBuckets?: Array<{ bucket: string; count: number }>
}

type CommitResult = {
  reveal?: Reveal
  peerResults?: PeerResults
  error?: string
  message?: string
  preserveInput?: boolean
}

const THREE_LABELS: Record<string, string> = {
  '1': 'contentTools.tools.predict_reveal.confidence.guessing',
  '2': 'contentTools.tools.predict_reveal.confidence.fairlySure',
  '3': 'contentTools.tools.predict_reveal.confidence.certain',
}

function confidenceOptions(
  scale: string,
  overrides: Record<string, string> | undefined,
  t: ContentToolRendererProps['t'],
): Array<{ value: number; label: string }> {
  if (scale === 'none') return []
  if (scale === 'three') {
    return [1, 2, 3].map((v) => ({
      value: v,
      label:
        overrides?.[String(v)] ||
        overrides?.[v === 1 ? 'guessing' : v === 2 ? 'fairly_sure' : 'certain'] ||
        t(THREE_LABELS[String(v)]),
    }))
  }
  if (scale === 'five') {
    return [1, 2, 3, 4, 5].map((v) => ({
      value: v,
      label: overrides?.[String(v)] || String(v),
    }))
  }
  // percent: present as 5 radios at 0/25/50/75/100 for keyboard a11y (not a bare slider)
  return [0, 25, 50, 75, 100].map((v) => ({
    value: v,
    label: overrides?.[String(v)] || `${v}%`,
  }))
}

export default function PredictRevealRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const questionId = useId()
  const predLegendId = useId()
  const confLegendId = useId()
  const revealHeadingRef = useRef<HTMLHeadingElement | null>(null)

  const question = typeof config.question === 'string' ? config.question : ''
  const mode = config.mode === 'open' ? 'open' : 'choice'
  const outcomes = Array.isArray(config.outcomes) ? (config.outcomes as Outcome[]) : []
  const openPlaceholder =
    typeof config.openPlaceholder === 'string' ? config.openPlaceholder : ''
  const confidenceScale =
    typeof config.confidenceScale === 'string' ? config.confidenceScale : 'three'
  const confidenceRequired = config.confidenceRequired !== false
  const reflectionPrompt =
    typeof config.reflectionPrompt === 'string' ? config.reflectionPrompt : ''
  const confidenceLabels =
    config.confidenceLabels && typeof config.confidenceLabels === 'object'
      ? (config.confidenceLabels as Record<string, string>)
      : undefined

  const committedAt = typeof state.committedAt === 'string' ? state.committedAt : ''
  const committed = Boolean(committedAt)
  const savedPrediction = (state.prediction ?? null) as Prediction | null
  const draft = (state.draft && typeof state.draft === 'object'
    ? state.draft
    : {}) as { outcomeId?: string; text?: string; confidence?: number }
  const savedReflection = typeof state.reflection === 'string' ? state.reflection : ''

  const [outcomeId, setOutcomeId] = useState(draft.outcomeId || savedPrediction?.outcomeId || '')
  const [text, setText] = useState(draft.text || savedPrediction?.text || '')
  const [confidence, setConfidence] = useState<number | null>(
    typeof draft.confidence === 'number' ? draft.confidence : null,
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [reveal, setReveal] = useState<Reveal | null>(null)
  const [peerResults, setPeerResults] = useState<PeerResults | null>(null)
  const [reflection, setReflection] = useState(savedReflection)
  const [liveMsg, setLiveMsg] = useState('')
  const runActionRef = useRef(runAction)
  runActionRef.current = runAction
  const fetchedRevealForCommit = useRef<string | null>(null)

  useEffect(() => {
    if (!committed || !committedAt) return
    if (fetchedRevealForCommit.current === committedAt && reveal) return
    let cancelled = false
    void (async () => {
      try {
        const raw = (await runActionRef.current('commit', {})) as CommitResult
        if (cancelled) return
        if (raw?.reveal) {
          fetchedRevealForCommit.current = committedAt
          setReveal(raw.reveal)
          setPeerResults(raw.peerResults ?? null)
        }
      } catch {
        // reveal stays null; student still sees prediction from state
      }
    })()
    return () => {
      cancelled = true
    }
  }, [committed, committedAt, reveal])

  useEffect(() => {
    if (!reveal) return
    const msg = t('contentTools.tools.predict_reveal.revealedAnnounce')
    setLiveMsg(msg)
    announce(msg)
    revealHeadingRef.current?.focus()
  }, [reveal, announce, t])

  function persistDraft(next: {
    outcomeId?: string
    text?: string
    confidence?: number | null
  }) {
    if (committed || readOnly) return
    const draftNext: Record<string, unknown> = {}
    if (next.outcomeId) draftNext.outcomeId = next.outcomeId
    if (next.text) draftNext.text = next.text
    if (typeof next.confidence === 'number') draftNext.confidence = next.confidence
    void save({ draft: draftNext, v: 1 })
  }

  const confOpts = confidenceOptions(confidenceScale, confidenceLabels, t)
  const canCommit =
    !committed &&
    !busy &&
    !readOnly &&
    (mode === 'choice' ? Boolean(outcomeId) : Boolean(text.trim())) &&
    (confidenceScale === 'none' || !confidenceRequired || confidence != null)

  async function onCommit() {
    if (!canCommit) {
      if (confidenceScale !== 'none' && confidenceRequired && confidence == null) {
        setError(t('contentTools.tools.predict_reveal.confidenceRequired'))
      }
      return
    }
    setBusy(true)
    setError(null)
    try {
      const input: Record<string, unknown> = {
        prediction:
          mode === 'choice' ? { outcomeId } : { text: text.trim() },
      }
      if (confidenceScale !== 'none' && confidence != null) {
        input.confidence = confidence
      }
      const raw = (await runAction('commit', input)) as CommitResult
      if (raw?.error) {
        setError(raw.message || raw.error)
        return
      }
      if (raw?.reveal) {
        setReveal(raw.reveal)
        setPeerResults(raw.peerResults ?? null)
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.predict_reveal.commitError'))
    } finally {
      setBusy(false)
    }
  }

  async function onReflect() {
    if (!committed || readOnly || !reflection.trim()) return
    setBusy(true)
    setError(null)
    try {
      const raw = (await runAction('reflect', { text: reflection.trim() })) as CommitResult
      if (raw?.error) {
        setError(raw.message || raw.error)
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.predict_reveal.commitError'))
    } finally {
      setBusy(false)
    }
  }

  const predictionLabel = (() => {
    if (mode === 'open') return savedPrediction?.text || text
    const id = savedPrediction?.outcomeId || outcomeId
    return outcomes.find((o) => o.id === id)?.text || id
  })()

  return (
    <div className="space-y-4" data-content-tool="predict_reveal" data-testid="predict-reveal">
      <p id={questionId} className="text-sm font-medium text-slate-900 dark:text-neutral-100">
        {question}
      </p>

      {!committed ? (
        <>
          <fieldset disabled={readOnly || busy} aria-labelledby={predLegendId}>
            <legend id={predLegendId} className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-300">
              {t('contentTools.tools.predict_reveal.yourPrediction')}
            </legend>
            {mode === 'choice' ? (
              <div className="space-y-2" role="radiogroup" aria-labelledby={predLegendId}>
                {outcomes.map((o) => (
                  <label
                    key={o.id}
                    className="flex cursor-pointer items-start gap-2 rounded border border-slate-200 px-3 py-2 text-sm dark:border-neutral-700"
                  >
                    <input
                      type="radio"
                      name={`${questionId}-pred`}
                      value={o.id}
                      checked={outcomeId === o.id}
                      onChange={() => {
                        setOutcomeId(o.id)
                        persistDraft({ outcomeId: o.id, text, confidence })
                      }}
                      className="mt-0.5"
                    />
                    <span>{o.text}</span>
                  </label>
                ))}
              </div>
            ) : (
              <textarea
                className="w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                rows={3}
                placeholder={openPlaceholder || t('contentTools.tools.predict_reveal.openPlaceholder')}
                value={text}
                aria-labelledby={predLegendId}
                onChange={(e) => {
                  setText(e.target.value)
                  persistDraft({ outcomeId, text: e.target.value, confidence })
                }}
              />
            )}
          </fieldset>

          {confidenceScale !== 'none' ? (
            <fieldset disabled={readOnly || busy} aria-labelledby={confLegendId}>
              <legend id={confLegendId} className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-300">
                {t('contentTools.tools.predict_reveal.howSure')}
              </legend>
              <div className="flex flex-wrap gap-3" role="radiogroup" aria-labelledby={confLegendId}>
                {confOpts.map((opt) => (
                  <label key={opt.value} className="flex cursor-pointer items-center gap-1.5 text-sm">
                    <input
                      type="radio"
                      name={`${questionId}-conf`}
                      value={opt.value}
                      checked={confidence === opt.value}
                      onChange={() => {
                        setConfidence(opt.value)
                        persistDraft({ outcomeId, text, confidence: opt.value })
                      }}
                    />
                    <span>{opt.label}</span>
                  </label>
                ))}
              </div>
            </fieldset>
          ) : null}

          {!readOnly ? (
            <div className="space-y-1">
              <button
                type="button"
                data-testid="predict-reveal-commit"
                className="rounded bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
                disabled={!canCommit}
                onClick={() => void onCommit()}
              >
                {busy
                  ? t('contentTools.tools.predict_reveal.committing')
                  : t('contentTools.tools.predict_reveal.commit')}
              </button>
              <p className="text-xs text-slate-500 dark:text-neutral-400">
                {t('contentTools.tools.predict_reveal.commitHelper')}
              </p>
            </div>
          ) : null}
        </>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2" data-testid="predict-reveal-side-by-side">
          <div className="rounded border border-slate-200 p-3 dark:border-neutral-700">
            <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-600 dark:text-neutral-300">
              {t('contentTools.tools.predict_reveal.yourPrediction')}
            </h3>
            <p className="text-sm text-slate-900 dark:text-neutral-100">{predictionLabel}</p>
          </div>
          <div className="rounded border border-slate-200 p-3 dark:border-neutral-700">
            <h3
              ref={revealHeadingRef}
              tabIndex={-1}
              className="mb-1 text-xs font-semibold uppercase tracking-wide text-slate-600 outline-none dark:text-neutral-300"
              data-testid="predict-reveal-heading"
            >
              {t('contentTools.tools.predict_reveal.whatHappens')}
            </h3>
            {reveal ? (
              <div className="space-y-2 text-sm text-slate-900 dark:text-neutral-100">
                <p className="whitespace-pre-wrap">{reveal.markdown}</p>
                {reveal.imageUrl ? (
                  <img src={reveal.imageUrl} alt="" className="max-h-48 rounded" />
                ) : null}
              </div>
            ) : (
              <p className="text-sm text-slate-500">{t('contentTools.runtime.loading')}</p>
            )}
          </div>
        </div>
      )}

      {peerResults ? (
        <div data-testid="predict-reveal-peers" className="text-sm text-slate-700 dark:text-neutral-200">
          {peerResults.suppressed ? (
            <p>{t('contentTools.tools.predict_reveal.peersSuppressed')}</p>
          ) : (
            <div>
              <h3 className="mb-1 text-xs font-semibold uppercase tracking-wide">
                {t('contentTools.tools.predict_reveal.peerResults')}
              </h3>
              <ul className="list-disc ps-4">
                {(peerResults.outcomes ?? []).map((o) => {
                  const label = outcomes.find((x) => x.id === o.outcomeId)?.text || o.outcomeId
                  return (
                    <li key={o.outcomeId}>
                      {label}: {o.count}
                    </li>
                  )
                })}
              </ul>
            </div>
          )}
        </div>
      ) : null}

      {committed && reflectionPrompt && !readOnly ? (
        <div className="space-y-2">
          <label className="block text-sm font-medium text-slate-800 dark:text-neutral-100">
            {reflectionPrompt}
            <textarea
              className="mt-1 w-full rounded border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-950"
              rows={2}
              value={reflection}
              onChange={(e) => setReflection(e.target.value)}
              data-testid="predict-reveal-reflection"
            />
          </label>
          <button
            type="button"
            className="rounded border border-slate-300 px-3 py-1.5 text-sm dark:border-neutral-600"
            disabled={busy || !reflection.trim()}
            onClick={() => void onReflect()}
          >
            {t('contentTools.tools.predict_reveal.saveReflection')}
          </button>
        </div>
      ) : null}

      {committed && reflectionPrompt && readOnly && savedReflection ? (
        <p className="text-sm text-slate-700 dark:text-neutral-200">
          <span className="font-medium">{reflectionPrompt}</span> {savedReflection}
        </p>
      ) : null}

      {error ? (
        <p className="text-sm text-rose-600" role="alert" data-testid="predict-reveal-error">
          {error}
        </p>
      ) : null}

      <div className="sr-only" aria-live="polite">
        {liveMsg}
      </div>
    </div>
  )
}
