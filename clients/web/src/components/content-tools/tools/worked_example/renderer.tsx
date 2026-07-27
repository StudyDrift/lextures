import { useEffect, useId, useRef, useState, type Ref } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'
import { MathPreview } from './math-preview'
import {
  prefersReducedMotion,
  type CheckResult,
  type HintResult,
  type RevealResult,
  type Step,
  type StepProgress,
} from './types'

function asSteps(config: Record<string, unknown>): Step[] {
  return Array.isArray(config.steps) ? (config.steps as Step[]) : []
}

function asProgress(state: Record<string, unknown>): Record<string, StepProgress> {
  if (state.steps && typeof state.steps === 'object') {
    return state.steps as Record<string, StepProgress>
  }
  return {}
}

function stepDone(sp: StepProgress | undefined): boolean {
  if (!sp) return false
  if (sp.revealed || sp.completedAt) return true
  const last = sp.attempts?.[sp.attempts.length - 1]
  return last?.result === 'correct' || last?.result === 'needs_review'
}

export default function WorkedExampleRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const titleId = useId()
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement | null>(null)
  const steps = asSteps(config)
  const problem = typeof config.problem === 'string' ? config.problem : ''
  const title =
    typeof config.title === 'string' && config.title.trim()
      ? config.title
      : t('contentTools.tools.worked_example.defaultTitle')
  const showAllSteps = config.showAllSteps === true
  const allowRevealAll = config.allowRevealAll === true
  const attemptsPerStep =
    typeof config.attemptsPerStep === 'number' && config.attemptsPerStep > 0
      ? config.attemptsPerStep
      : 3

  const progress = asProgress(state)
  const blankedIds = Array.isArray(state.blankedStepIds)
    ? (state.blankedStepIds as string[])
    : steps.filter((s) => s.blank).map((s) => s.id)
  const blankedSet = new Set(blankedIds)

  const [localDrafts, setLocalDrafts] = useState<Record<string, string>>(() => {
    const out: Record<string, string> = {}
    for (const [id, sp] of Object.entries(progress)) {
      if (sp.draft) out[id] = sp.draft
    }
    return out
  })
  const [busy, setBusy] = useState(false)
  const [prepared, setPrepared] = useState(blankedIds.length > 0 && Array.isArray(state.blankedStepIds))
  const [lastCheck, setLastCheck] = useState<Record<string, CheckResult>>({})
  const [hints, setHints] = useState<Record<string, string[]>>({})
  const [reveals, setReveals] = useState<Record<string, RevealResult>>({})
  const [error, setError] = useState<string | null>(null)

  const currentId =
    (typeof state.currentStepId === 'string' && state.currentStepId) ||
    blankedIds.find((id) => !stepDone(progress[id])) ||
    blankedIds[0] ||
    steps[0]?.id ||
    ''

  useEffect(() => {
    if (prepared || readOnly) return
    let cancelled = false
    setBusy(true)
    void runAction('prepare', {})
      .then((raw) => {
        if (cancelled) return
        const r = raw as { blankedStepIds?: string[]; currentStepId?: string }
        setPrepared(true)
        if (r.currentStepId) {
          // state refresh comes from host; keep local awareness
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : t('contentTools.tools.worked_example.error'))
          setPrepared(true)
        }
      })
      .finally(() => {
        if (!cancelled) setBusy(false)
      })
    return () => {
      cancelled = true
    }
  }, [prepared, readOnly, runAction, t])

  useEffect(() => {
    if (!currentId || readOnly) return
    if (prefersReducedMotion()) return
    const el = inputRef.current
    if (el && typeof el.focus === 'function') {
      el.focus()
    }
  }, [currentId, readOnly])

  function isVisible(step: Step, index: number): boolean {
    if (showAllSteps) return true
    if (!blankedSet.has(step.id)) return true
    // Show completed blanked steps and the current one; dim/hide future blanked.
    for (let i = 0; i < index; i++) {
      const prev = steps[i]
      if (!blankedSet.has(prev.id)) continue
      if (!stepDone(progress[prev.id])) return false
    }
    return true
  }

  function updateDraft(stepId: string, value: string) {
    setLocalDrafts((prev) => ({ ...prev, [stepId]: value }))
    const nextSteps: Record<string, StepProgress> = { ...progress }
    nextSteps[stepId] = { ...(nextSteps[stepId] ?? {}), draft: value }
    save({ steps: nextSteps, v: 1 })
  }

  async function onCheck(stepId: string) {
    if (busy || readOnly) return
    setBusy(true)
    setError(null)
    try {
      const value = localDrafts[stepId] ?? progress[stepId]?.draft ?? ''
      const raw = (await runAction('check_step', { stepId, value })) as CheckResult
      setLastCheck((prev) => ({ ...prev, [stepId]: raw }))
      if (raw.error) {
        setError(raw.message || raw.error)
        announce(raw.message || raw.error)
      } else if (raw.result === 'correct') {
        announce(t('contentTools.tools.worked_example.announce.correct'))
      } else if (raw.result === 'needs_review') {
        announce(t('contentTools.tools.worked_example.announce.needsReview'))
      } else {
        announce(t('contentTools.tools.worked_example.announce.incorrect'))
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.worked_example.error'))
    } finally {
      setBusy(false)
    }
  }

  async function onHint(stepId: string) {
    if (busy || readOnly) return
    setBusy(true)
    setError(null)
    try {
      const raw = (await runAction('hint', { stepId })) as HintResult
      if (raw.error) {
        setError(raw.error)
      } else if (raw.hint) {
        setHints((prev) => ({
          ...prev,
          [stepId]: [...(prev[stepId] ?? []), raw.hint!],
        }))
        announce(t('contentTools.tools.worked_example.announce.hint'))
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.worked_example.error'))
    } finally {
      setBusy(false)
    }
  }

  async function onReveal(stepId: string) {
    if (busy || readOnly) return
    setBusy(true)
    setError(null)
    try {
      const raw = (await runAction('reveal_step', { stepId })) as RevealResult
      setReveals((prev) => ({ ...prev, [stepId]: raw }))
      if (raw.error) {
        setError(raw.message || raw.error)
      } else {
        announce(t('contentTools.tools.worked_example.announce.revealed'))
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.worked_example.error'))
    } finally {
      setBusy(false)
    }
  }

  async function onRevealAll() {
    if (busy || readOnly || !allowRevealAll) return
    setBusy(true)
    try {
      await runAction('reveal_all', {})
      announce(t('contentTools.tools.worked_example.announce.revealAll'))
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : t('contentTools.tools.worked_example.error'))
    } finally {
      setBusy(false)
    }
  }

  const blankedTotal = blankedIds.length || steps.filter((s) => s.blank).length
  const blankedDone = blankedIds.filter((id) => stepDone(progress[id])).length
  const hintsUsedTotal = Object.values(progress).reduce((n, sp) => n + (sp.hintsUsed ?? 0), 0)

  return (
    <div
      className="space-y-4"
      data-content-tool="worked_example"
      data-testid="worked-example"
      aria-labelledby={titleId}
    >
      <div className="sticky top-0 z-[1] space-y-1 border-b border-[color:var(--lex-border)] bg-[color:var(--lex-surface,#fff)] pb-3">
        <h3 id={titleId} className="text-base font-semibold text-[color:var(--lex-fg)]">
          {title}
        </h3>
        <p className="whitespace-pre-wrap text-sm text-[color:var(--lex-fg)]">{problem}</p>
      </div>

      <ol className="space-y-4">
        {steps.map((step, index) => {
          const visible = isVisible(step, index)
          if (!visible) {
            return (
              <li
                key={step.id}
                className="rounded-md border border-dashed border-[color:var(--lex-border)] px-3 py-2 text-sm text-[color:var(--lex-muted)]"
                aria-hidden="true"
              >
                {t('contentTools.tools.worked_example.stepLocked', { n: index + 1 })}
              </li>
            )
          }

          const isBlanked = blankedSet.has(step.id)
          const sp = progress[step.id]
          const done = stepDone(sp)
          const isCurrent = isBlanked && !done && step.id === currentId
          const used = sp?.attempts?.length ?? 0
          const remaining = Math.max(0, attemptsPerStep - used)
          const check = lastCheck[step.id]
          const reveal = reveals[step.id]
          const lastAttempt = sp?.attempts?.[sp.attempts.length - 1]
          const canReveal =
            isBlanked &&
            !done &&
            (remaining === 0 || check?.canReveal === true) &&
            !readOnly

          const groupLabel =
            step.label?.trim() ||
            t('contentTools.tools.worked_example.stepN', { n: index + 1 })

          return (
            <li key={step.id}>
              <fieldset
                className={`space-y-2 rounded-md border px-3 py-3 ${
                  isCurrent
                    ? 'border-[color:var(--lex-accent,#2563eb)]'
                    : 'border-[color:var(--lex-border)]'
                }`}
                disabled={readOnly || (isBlanked && !isCurrent && !done)}
              >
                <legend className="px-1 text-sm font-medium text-[color:var(--lex-fg)]">
                  {groupLabel}
                </legend>
                <p className="whitespace-pre-wrap text-sm text-[color:var(--lex-fg)]">{step.text}</p>

                {isBlanked && done && (
                  <div className="space-y-1 text-sm" data-testid={`worked-example-done-${step.id}`}>
                    {lastAttempt && (
                      <p>
                        {t('contentTools.tools.worked_example.yourAnswer')}:{' '}
                        <code>{lastAttempt.value}</code>
                        {lastAttempt.result === 'correct' && (
                          <span className="ms-2 text-green-700 dark:text-green-400">
                            {t('contentTools.tools.worked_example.correct')}
                          </span>
                        )}
                        {lastAttempt.result === 'needs_review' && (
                          <span className="ms-2 text-amber-700 dark:text-amber-400">
                            {t('contentTools.tools.worked_example.needsReview')}
                          </span>
                        )}
                        {sp?.revealed && (
                          <span className="ms-2 text-[color:var(--lex-muted)]">
                            {t('contentTools.tools.worked_example.revealed')}
                          </span>
                        )}
                      </p>
                    )}
                    {(reveal?.expectedDisplay || sp?.revealed) && reveal?.expectedDisplay && (
                      <p>
                        {t('contentTools.tools.worked_example.expected')}:{' '}
                        <code>{reveal.expectedDisplay}</code>
                      </p>
                    )}
                    {reveal?.explanation && (
                      <p className="text-[color:var(--lex-muted)]">{reveal.explanation}</p>
                    )}
                  </div>
                )}

                {isBlanked && isCurrent && (
                  <div className="space-y-2">
                    {step.blank?.type === 'choice' ? (
                      <div
                        role="radiogroup"
                        aria-label={groupLabel}
                        className="flex flex-col gap-2"
                      >
                        {(step.blank.options ?? []).map((opt) => (
                          <label key={opt.id} className="flex items-center gap-2 text-sm">
                            <input
                              type="radio"
                              name={`we-${step.id}`}
                              value={opt.id}
                              checked={(localDrafts[step.id] ?? '') === opt.id}
                              onChange={() => updateDraft(step.id, opt.id)}
                              disabled={readOnly || busy}
                            />
                            {opt.text}
                          </label>
                        ))}
                      </div>
                    ) : (
                      <>
                        <label className="block text-sm" htmlFor={`we-input-${step.id}`}>
                          {t('contentTools.tools.worked_example.yourAnswer')}
                          {step.blank?.unit ? ` (${step.blank.unit})` : ''}
                        </label>
                        <input
                          id={`we-input-${step.id}`}
                          ref={
                            step.id === currentId
                              ? (inputRef as Ref<HTMLInputElement>)
                              : undefined
                          }
                          className="w-full rounded-md border border-[color:var(--lex-border)] bg-transparent px-3 py-2 text-sm"
                          value={localDrafts[step.id] ?? ''}
                          onChange={(e) => updateDraft(step.id, e.target.value)}
                          disabled={readOnly || busy}
                          autoComplete="off"
                          data-testid={`worked-example-input-${step.id}`}
                        />
                        {step.blank?.type === 'expression' && (
                          <MathPreview
                            source={localDrafts[step.id] ?? ''}
                            label={t('contentTools.tools.worked_example.mathPreviewEmpty')}
                          />
                        )}
                      </>
                    )}

                    <div className="flex flex-wrap gap-2">
                      <button
                        type="button"
                        className="rounded-md bg-[color:var(--lex-accent,#2563eb)] px-3 py-1.5 text-sm text-white disabled:opacity-50"
                        disabled={readOnly || busy}
                        onClick={() => void onCheck(step.id)}
                        data-testid={`worked-example-check-${step.id}`}
                      >
                        {t('contentTools.tools.worked_example.check')}
                      </button>
                      <button
                        type="button"
                        className="rounded-md border border-[color:var(--lex-border)] px-3 py-1.5 text-sm disabled:opacity-50"
                        disabled={readOnly || busy}
                        onClick={() => void onHint(step.id)}
                        data-testid={`worked-example-hint-${step.id}`}
                      >
                        {t('contentTools.tools.worked_example.hint')}
                      </button>
                      {canReveal && (
                        <button
                          type="button"
                          className="rounded-md border border-[color:var(--lex-border)] px-3 py-1.5 text-sm disabled:opacity-50"
                          disabled={readOnly || busy}
                          onClick={() => void onReveal(step.id)}
                          data-testid={`worked-example-reveal-${step.id}`}
                        >
                          {t('contentTools.tools.worked_example.showStep')}
                        </button>
                      )}
                    </div>

                    {check && !check.error && (
                      <p
                        className="text-sm"
                        role="status"
                        data-testid={`worked-example-result-${step.id}`}
                      >
                        {check.result === 'correct'
                          ? t('contentTools.tools.worked_example.correct')
                          : check.result === 'needs_review'
                            ? t('contentTools.tools.worked_example.needsReview')
                            : t('contentTools.tools.worked_example.incorrect')}
                        {typeof check.attemptsRemaining === 'number' &&
                          check.result === 'incorrect' && (
                            <span className="ms-2 text-[color:var(--lex-muted)]">
                              {t('contentTools.tools.worked_example.attemptsLeft', {
                                count: check.attemptsRemaining,
                              })}
                            </span>
                          )}
                        {check.feedback && (
                          <span className="ms-2 text-[color:var(--lex-muted)]">{check.feedback}</span>
                        )}
                      </p>
                    )}

                    {(hints[step.id] ?? []).map((h, i) => (
                      <p key={i} className="text-sm text-[color:var(--lex-muted)]" role="status">
                        {t('contentTools.tools.worked_example.hintLabel', { n: i + 1 })}: {h}
                      </p>
                    ))}
                  </div>
                )}
              </fieldset>
            </li>
          )
        })}
      </ol>

      <footer className="flex flex-wrap items-center justify-between gap-2 border-t border-[color:var(--lex-border)] pt-3 text-sm text-[color:var(--lex-muted)]">
        <span data-testid="worked-example-progress">
          {t('contentTools.tools.worked_example.progress', {
            done: blankedDone,
            total: blankedTotal,
          })}
          {hintsUsedTotal > 0 &&
            ` · ${t('contentTools.tools.worked_example.hintsUsed', { count: hintsUsedTotal })}`}
        </span>
        {allowRevealAll && !readOnly && (
          <button
            type="button"
            className="rounded-md border border-[color:var(--lex-border)] px-3 py-1.5 text-sm"
            disabled={busy}
            onClick={() => void onRevealAll()}
            data-testid="worked-example-reveal-all"
          >
            {t('contentTools.tools.worked_example.revealAll')}
          </button>
        )}
      </footer>

      {error && (
        <p className="text-sm text-red-700 dark:text-red-400" role="alert">
          {error}
        </p>
      )}
    </div>
  )
}
