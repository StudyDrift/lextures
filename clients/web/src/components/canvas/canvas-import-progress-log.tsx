import { ChevronDown, Loader2 } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type { CanvasImportProgressEntry } from '../../hooks/use-canvas-import-progress-log'

type CanvasImportProgressLogProps = {
  entries: CanvasImportProgressEntry[]
  title?: string
  defaultExpanded?: boolean
  className?: string
  maxHeightClassName?: string
  active?: boolean
}

export function CanvasImportProgressLog({
  entries,
  title = 'Import progress',
  defaultExpanded = true,
  className = '',
  maxHeightClassName = 'max-h-48',
  active = false,
}: CanvasImportProgressLogProps) {
  const [expanded, setExpanded] = useState(defaultExpanded)
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!expanded || entries.length === 0) return
    bottomRef.current?.scrollIntoView({ block: 'nearest' })
  }, [entries, expanded])

  if (entries.length === 0) return null

  return (
    <div
      className={[
        'rounded-xl border border-slate-200 bg-slate-50/80 dark:border-neutral-700 dark:bg-neutral-900/60',
        className,
      ].join(' ')}
    >
      <button
        type="button"
        className="flex w-full items-center gap-2 px-3 py-2.5 text-start text-sm font-medium text-slate-800 dark:text-neutral-100"
        onClick={() => setExpanded((open) => !open)}
        aria-expanded={expanded}
      >
        {active ? (
          <Loader2
            className="h-4 w-4 shrink-0 motion-safe:animate-spin text-indigo-600 dark:text-indigo-400"
            aria-hidden
          />
        ) : (
          <span
            className="h-4 w-4 shrink-0 rounded-full bg-indigo-600/15 ring-2 ring-indigo-600/30 dark:bg-indigo-400/15 dark:ring-indigo-400/30"
            aria-hidden
          />
        )}
        <span className="min-w-0 flex-1 truncate">{title}</span>
        <span className="shrink-0 text-xs tabular-nums text-slate-500 dark:text-neutral-500">
          {entries.length}
        </span>
        <ChevronDown
          className={[
            'h-4 w-4 shrink-0 text-slate-500 transition dark:text-neutral-400',
            expanded ? 'rotate-180' : '',
          ].join(' ')}
          aria-hidden
        />
      </button>
      {expanded ? (
        <div
          role="log"
          aria-live="polite"
          aria-relevant="additions"
          className={['overflow-y-auto border-t border-slate-200 px-3 py-3 dark:border-neutral-700', maxHeightClassName].join(
            ' ',
          )}
        >
          <div className="relative ps-4">
            <div
              className="absolute top-0 bottom-0 start-[5px] w-px bg-slate-300 dark:bg-neutral-600"
              aria-hidden
            />
            <ul className="space-y-2">
              {entries.map((entry) => (
                <li
                  key={entry.id}
                  className="canvas-import-status-in text-sm leading-snug text-slate-600 dark:text-neutral-400"
                >
                  {entry.text}
                </li>
              ))}
            </ul>
            <div ref={bottomRef} aria-hidden />
          </div>
        </div>
      ) : null}
    </div>
  )
}
