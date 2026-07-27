import { useEffect, useId, useRef, useState } from 'react'
import { evalPredicate } from '../../../../lib/safe-expression'
import type { ContentToolRendererProps } from '../../host/runtime-contract'
import { DataTable } from './data-table'
import {
  appendTrace,
  computeSeries,
  defaultParams,
  paramsAsFloats,
  resolveSweep,
  scalarReadouts,
  trendSummary,
} from './model'
import { ParameterControls } from './parameter-controls'
import { PlotSvg } from './plot-svg'
import {
  ANNOUNCE_THROTTLE_MS,
  type NoticingPrompt,
  type Parameter,
  type ParameterExplorerConfig,
  type ParameterExplorerState,
} from './types'

function parseConfig(raw: Record<string, unknown>): ParameterExplorerConfig {
  return {
    prompt: typeof raw.prompt === 'string' ? raw.prompt : '',
    hint: typeof raw.hint === 'string' ? raw.hint : undefined,
    parameters: Array.isArray(raw.parameters) ? (raw.parameters as Parameter[]) : [],
    model: (raw.model && typeof raw.model === 'object'
      ? raw.model
      : { kind: 'preset', preset: 'quadratic', bind: { a: 'a', b: 'b', c: 'c' } }) as ParameterExplorerConfig['model'],
    outputs: Array.isArray(raw.outputs)
      ? (raw.outputs as ParameterExplorerConfig['outputs'])
      : [{ kind: 'plot', label: 'Plot' }, { kind: 'table', label: 'Table' }],
    noticingPrompts: Array.isArray(raw.noticingPrompts)
      ? (raw.noticingPrompts as NoticingPrompt[])
      : [],
    requireAllCheckpoints: Boolean(raw.requireAllCheckpoints),
  }
}

