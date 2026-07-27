import { useMemo, useState } from 'react'
import type { Checkpoint, TranscriptLine } from './types'
import { formatTimestamp } from './types'

type Props = {
  lines: TranscriptLine[]
  checkpoints: Checkpoint[]
  currentTime: number
  transcriptOnly: boolean
  onToggleTranscriptOnly: (next: boolean) => void
  onSeek: (sec: number) => void
  onOpenCheckpoint: (id: string) => void
  t: (key: string, options?: Record<string, unknown>) => string
}

export function TranscriptPanel({
  lines,
  checkpoints,
  currentTime,
  transcriptOnly,
  onToggleTranscriptOnly,
  onSeek,
  onOpenCheckpoint,
  t,
}: Props) {
  const [query, setQuery] = useState('')
  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase()
    if (!q) return lines
    return lines.filter((l) => l.text.toLowerCase().includes(q))
  }, [lines, query])

  return (
    <aside
      className="flex flex-col gap-3 rounded-lg border border-slate-200 bg-slate-50 p-3 dark:border-neutral-700 dark:bg-neutral-900/60"
      data-testid="media-checkpoint-transcript"
      aria-label={t('contentTools.tools.media_checkpoints.transcript')}
    >
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-semibold text-slate-800 dark:text-neutral-100">
          {t('contentTools.tools.media_checkpoints.transcript')}
        </h3>
        <label className="flex min-h-11 items-center gap-2 text-xs font-medium text-slate-700 dark:text-neutral-200">
          <input
            type="checkbox"
            checked={transcriptOnly}
            onChange={(e) => onToggleTranscriptOnly(e.target.checked)}
            data-testid="media-checkpoint-transcript-only"
          />
          {t('contentTools.tools.media_checkpoints.transcriptOnly')}
        </label>
      </div>

      <label className="block text-xs">
        <span className="sr-only">{t('contentTools.tools.media_checkpoints.transcriptSearch')}</span>
        <input
          type="search"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder={t('contentTools.tools.media_checkpoints.transcriptSearch')}
          className="w-full min-h-11 rounded-md border border-slate-200 bg-white px-2 py-1.5 text-sm dark:border-neutral-600 dark:bg-neutral-950"
        />
      </label>

      {transcriptOnly ? (
        <div className="space-y-2" role="list" aria-label={t('contentTools.tools.media_checkpoints.checkpoints')}>
          {checkpoints.map((cp, i) => (
            <button
              key={cp.id}
              type="button"
              role="listitem"
              onClick={() => onOpenCheckpoint(cp.id)}
              className="flex w-full min-h-11 items-start justify-between gap-2 rounded-md border border-slate-200 bg-white px-3 py-2 text-start text-sm dark:border-neutral-600 dark:bg-neutral-950"
            >
              <span>
                {t('contentTools.tools.media_checkpoints.checkpointN', { n: i + 1 })} — {cp.question.prompt}
              </span>
              <span className="shrink-0 text-xs text-slate-500">{formatTimestamp(cp.atSec)}</span>
            </button>
          ))}
        </div>
      ) : null}

      <ol className="max-h-64 space-y-1 overflow-y-auto text-sm" dir="auto">
        {filtered.length === 0 ? (
          <li className="text-slate-500">{t('contentTools.tools.media_checkpoints.transcriptEmpty')}</li>
        ) : (
          filtered.map((line) => {
            const active = Math.abs(currentTime - line.atSec) < 2.5
            return (
              <li key={line.id}>
                <button
                  type="button"
                  onClick={() => onSeek(line.atSec)}
                  className={`flex w-full min-h-11 gap-2 rounded px-2 py-1.5 text-start ${
                    active
                      ? 'bg-sky-100 text-slate-900 dark:bg-sky-900/40 dark:text-neutral-50'
                      : 'hover:bg-white/80 dark:hover:bg-neutral-800'
                  }`}
                >
                  <span className="shrink-0 font-mono text-xs text-slate-500">
                    {formatTimestamp(line.atSec)}
                  </span>
                  <span>{line.text}</span>
                </button>
              </li>
            )
          })
        )}
      </ol>
    </aside>
  )
}
