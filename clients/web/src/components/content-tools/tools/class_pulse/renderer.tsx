import { useEffect, useId, useRef, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'

type Option = { id: string; text: string }
type Vote = { round: 1 | 2; optionId: string; at: string }
type CountedOption = { optionId: string; count: number; percent?: number }
type RoundAggregate = {
  round: number
  suppressed: boolean
  reason?: string
  learners: number
  options?: CountedOption[]
}
type Reveal = { correctOptionId?: string; explanation?: string }
type VoteResult = {
  error?: string
  message?: string
  aggregate?: RoundAggregate
  aggregateRound2?: RoundAggregate
  suppressed?: boolean
  reason?: string
  reveal?: Reveal
  state?: { votes?: Vote[]; completedAt?: string }
}

const POLL_MS = 30_000

function optionLabel(options: Option[], id: string): string {
  return options.find((o) => o.id === id)?.text || id
}

function DistributionView({
  title,
  aggregate,
  options,
  yourOptionId,
  showPercentages,
  t,
  testId,
}: {
  title: string
  aggregate: RoundAggregate
  options: Option[]
  yourOptionId?: string
  showPercentages: boolean
  t: ContentToolRendererProps['t']
  testId: string
}) {
  if (aggregate.suppressed) {
    return (
      <div data-testid={testId} className="space-y-1 text-sm text-fg-default">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
          {title}
        </h3>
        <p data-testid={`${testId}-suppressed`}>
          {t('contentTools.tools.class_pulse.suppressed', {
            min: String(aggregate.learners),
            count: String(aggregate.learners),
          })}
        </p>
        <p className="text-xs text-fg-muted">
          {t('contentTools.tools.class_pulse.waitingForMore', {
            count: String(aggregate.learners),
          })}
        </p>
      </div>
    )
  }

  const rows = (aggregate.options ?? []).map((o) => ({
    ...o,
    label: optionLabel(options, o.optionId),
    yours: yourOptionId === o.optionId,
  }))
  const max = Math.max(1, ...rows.map((r) => r.count))

  return (
    <div data-testid={testId} className="space-y-3">
      <div className="flex items-baseline justify-between gap-2">
        <h3 className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
          {title}
        </h3>
        <p className="text-xs text-fg-muted">
          {t('contentTools.tools.class_pulse.respondents', { count: String(aggregate.learners) })}
        </p>
      </div>

      <ul className="space-y-2" aria-hidden="true">
        {rows.map((r) => {
          const width = `${Math.round((r.count / max) * 100)}%`
          return (
            <li key={r.optionId} className="space-y-1">
              <div className="flex flex-wrap items-baseline justify-between gap-2 text-sm">
                <span className="font-medium text-fg-default">
                  {r.label}
                  {r.yours ? (
                    <span className="ms-2 text-xs font-normal text-fg-muted">
                      {t('contentTools.tools.class_pulse.yourAnswer')}
                    </span>
                  ) : null}
                </span>
                <span className="text-xs text-fg-muted">
                  {showPercentages && typeof r.percent === 'number'
                    ? t('contentTools.tools.class_pulse.countPercent', {
                        count: String(r.count),
                        percent: String(r.percent),
                      })
                    : t('contentTools.tools.class_pulse.countOnly', { count: String(r.count) })}
                </span>
              </div>
              <div className="h-2 w-full overflow-hidden rounded bg-surface-sunken">
                <div
                  className="h-full bg-slate-700 dark:bg-neutral-300"
                  style={{ width }}
                  data-testid={`${testId}-bar-${r.optionId}`}
                />
              </div>
            </li>
          )
        })}
      </ul>

      <table className="w-full text-sm text-fg-default" data-testid={`${testId}-table`}>
        <caption className="sr-only">{title}</caption>
        <thead>
          <tr className="border-b border-border-default text-start text-xs uppercase tracking-wide text-fg-muted dark:border-border-default dark:text-fg-muted">
            <th scope="col" className="py-1 pe-2">
              {t('contentTools.tools.class_pulse.colOption')}
            </th>
            <th scope="col" className="py-1 pe-2">
              {t('contentTools.tools.class_pulse.colCount')}
            </th>
            {showPercentages ? (
              <th scope="col" className="py-1">
                {t('contentTools.tools.class_pulse.colPercent')}
              </th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.optionId} className="border-b border-border-subtle">
              <td className="py-1 pe-2">
                {r.label}
                {r.yours ? ` (${t('contentTools.tools.class_pulse.yourAnswer')})` : ''}
              </td>
              <td className="py-1 pe-2">{r.count}</td>
              {showPercentages ? (
                <td className="py-1">{typeof r.percent === 'number' ? `${r.percent}%` : '—'}</td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

export default function ClassPulseRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const questionId = useId()
  const legendId = useId()

  const question = typeof config.question === 'string' ? config.question : ''
  const options = Array.isArray(config.options) ? (config.options as Option[]) : []
  const allowSecondVote = config.allowSecondVote === true
  const showPercentages = config.showPercentages !== false

  const votes = (Array.isArray(state.votes) ? state.votes : []) as Vote[]
  const vote1 = votes.find((v) => v.round === 1)
  const vote2 = votes.find((v) => v.round === 2)
  const draft = (state.draft && typeof state.draft === 'object'
    ? state.draft
    : {}) as { optionId?: string; round?: number }

  const activeRound: 1 | 2 = vote1 && allowSecondVote && !vote2 ? 2 : 1
  const [selected, setSelected] = useState(
    draft.optionId || (activeRound === 2 ? '' : vote1?.optionId) || '',
  )
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [aggregate, setAggregate] = useState<RoundAggregate | null>(null)
  const [aggregateRound2, setAggregateRound2] = useState<RoundAggregate | null>(null)
  const [reveal, setReveal] = useState<Reveal | null>(null)
  const [liveMsg, setLiveMsg] = useState('')
  const announcedRef = useRef(false)
  const runActionRef = useRef(runAction)
  runActionRef.current = runAction

  const hasVoted = Boolean(vote1)

  useEffect(() => {
    if (!hasVoted) return
    let cancelled = false
    void (async () => {
      try {
        const raw = (await runActionRef.current('aggregate', {})) as VoteResult
        if (cancelled || raw?.error) return
        if (raw.aggregate) setAggregate(raw.aggregate)
        if (raw.aggregateRound2) setAggregateRound2(raw.aggregateRound2)
        if (raw.reveal) setReveal(raw.reveal)
      } catch {
        // keep prior aggregate
      }
    })()
    return () => {
      cancelled = true
    }
  }, [hasVoted, vote1?.at, vote2?.at])

  useEffect(() => {
    if (!hasVoted) return
    let timer: ReturnType<typeof setInterval> | null = null

    const tick = () => {
      if (typeof document !== 'undefined' && document.visibilityState === 'hidden') return
      void (async () => {
        try {
          const raw = (await runActionRef.current('aggregate', {})) as VoteResult
          if (raw?.error) return
          if (raw.aggregate) setAggregate(raw.aggregate)
          if (raw.aggregateRound2) setAggregateRound2(raw.aggregateRound2)
          if (raw.reveal) setReveal(raw.reveal)
        } catch {
          // ignore poll errors
        }
      })()
    }

    const onVisibility = () => {
      if (document.visibilityState === 'visible') tick()
    }

    timer = setInterval(tick, POLL_MS)
    document.addEventListener('visibilitychange', onVisibility)
    return () => {
      if (timer) clearInterval(timer)
      document.removeEventListener('visibilitychange', onVisibility)
    }
  }, [hasVoted])

  useEffect(() => {
    if (!aggregate || announcedRef.current) return
    if (aggregate.suppressed) {
      const msg = t('contentTools.tools.class_pulse.suppressedAnnounce')
      setLiveMsg(msg)
      announce(msg)
      announcedRef.current = true
      return
    }
    const msg = t('contentTools.tools.class_pulse.resultsAnnounce', {
      count: String(aggregate.learners),
    })
    setLiveMsg(msg)
    announce(msg)
    announcedRef.current = true
  }, [aggregate, announce, t])

  function persistDraft(optionId: string, round: number) {
    if (readOnly) return
    if (round === 1 && vote1) return
    if (round === 2 && vote2) return
    void save({ draft: { optionId, round }, v: 1, votes })
  }

  const canVote =
    !busy &&
    !readOnly &&
    Boolean(selected) &&
    ((activeRound === 1 && !vote1) || (activeRound === 2 && vote1 && !vote2))

  async function onVote() {
    if (!canVote) return
    setBusy(true)
    setError(null)
    try {
      const raw = (await runAction('vote', { optionId: selected, round: activeRound })) as VoteResult
      if (raw?.error) {
        setError(raw.message || raw.error)
        return
      }
      if (raw.aggregate) setAggregate(raw.aggregate)
      if (raw.aggregateRound2) setAggregateRound2(raw.aggregateRound2)
      if (raw.reveal) setReveal(raw.reveal)
      announcedRef.current = false
      if (activeRound === 2) setSelected('')
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.class_pulse.voteError'))
    } finally {
      setBusy(false)
    }
  }

  const showVoting = !vote1 || (allowSecondVote && vote1 && !vote2 && aggregate)
  const votingRound: 1 | 2 = vote1 && allowSecondVote && !vote2 ? 2 : 1

  return (
    <div className="space-y-4" data-content-tool="class_pulse" data-testid="class-pulse">
      <div className="space-y-1">
        <p className="text-xs font-semibold uppercase tracking-wide text-fg-muted">
          {t('contentTools.tools.class_pulse.label')}
        </p>
        <p id={questionId} className="text-sm font-medium text-fg-default">
          {question}
        </p>
      </div>

      {showVoting && !(votingRound === 2 && readOnly) ? (
        <fieldset disabled={readOnly || busy || (votingRound === 1 && Boolean(vote1))} aria-labelledby={legendId}>
          <legend id={legendId} className="mb-2 text-xs font-semibold uppercase tracking-wide text-fg-muted">
            {votingRound === 2
              ? t('contentTools.tools.class_pulse.revotePrompt')
              : t('contentTools.tools.class_pulse.chooseOption')}
          </legend>
          {votingRound === 2 ? (
            <p className="mb-2 text-sm text-fg-muted">
              {t('contentTools.tools.class_pulse.discussThenRevote')}
            </p>
          ) : null}
          <div className="space-y-2" role="radiogroup" aria-labelledby={legendId}>
            {options.map((o) => (
              <label
                key={`${votingRound}-${o.id}`}
                className="flex cursor-pointer items-start gap-2 rounded border border-border-default px-3 py-2 text-sm dark:border-border-default"
              >
                <input
                  type="radio"
                  name={`${questionId}-r${votingRound}`}
                  value={o.id}
                  checked={selected === o.id}
                  disabled={readOnly || busy || (votingRound === 1 && Boolean(vote1))}
                  onChange={() => {
                    setSelected(o.id)
                    persistDraft(o.id, votingRound)
                  }}
                  className="mt-0.5"
                />
                <span>{o.text}</span>
              </label>
            ))}
          </div>
          {!readOnly && ((votingRound === 1 && !vote1) || (votingRound === 2 && !vote2)) ? (
            <button
              type="button"
              data-testid="class-pulse-submit"
              className="mt-3 rounded bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
              disabled={!canVote}
              onClick={() => void onVote()}
            >
              {busy
                ? t('contentTools.tools.class_pulse.submitting')
                : votingRound === 2
                  ? t('contentTools.tools.class_pulse.submitRevote')
                  : t('contentTools.tools.class_pulse.submitVote')}
            </button>
          ) : null}
        </fieldset>
      ) : null}

      {vote1 && aggregate ? (
        <div className={vote2 && aggregateRound2 ? 'grid gap-4 lg:grid-cols-2' : undefined}>
          <DistributionView
            title={
              allowSecondVote
                ? t('contentTools.tools.class_pulse.round1Results')
                : t('contentTools.tools.class_pulse.results')
            }
            aggregate={aggregate}
            options={options}
            yourOptionId={vote1.optionId}
            showPercentages={showPercentages}
            t={t}
            testId="class-pulse-aggregate"
          />
          {vote2 && aggregateRound2 ? (
            <DistributionView
              title={t('contentTools.tools.class_pulse.round2Results')}
              aggregate={aggregateRound2}
              options={options}
              yourOptionId={vote2.optionId}
              showPercentages={showPercentages}
              t={t}
              testId="class-pulse-aggregate-r2"
            />
          ) : null}
        </div>
      ) : null}

      {reveal?.correctOptionId ? (
        <div
          className="rounded border border-border-default p-3 text-sm dark:border-border-default"
          data-testid="class-pulse-reveal"
        >
          <p className="font-medium text-fg-default">
            {t('contentTools.tools.class_pulse.correctAnswer', {
              answer: optionLabel(options, reveal.correctOptionId),
            })}
          </p>
          {reveal.explanation ? (
            <p className="mt-1 whitespace-pre-wrap text-fg-default">
              {reveal.explanation}
            </p>
          ) : null}
        </div>
      ) : null}

      {error ? (
        <p className="text-sm text-rose-600" role="alert" data-testid="class-pulse-error">
          {error}
        </p>
      ) : null}

      <div className="sr-only" aria-live="polite">
        {liveMsg}
      </div>
    </div>
  )
}