function parseState(
  raw: Record<string, unknown>,
  parameters: Parameter[],
): ParameterExplorerState {
  const defaults = defaultParams(parameters)
  const params =
    raw.params && typeof raw.params === 'object'
      ? { ...defaults, ...(raw.params as Record<string, number | boolean | string>) }
      : defaults
  return {
    v: 1,
    params,
    trace: Array.isArray(raw.trace)
      ? (raw.trace as ParameterExplorerState['trace'])
      : [],
    checkpoints:
      raw.checkpoints && typeof raw.checkpoints === 'object'
        ? (raw.checkpoints as Record<string, string>)
        : {},
    answers:
      raw.answers && typeof raw.answers === 'object'
        ? (raw.answers as Record<string, string>)
        : {},
    completedAt: typeof raw.completedAt === 'string' ? raw.completedAt : undefined,
  }
}

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export default function ParameterExplorerRenderer({
  config: rawConfig,
  state: rawState,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const cfg = parseConfig(rawConfig)
  const initial = parseState(rawState, cfg.parameters)
  const [params, setParams] = useState(initial.params)
  const [trace, setTrace] = useState(initial.trace)
  const [checkpoints, setCheckpoints] = useState(initial.checkpoints)
  const [answers, setAnswers] = useState(initial.answers)
  const [completedAt, setCompletedAt] = useState(initial.completedAt)
  const [view, setView] = useState<'plot' | 'table'>('plot')
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [reducedMotion, setReducedMotion] = useState(false)
  const lastAnnounce = useRef(0)
  const draftTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const baseId = useId()

  useEffect(() => {
    setReducedMotion(prefersReducedMotion())
  }, [])

  // Sync from server state after reset / external patch
  useEffect(() => {
    const next = parseState(rawState, cfg.parameters)
    setParams(next.params)
    setTrace(next.trace)
    setCheckpoints(next.checkpoints)
    setAnswers(next.answers)
    setCompletedAt(next.completedAt)
    // eslint-disable-next-line react-hooks/exhaustive-deps -- only when server state identity changes
  }, [rawState])

  const series = computeSeries(cfg.model, params)
  const sweep = resolveSweep(cfg.model)
  const summary = trendSummary(series)
  const readouts = scalarReadouts(cfg.model, params)
  const prompts = cfg.noticingPrompts ?? []
  const answeredCount = prompts.filter((p) => (answers[p.id] ?? '').trim() !== '').length

  function scheduleSave(next: {
    params: typeof params
    trace: typeof trace
    checkpoints: typeof checkpoints
    answers: typeof answers
    completedAt?: string
  }) {
    if (readOnly) return
    if (draftTimer.current) clearTimeout(draftTimer.current)
    draftTimer.current = setTimeout(() => {
      void Promise.resolve(
        save({
          v: 1,
          params: next.params,
          trace: next.trace,
          checkpoints: next.checkpoints,
          answers: next.answers,
          ...(next.completedAt ? { completedAt: next.completedAt } : {}),
        }),
      ).catch(() => {
        // failed save must not block manipulation (NFR)
      })
    }, 400)
  }

  function updateParam(id: string, value: number | boolean | string) {
    if (readOnly) return
    const nextParams = { ...params, [id]: value }
    const nextTrace = appendTrace(trace, nextParams)
    setParams(nextParams)
    setTrace(nextTrace)
    scheduleSave({
      params: nextParams,
      trace: nextTrace,
      checkpoints,
      answers,
      completedAt,
    })

    const now = Date.now()
    if (now - lastAnnounce.current >= ANNOUNCE_THROTTLE_MS) {
      lastAnnounce.current = now
      const p = cfg.parameters.find((x) => x.id === id)
      const unit = p && p.kind === 'number' ? p.unit : undefined
      announce(
        t('contentTools.tools.parameter_explorer.announce.paramChanged', {
          label: p?.label ?? id,
          value: unit ? `${value} ${unit}` : String(value),
        }),
      )
    }

    // Auto-check unlockable prompts client-side, then confirm via action
    for (const prompt of prompts) {
      if (!prompt.unlockWhen || checkpoints[prompt.id]) continue
      try {
        if (evalPredicate(prompt.unlockWhen, paramsAsFloats(nextParams))) {
          void claimCheckpoint(prompt.id, nextParams)
        }
      } catch {
        // ignore bad predicates at runtime
      }
    }
  }

  async function claimCheckpoint(
    promptId: string,
    nextParams: Record<string, number | boolean | string>,
  ) {
    try {
      const result = (await runAction('checkpoint', {
        promptId,
        params: nextParams,
      })) as {
        unlocked?: boolean
        hitAt?: string
        already?: boolean
        error?: string
      }
      if (result.unlocked && result.hitAt) {
        setCheckpoints((prev) => {
          const next = { ...prev, [promptId]: result.hitAt! }
          scheduleSave({
            params: nextParams,
            trace,
            checkpoints: next,
            answers,
            completedAt,
          })
          return next
        })
        announce(t('contentTools.tools.parameter_explorer.announce.checkpoint', { id: promptId }))
      }
    } catch {
      // non-blocking
    }
  }

  async function submitAnswer(prompt: NoticingPrompt, answer: string) {
    if (readOnly || busy) return
    setBusy(true)
    setError(null)
    try {
      const result = (await runAction('submit_answer', {
        promptId: prompt.id,
        answer,
        params,
      })) as {
        ok?: boolean
        completed?: boolean
        error?: string
        message?: string
        preserveInput?: boolean
      }
      if (result.error) {
        setError(result.message || result.error)
        if (!result.preserveInput) {
          setAnswers((prev) => ({ ...prev, [prompt.id]: answer }))
        }
        return
      }
      setAnswers((prev) => {
        const next = { ...prev, [prompt.id]: answer }
        const doneAt = result.completed ? new Date().toISOString() : completedAt
        if (result.completed) setCompletedAt(doneAt)
        scheduleSave({
          params,
          trace,
          checkpoints,
          answers: next,
          completedAt: doneAt,
        })
        return next
      })
      if (result.completed) {
        announce(t('contentTools.tools.parameter_explorer.announce.complete'))
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'submit failed')
    } finally {
      setBusy(false)
    }
  }

  async function resetDefaults() {
    if (readOnly || busy) return
    setBusy(true)
    setError(null)
    try {
      await runAction('reset_defaults', {})
      const defaults = defaultParams(cfg.parameters)
      setParams(defaults)
      setTrace([])
      setCheckpoints({})
      setAnswers({})
      setCompletedAt(undefined)
      announce(t('contentTools.tools.parameter_explorer.announce.reset'))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'reset failed')
    } finally {
      setBusy(false)
    }
  }

  const coverageBins = (() => {
    const bins = new Set<string>()
    for (const entry of trace) {
      for (const p of cfg.parameters) {
        if (p.kind !== 'number') continue
        const v = entry.params[p.id]
        if (typeof v !== 'number') continue
        const span = p.max - p.min
        let bucket = 0
        if (span > 0) {
          bucket = Math.min(9, Math.max(0, Math.floor(((v - p.min) / span) * 10)))
        }
        bins.add(`${p.id}:${bucket}`)
      }
    }
    return bins.size
  })()

  return (
    <div
      className="space-y-4 text-slate-900 dark:text-neutral-100"
      data-testid="parameter-explorer"
      data-completed={completedAt ? '1' : '0'}
    >
      <div>
        <p className="text-sm font-medium">{cfg.prompt}</p>
        {cfg.hint ? (
          <p className="mt-1 text-xs text-slate-600 dark:text-neutral-400">{cfg.hint}</p>
        ) : null}
      </div>

      <div className="grid gap-4 md:grid-cols-[minmax(0,14rem)_minmax(0,1fr)]">
        <div className="space-y-3">
          <ParameterControls
            parameters={cfg.parameters}
            values={params}
            onChange={updateParam}
            readOnly={readOnly}
            t={t}
          />
          {!readOnly ? (
            <button
              type="button"
              className="text-xs font-medium text-teal-800 underline dark:text-teal-300"
              data-testid="parameter-explorer-reset"
              onClick={() => void resetDefaults()}
              disabled={busy}
            >
              {t('contentTools.tools.parameter_explorer.resetDefaults')}
            </button>
          ) : null}
          {trace.length > 0 ? (
            <p className="text-xs text-slate-500" data-testid="parameter-explorer-coverage">
              {t('contentTools.tools.parameter_explorer.coverageStrip', {
                visited: trace.length,
                bins: coverageBins,
              })}
            </p>
          ) : null}
        </div>

        <div className="space-y-2">
          <div className="flex gap-2 text-xs" role="tablist" aria-label={t('contentTools.tools.parameter_explorer.outputTabs')}>
            <button
              type="button"
              role="tab"
              aria-selected={view === 'plot'}
              className={`rounded px-2 py-1 ${view === 'plot' ? 'bg-teal-800 text-white' : 'bg-slate-100 dark:bg-neutral-800'}`}
              onClick={() => setView('plot')}
            >
              {t('contentTools.tools.parameter_explorer.plotTab')}
            </button>
            <button
              type="button"
              role="tab"
              aria-selected={view === 'table'}
              className={`rounded px-2 py-1 ${view === 'table' ? 'bg-teal-800 text-white' : 'bg-slate-100 dark:bg-neutral-800'}`}
              data-testid="parameter-explorer-table-tab"
              onClick={() => setView('table')}
            >
              {t('contentTools.tools.parameter_explorer.tableTab')}
            </button>
          </div>

          {view === 'plot' ? (
            <PlotSvg
              points={series}
              xLabel={sweep.xLabel}
              yLabel={sweep.yLabel}
              summary={summary}
              reducedMotion={reducedMotion}
            />
          ) : (
            <DataTable
              points={series}
              xLabel={sweep.xLabel}
              yLabel={sweep.yLabel}
              summary={summary}
              caption={t('contentTools.tools.parameter_explorer.tableCaption')}
            />
          )}

          {readouts.length > 0 ? (
            <ul
              className="flex flex-wrap gap-3 text-xs tabular-nums"
              data-testid="parameter-explorer-readout"
            >
              {readouts.map((r) => (
                <li key={r.label}>
                  <span className="text-slate-500">{r.label}: </span>
                  <strong>{Number.isFinite(r.value) ? Math.round(r.value * 1000) / 1000 : '—'}</strong>
                </li>
              ))}
            </ul>
          ) : null}
          <p className="sr-only" aria-live="polite">
            {summary}
          </p>
        </div>
      </div>

      {prompts.length > 0 ? (
        <div className="space-y-3 border-t border-slate-200 pt-3 dark:border-neutral-700">
          <p className="text-xs text-slate-600 dark:text-neutral-400" data-testid="parameter-explorer-progress">
            {t('contentTools.tools.parameter_explorer.progress', {
              answered: answeredCount,
              total: prompts.length,
            })}
          </p>
          {prompts.map((prompt) => {
            const locked =
              Boolean(prompt.unlockWhen) && !checkpoints[prompt.id]
            return (
              <fieldset
                key={prompt.id}
                disabled={readOnly || locked || busy}
                className={`space-y-2 rounded border p-3 ${locked ? 'opacity-60' : ''}`}
                data-testid={`parameter-explorer-prompt-${prompt.id}`}
                data-locked={locked ? '1' : '0'}
              >
                <legend className="px-1 text-sm font-medium">
                  {locked ? (
                    <span className="mr-1 text-xs font-normal uppercase tracking-wide text-slate-500">
                      [{t('contentTools.tools.parameter_explorer.locked')}]
                    </span>
                  ) : null}
                  {prompt.text}
                </legend>
                {locked ? (
                  <p className="text-xs text-slate-500">
                    {t('contentTools.tools.parameter_explorer.lockedHint')}
                  </p>
                ) : prompt.kind === 'choice' ? (
                  <div className="space-y-1" role="radiogroup" aria-labelledby={`${baseId}-${prompt.id}`}>
                    {(prompt.options ?? []).map((opt) => (
                      <label key={opt.value} className="flex items-center gap-2 text-sm">
                        <input
                          type="radio"
                          name={`${baseId}-${prompt.id}`}
                          value={opt.value}
                          checked={answers[prompt.id] === opt.value}
                          onChange={() => void submitAnswer(prompt, opt.value)}
                        />
                        {opt.label}
                      </label>
                    ))}
                  </div>
                ) : (
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <textarea
                      className="min-h-[4rem] flex-1 rounded border border-slate-300 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
                      defaultValue={answers[prompt.id] ?? ''}
                      id={`${baseId}-ans-${prompt.id}`}
                      onBlur={(e) => {
                        const v = e.target.value.trim()
                        if (v && v !== (answers[prompt.id] ?? '')) {
                          void submitAnswer(prompt, v)
                        }
                      }}
                    />
                    <button
                      type="button"
                      className="rounded bg-teal-800 px-3 py-1.5 text-sm text-white"
                      onClick={() => {
                        const el = document.getElementById(
                          `${baseId}-ans-${prompt.id}`,
                        ) as HTMLTextAreaElement | null
                        void submitAnswer(prompt, (el?.value ?? '').trim())
                      }}
                    >
                      {t('contentTools.tools.parameter_explorer.saveAnswer')}
                    </button>
                  </div>
                )}
              </fieldset>
            )
          })}
        </div>
      ) : null}

      {error ? (
        <p className="text-sm text-red-700 dark:text-red-300" role="alert">
          {error}
        </p>
      ) : null}
      {completedAt ? (
        <p className="text-xs font-medium text-teal-800 dark:text-teal-300" role="status">
          {t('contentTools.tools.parameter_explorer.complete')}
        </p>
      ) : null}
    </div>
  )
}
