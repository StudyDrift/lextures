import { useEffect, useMemo, useRef, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'
import { findDueCheckpoint, mergeLocalSegments } from './checkpoint-engine'
import { CheckpointMediaPlayer } from './media-player'
import { QuestionCard } from './question-card'
import { TranscriptPanel } from './transcript-panel'
import {
  formatTimestamp,
  isCheckpointDone,
  parseTranscriptMarkdown,
  type AnswerResult,
  type Checkpoint,
  type CheckpointAnswer,
  type MediaCheckpointsConfig,
  type MediaCheckpointsState,
} from './types'

const PROGRESS_THROTTLE_MS = 15_000

export default function MediaCheckpointsRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const cfg = config as MediaCheckpointsConfig
  const st = state as MediaCheckpointsState
  const checkpoints = Array.isArray(cfg.checkpoints) ? cfg.checkpoints : []
  const answers = (st.answers && typeof st.answers === 'object' ? st.answers : {}) as Record<
    string,
    CheckpointAnswer
  >
  const media = cfg.media ?? {}
  const kind = media.kind === 'audio' ? 'audio' : 'video'
  const durationSec = typeof media.durationSec === 'number' ? media.durationSec : 0
  const src = typeof media.url === 'string' ? media.url : ''
  const captionUrl = typeof media.captionUrl === 'string' ? media.captionUrl : undefined
  const preventSkip = cfg.preventSkipPastUnanswered === true
  const transcriptLines = useMemo(
    () => parseTranscriptMarkdown(cfg.transcriptMarkdown ?? ''),
    [cfg.transcriptMarkdown],
  )

  const [currentTime, setCurrentTime] = useState(0)
  const [activeCp, setActiveCp] = useState<Checkpoint | null>(null)
  const [lastResult, setLastResult] = useState<AnswerResult | null>(null)
  const [busy, setBusy] = useState(false)
  const [transcriptOnly, setTranscriptOnly] = useState(Boolean(st.usedTranscriptOnly))
  const [mediaFailed, setMediaFailed] = useState(false)
  const [localSegments, setLocalSegments] = useState<[number, number][]>(
    Array.isArray(st.watchedSegments) ? (st.watchedSegments as [number, number][]) : [],
  )
  const promptedRef = useRef<Set<string>>(new Set())
  const lastProgressAt = useRef(0)
  const furthestRef = useRef(st.furthestSec ?? 0)

  const watchedKey = JSON.stringify(st.watchedSegments ?? [])
  useEffect(() => {
    if (Array.isArray(st.watchedSegments)) {
      setLocalSegments(st.watchedSegments as [number, number][])
    }
  }, [watchedKey, st.watchedSegments])

  const answeredCount = checkpoints.filter((cp) => isCheckpointDone(answers, cp)).length

  async function flushProgress(force = false) {
    if (readOnly) return
    const now = Date.now()
    if (!force && now - lastProgressAt.current < PROGRESS_THROTTLE_MS) return
    lastProgressAt.current = now
    try {
      await runAction('record_progress', {
        watchedSegments: localSegments,
        furthestSec: furthestRef.current,
        usedTranscriptOnly: transcriptOnly || undefined,
      })
    } catch {
      // Fall back to host autosave of state fields.
      void save({
        v: 1,
        answers,
        watchedSegments: localSegments,
        furthestSec: furthestRef.current,
        usedTranscriptOnly: transcriptOnly || undefined,
      })
    }
  }

  function handleTimeUpdate(time: number, playing: boolean) {
    setCurrentTime(time)
    if (time > furthestRef.current) furthestRef.current = time
    if (!playing || activeCp || transcriptOnly || mediaFailed) return
    const due = findDueCheckpoint(checkpoints, answers, time, promptedRef.current)
    if (!due) return
    promptedRef.current.add(due.id)
    setActiveCp(due)
    setLastResult(null)
    announce(
      t('contentTools.tools.media_checkpoints.announcePaused', {
        time: formatTimestamp(due.atSec),
        n: checkpoints.findIndex((c) => c.id === due.id) + 1,
        total: checkpoints.length,
      }),
    )
  }

  function handleSegment(start: number, end: number) {
    setLocalSegments((prev) => {
      const next = mergeLocalSegments(prev, start, end)
      return next
    })
    if (end > furthestRef.current) furthestRef.current = end
    void flushProgress(false)
  }

  function openCheckpoint(id: string) {
    const cp = checkpoints.find((c) => c.id === id)
    if (!cp) return
    promptedRef.current.add(cp.id)
    setActiveCp(cp)
    setLastResult(null)
    announce(
      t('contentTools.tools.media_checkpoints.announcePaused', {
        time: formatTimestamp(cp.atSec),
        n: checkpoints.findIndex((c) => c.id === cp.id) + 1,
        total: checkpoints.length,
      }),
    )
  }

  async function onSubmit(value: string | string[] | number) {
    if (!activeCp || readOnly || busy) return
    setBusy(true)
    try {
      const raw = await runAction('answer_checkpoint', {
        checkpointId: activeCp.id,
        value,
        transcriptOnly: transcriptOnly || mediaFailed || undefined,
      })
      const result =
        raw && typeof raw === 'object' ? (raw as AnswerResult) : ({ correct: false } as AnswerResult)
      setLastResult(result)
      if (result.error) {
        announce(result.message || result.error)
      } else if (result.correct) {
        announce(t('contentTools.tools.media_checkpoints.announceCorrect'))
      } else {
        announce(t('contentTools.tools.media_checkpoints.announceIncorrect'))
      }
    } catch {
      setLastResult({
        error: 'error',
        message: t('contentTools.tools.media_checkpoints.error'),
      })
    } finally {
      setBusy(false)
    }
  }

  function onContinue() {
    setActiveCp(null)
    setLastResult(null)
    announce(t('contentTools.tools.media_checkpoints.announceContinue'))
  }

  function onMediaError() {
    setMediaFailed(true)
    setTranscriptOnly(true)
    announce(t('contentTools.tools.media_checkpoints.mediaUnavailable'))
  }

  function onSeekClamped() {
    announce(t('contentTools.tools.media_checkpoints.seekBlocked'))
  }

  useEffect(() => {
    return () => {
      void flushProgress(true)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- flush once on unmount
  }, [])

  const showPlayer = !transcriptOnly && !mediaFailed && Boolean(src)

  return (
    <div className="space-y-4" data-content-tool="media_checkpoints">
      {mediaFailed || (!src && transcriptLines.length > 0) ? (
        <div
          className="rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-950 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-100"
          role="status"
        >
          {t('contentTools.tools.media_checkpoints.mediaUnavailable')}
        </div>
      ) : null}

      {showPlayer ? (
        <CheckpointMediaPlayer
          kind={kind}
          src={src}
          captionUrl={captionUrl}
          durationSec={durationSec}
          checkpoints={checkpoints}
          answers={answers}
          preventSkip={preventSkip}
          pausedForCheckpoint={Boolean(activeCp)}
          onTimeUpdate={handleTimeUpdate}
          onSegment={handleSegment}
          onMediaError={onMediaError}
          onSeekClamped={onSeekClamped}
          t={t}
        />
      ) : null}

      {activeCp ? (
        <QuestionCard
          checkpoint={activeCp}
          index={Math.max(0, checkpoints.findIndex((c) => c.id === activeCp.id))}
          total={checkpoints.length}
          disabled={readOnly}
          busy={busy}
          lastResult={lastResult}
          onSubmit={onSubmit}
          onContinue={onContinue}
          t={t}
        />
      ) : null}

      <TranscriptPanel
        lines={transcriptLines}
        checkpoints={checkpoints}
        currentTime={currentTime}
        transcriptOnly={transcriptOnly}
        onToggleTranscriptOnly={(next) => {
          setTranscriptOnly(next)
          if (next) {
            announce(t('contentTools.tools.media_checkpoints.announceTranscriptOnly'))
          }
        }}
        onSeek={(sec) => {
          setCurrentTime(sec)
        }}
        onOpenCheckpoint={openCheckpoint}
        t={t}
      />

      <footer className="flex flex-wrap items-center justify-between gap-2 text-xs text-fg-muted">
        <span>
          {t('contentTools.tools.media_checkpoints.progress', {
            done: answeredCount,
            total: checkpoints.length,
          })}
        </span>
        <span>
          {t('contentTools.tools.media_checkpoints.furthest', {
            time: formatTimestamp(furthestRef.current || st.furthestSec || 0),
          })}
        </span>
        <span className="text-fg-muted">
          {t('contentTools.tools.media_checkpoints.watchDisclaimer')}
        </span>
      </footer>
    </div>
  )
}
