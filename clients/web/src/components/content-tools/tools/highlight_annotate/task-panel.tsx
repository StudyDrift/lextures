import type { ReactNode } from 'react'
import type { Tag } from './types'

/** Light inline markdown for instructor prompts (**bold**, *italic*). */
export function PromptText({ text }: { text: string }) {
  const nodes: ReactNode[] = []
  const re = /(\*\*[^*]+\*\*|\*[^*]+\*)/g
  let last = 0
  let m: RegExpExecArray | null
  let key = 0
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) nodes.push(text.slice(last, m.index))
    const token = m[0]
    if (token.startsWith('**')) {
      nodes.push(
        <strong key={key++} className="font-semibold">
          {token.slice(2, -2)}
        </strong>,
      )
    } else {
      nodes.push(
        <em key={key++} className="italic">
          {token.slice(1, -1)}
        </em>,
      )
    }
    last = m.index + token.length
  }
  if (last < text.length) nodes.push(text.slice(last))
  return <>{nodes}</>
}

export type TaskPanelProps = {
  promptId: string
  prompt: string
  tags: Tag[]
  countsByTag: Map<string, number>
  progress: number
  minAnnotations: number
  activeCount: number
  remaining: number
  complete: boolean
  progressPct: number
  t: (key: string, options?: Record<string, unknown>) => string
}

/** Prompt, progress bar, and available labels for the annotation task. */
export function TaskPanel({
  promptId,
  prompt,
  tags,
  countsByTag,
  progress,
  minAnnotations,
  activeCount,
  remaining,
  complete,
  progressPct,
  t,
}: TaskPanelProps) {
  return (
    <div
      className="rounded-xl border border-border-default bg-gradient-to-b from-slate-50 to-white p-4 dark:border-border-default dark:from-neutral-900/80 dark:to-neutral-950"
      data-testid="ha-prompt"
    >
      <p className="text-xs font-semibold uppercase tracking-wide text-accent-fg">
        {t('contentTools.tools.highlight_annotate.yourTask')}
      </p>
      <p
        id={promptId}
        className="mt-1 text-sm leading-relaxed text-fg-default sm:text-[15px]"
      >
        <PromptText text={prompt} />
      </p>

      <div className="mt-3 space-y-1.5">
        <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
          <p
            data-testid="ha-progress"
            className={
              complete
                ? 'font-medium text-emerald-700 dark:text-emerald-400'
                : 'font-medium text-fg-default'
            }
          >
            {complete
              ? t('contentTools.tools.highlight_annotate.progressComplete', {
                  done: activeCount,
                  required: minAnnotations,
                })
              : t('contentTools.tools.highlight_annotate.progress', {
                  done: progress,
                  required: minAnnotations,
                })}
          </p>
          {!complete && remaining > 0 ? (
            <span className="text-fg-muted">
              {t('contentTools.tools.highlight_annotate.remaining', { count: remaining })}
            </span>
          ) : null}
        </div>
        <div
          className="h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-surface-overlay"
          role="progressbar"
          aria-valuenow={progress}
          aria-valuemin={0}
          aria-valuemax={minAnnotations}
          aria-label={t('contentTools.tools.highlight_annotate.progress', {
            done: progress,
            required: minAnnotations,
          })}
        >
          <div
            className={[
              'h-full rounded-full transition-all duration-300',
              complete ? 'bg-emerald-500' : 'bg-indigo-500',
            ].join(' ')}
            style={{ width: `${progressPct}%` }}
          />
        </div>
      </div>

      <div className="mt-3">
        <p className="mb-1.5 text-xs font-medium text-fg-muted">
          {t('contentTools.tools.highlight_annotate.labelsTitle')}
        </p>
        <ul
          className="flex flex-wrap gap-2"
          aria-label={t('contentTools.tools.highlight_annotate.tagLegend')}
        >
          {tags.map((tag) => (
            <li
              key={tag.id}
              className="inline-flex items-center gap-1.5 rounded-full border border-border-default bg-surface-raised px-2.5 py-1 text-xs shadow-sm dark:border-border-default dark:bg-surface-raised"
            >
              <span
                aria-hidden
                className="inline-block h-2.5 w-2.5 shrink-0 rounded-full"
                style={{ backgroundColor: tag.color }}
              />
              <span className="font-medium text-fg-default">{tag.label}</span>
              {tag.description ? (
                <span className="text-fg-muted">— {tag.description}</span>
              ) : null}
              {(countsByTag.get(tag.id) ?? 0) > 0 ? (
                <span className="rounded-full bg-surface-sunken px-1.5 py-px text-[10px] font-semibold text-fg-muted dark:bg-surface-overlay dark:text-fg-muted">
                  {countsByTag.get(tag.id)}
                </span>
              ) : null}
            </li>
          ))}
        </ul>
      </div>
    </div>
  )
}

export type HowToCoachProps = {
  unitWord: string
  requireNote: boolean
  t: (key: string, options?: Record<string, unknown>) => string
}

/** Three-step coaching cards shown while the task is incomplete. */
export function HowToCoach({ unitWord, requireNote, t }: HowToCoachProps) {
  return (
    <ol
      className="grid gap-2 sm:grid-cols-3"
      data-testid="ha-empty-hint"
      aria-label={t('contentTools.tools.highlight_annotate.howToTitle')}
    >
      <li className="flex gap-2 rounded-lg border border-indigo-100 bg-indigo-50/70 px-3 py-2 text-xs text-indigo-950 dark:border-indigo-900/50 dark:bg-indigo-950/30 dark:text-indigo-100">
        <span
          className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-accent-solid text-[11px] font-bold text-white"
          aria-hidden
        >
          1
        </span>
        <span>
          {t('contentTools.tools.highlight_annotate.howTo.step1', { unit: unitWord })}
        </span>
      </li>
      <li className="flex gap-2 rounded-lg border border-indigo-100 bg-indigo-50/70 px-3 py-2 text-xs text-indigo-950 dark:border-indigo-900/50 dark:bg-indigo-950/30 dark:text-indigo-100">
        <span
          className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-accent-solid text-[11px] font-bold text-white"
          aria-hidden
        >
          2
        </span>
        <span>{t('contentTools.tools.highlight_annotate.howTo.step2')}</span>
      </li>
      <li className="flex gap-2 rounded-lg border border-indigo-100 bg-indigo-50/70 px-3 py-2 text-xs text-indigo-950 dark:border-indigo-900/50 dark:bg-indigo-950/30 dark:text-indigo-100">
        <span
          className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-accent-solid text-[11px] font-bold text-white"
          aria-hidden
        >
          3
        </span>
        <span>
          {requireNote
            ? t('contentTools.tools.highlight_annotate.howTo.step3Required')
            : t('contentTools.tools.highlight_annotate.howTo.step3Optional')}
        </span>
      </li>
    </ol>
  )
}
