import { useEffect, useId, useRef, useState } from 'react'
import type { AnswerResult, Checkpoint, CheckpointQuestion } from './types'

type Props = {
  checkpoint: Checkpoint
  index: number
  total: number
  disabled?: boolean
  busy?: boolean
  lastResult?: AnswerResult | null
  onSubmit: (value: string | string[] | number) => void
  onContinue: () => void
  t: (key: string, options?: Record<string, unknown>) => string
}

export function QuestionCard({
  checkpoint,
  index,
  total,
  disabled,
  busy,
  lastResult,
  onSubmit,
  onContinue,
  t,
}: Props) {
  const headingId = useId()
  const rootRef = useRef<HTMLDivElement | null>(null)
  const q = checkpoint.question
  const [value, setValue] = useState<string | string[] | number>('')
  const [multi, setMulti] = useState<string[]>([])

  useEffect(() => {
    setValue('')
    setMulti([])
    rootRef.current?.focus()
  }, [checkpoint.id])

  const answered = Boolean(lastResult && !lastResult.error)
  const canContinue = answered && (lastResult?.done || lastResult?.correct)

  function submit() {
    if (disabled || busy) return
    if (q.type === 'multi') {
      if (multi.length === 0) return
      onSubmit(multi)
      return
    }
    if (value === '' || value === undefined || value === null) return
    onSubmit(value)
  }

  return (
    <div
      ref={rootRef}
      tabIndex={-1}
      className="rounded-lg border border-border-default bg-surface-raised p-4 shadow-sm outline-none focus-visible:ring-2 focus-visible:ring-sky-500 dark:border-border-default dark:bg-surface-raised"
      data-testid="media-checkpoint-question"
      aria-labelledby={headingId}
    >
      <p id={headingId} className="text-sm font-semibold text-fg-default">
        {t('contentTools.tools.media_checkpoints.questionHeading', {
          n: index + 1,
          total,
          time: formatShort(checkpoint.atSec),
        })}
      </p>
      <p className="mt-2 text-sm text-fg-default">{q.prompt}</p>

      <div className="mt-3 space-y-2">{renderInput(q, value, setValue, multi, setMulti, disabled || busy || canContinue)}</div>

      {lastResult?.feedback ? (
        <p className="mt-3 text-sm text-fg-muted" role="status">
          {lastResult.feedback}
        </p>
      ) : null}
      {lastResult?.error ? (
        <p className="mt-3 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {lastResult.message || lastResult.error}
        </p>
      ) : null}
      {answered && lastResult?.correct != null ? (
        <p className="mt-2 text-sm font-medium" role="status">
          {lastResult.correct
            ? t('contentTools.tools.media_checkpoints.correct')
            : t('contentTools.tools.media_checkpoints.incorrect')}
        </p>
      ) : null}

      <div className="mt-4 flex flex-wrap gap-2">
        {!canContinue ? (
          <button
            type="button"
            disabled={disabled || busy}
            onClick={submit}
            className="min-h-11 rounded-md bg-slate-900 px-3 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
          >
            {t('contentTools.tools.media_checkpoints.submit')}
          </button>
        ) : (
          <button
            type="button"
            onClick={onContinue}
            className="min-h-11 rounded-md bg-sky-700 px-3 py-2 text-sm font-medium text-white dark:bg-sky-500"
            data-testid="media-checkpoint-continue"
          >
            {t('contentTools.tools.media_checkpoints.continue')}
          </button>
        )}
        {typeof lastResult?.attemptsRemaining === 'number' ? (
          <span className="self-center text-xs text-fg-muted">
            {t('contentTools.tools.media_checkpoints.attemptsLeft', {
              count: lastResult.attemptsRemaining,
            })}
          </span>
        ) : null}
      </div>
    </div>
  )
}

function formatShort(sec: number): string {
  const s = Math.max(0, Math.floor(sec))
  const m = Math.floor(s / 60)
  const r = s % 60
  return `${m}:${r.toString().padStart(2, '0')}`
}

function renderInput(
  q: CheckpointQuestion,
  value: string | string[] | number,
  setValue: (v: string | string[] | number) => void,
  multi: string[],
  setMulti: (v: string[]) => void,
  disabled?: boolean,
) {
  if (q.type === 'single' || q.type === 'true_false') {
    const opts = q.options ?? []
    return (
      <fieldset className="space-y-2" disabled={disabled}>
        <legend className="sr-only">{q.prompt}</legend>
        {opts.map((o) => (
          <label key={o.id} className="flex min-h-11 items-center gap-2 text-sm">
            <input
              type="radio"
              name="mc-option"
              value={o.id}
              checked={value === o.id}
              onChange={() => setValue(o.id)}
            />
            <span>{o.text}</span>
          </label>
        ))}
      </fieldset>
    )
  }
  if (q.type === 'multi') {
    const opts = q.options ?? []
    return (
      <fieldset className="space-y-2" disabled={disabled}>
        <legend className="sr-only">{q.prompt}</legend>
        {opts.map((o) => {
          const checked = multi.includes(o.id)
          return (
            <label key={o.id} className="flex min-h-11 items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={checked}
                onChange={() => {
                  setMulti(
                    checked ? multi.filter((id) => id !== o.id) : [...multi, o.id],
                  )
                }}
              />
              <span>{o.text}</span>
            </label>
          )
        })}
      </fieldset>
    )
  }
  if (q.type === 'numeric') {
    return (
      <input
        type="text"
        inputMode="decimal"
        disabled={disabled}
        value={String(value ?? '')}
        onChange={(e) => setValue(e.target.value)}
        className="w-full min-h-11 rounded-md border border-border-default px-3 py-2 text-sm dark:border-border-default dark:bg-surface-base"
      />
    )
  }
  return (
    <input
      type="text"
      disabled={disabled}
      value={String(value ?? '')}
      onChange={(e) => setValue(e.target.value)}
      className="w-full min-h-11 rounded-md border border-border-default px-3 py-2 text-sm dark:border-border-default dark:bg-surface-base"
    />
  )
}
