import { useEffect, useRef, useState } from 'react'
import {
  loadMediaPlaybackPreferences,
  saveMediaPlaybackPreferences,
  supportedPlaybackRates,
} from '../../../../lib/media-playback-preferences'
import { clampSeekTime } from './checkpoint-engine'
import {
  formatTimestamp,
  isCheckpointDone,
  type Checkpoint,
  type CheckpointAnswer,
} from './types'

type Props = {
  kind: 'video' | 'audio'
  src?: string
  captionUrl?: string
  durationSec: number
  checkpoints: Checkpoint[]
  answers: Record<string, CheckpointAnswer>
  preventSkip: boolean
  pausedForCheckpoint: boolean
  onTimeUpdate: (t: number, playing: boolean) => void
  onSegment: (start: number, end: number) => void
  onMediaError: () => void
  onSeekClamped: () => void
  t: (key: string, options?: Record<string, unknown>) => string
}

export function CheckpointMediaPlayer({
  kind,
  src,
  captionUrl,
  durationSec,
  checkpoints,
  answers,
  preventSkip,
  pausedForCheckpoint,
  onTimeUpdate,
  onSegment,
  onMediaError,
  onSeekClamped,
  t,
}: Props) {
  const mediaRef = useRef<HTMLVideoElement | HTMLAudioElement | null>(null)
  const playHeadRef = useRef(0)
  const segmentStartRef = useRef<number | null>(null)
  const [prefs, setPrefs] = useState(() => loadMediaPlaybackPreferences())
  const [current, setCurrent] = useState(0)
  const [duration, setDuration] = useState(durationSec)
  const [playing, setPlaying] = useState(false)

  useEffect(() => {
    const el = mediaRef.current
    if (!el) return
    el.playbackRate = prefs.playbackRate
    el.volume = prefs.volume
    el.muted = prefs.muted
    for (const track of Array.from(el.textTracks)) {
      track.mode = prefs.captionsEnabled ? 'showing' : 'hidden'
    }
  }, [prefs, src, captionUrl])

  useEffect(() => {
    if (pausedForCheckpoint) {
      mediaRef.current?.pause()
      flushSegment()
    }
  }, [pausedForCheckpoint])

  function flushSegment() {
    const start = segmentStartRef.current
    if (start == null) return
    const end = playHeadRef.current
    segmentStartRef.current = null
    if (end > start + 0.2) onSegment(start, end)
  }

  function applySeek(target: number) {
    const el = mediaRef.current
    if (!el) return
    const { time, clamped } = clampSeekTime(preventSkip, checkpoints, answers, target)
    el.currentTime = time
    playHeadRef.current = time
    setCurrent(time)
    if (clamped) onSeekClamped()
  }

  function togglePlay() {
    const el = mediaRef.current
    if (!el || pausedForCheckpoint) return
    if (el.paused) void el.play()
    else el.pause()
  }

  function updatePrefs(partial: Partial<typeof prefs>) {
    const next = { ...prefs, ...partial }
    setPrefs(next)
    saveMediaPlaybackPreferences(next)
  }

  const MediaTag = kind === 'audio' ? 'audio' : 'video'

  return (
    <div className="space-y-2" data-testid="media-checkpoint-player">
      <MediaTag
        ref={mediaRef as never}
        className={
          kind === 'video'
            ? 'aspect-video w-full bg-black object-contain'
            : 'w-full'
        }
        playsInline
        preload="metadata"
        controls={false}
        src={src || undefined}
        onPlay={() => {
          setPlaying(true)
          segmentStartRef.current = playHeadRef.current
        }}
        onPause={() => {
          setPlaying(false)
          flushSegment()
        }}
        onEnded={() => {
          setPlaying(false)
          flushSegment()
        }}
        onError={() => onMediaError()}
        onLoadedMetadata={() => {
          const el = mediaRef.current
          if (el && Number.isFinite(el.duration) && el.duration > 0) {
            setDuration(el.duration)
          }
        }}
        onTimeUpdate={() => {
          const el = mediaRef.current
          if (!el) return
          playHeadRef.current = el.currentTime
          setCurrent(el.currentTime)
          onTimeUpdate(el.currentTime, !el.paused)
        }}
        onSeeking={() => {
          const el = mediaRef.current
          if (!el) return
          const { time, clamped } = clampSeekTime(
            preventSkip,
            checkpoints,
            answers,
            el.currentTime,
          )
          if (clamped && Math.abs(el.currentTime - time) > 0.05) {
            el.currentTime = time
            onSeekClamped()
          }
        }}
        onKeyDown={(e) => {
          const el = mediaRef.current
          if (!el) return
          if (e.key === ' ' || e.key === 'k' || e.key === 'K') {
            e.preventDefault()
            togglePlay()
          } else if (e.key === 'ArrowRight') {
            e.preventDefault()
            applySeek(el.currentTime + 5)
          } else if (e.key === 'ArrowLeft') {
            e.preventDefault()
            applySeek(el.currentTime - 5)
          } else if (e.key === 'c' || e.key === 'C') {
            e.preventDefault()
            updatePrefs({ captionsEnabled: !prefs.captionsEnabled })
          }
        }}
        tabIndex={0}
        aria-label={t('contentTools.tools.media_checkpoints.playerLabel')}
      >
        {captionUrl ? (
          <track kind="captions" src={captionUrl} srcLang="en" default={prefs.captionsEnabled} />
        ) : null}
      </MediaTag>

      <div className="relative h-3 rounded bg-slate-200 dark:bg-neutral-700">
        <div
          className="absolute inset-y-0 start-0 rounded bg-sky-600/70 dark:bg-sky-500/70"
          style={{ width: `${duration > 0 ? (current / duration) * 100 : 0}%` }}
        />
        {checkpoints.map((cp) => {
          const done = isCheckpointDone(answers, cp)
          const left = duration > 0 ? (cp.atSec / duration) * 100 : 0
          return (
            <button
              key={cp.id}
              type="button"
              title={`${formatTimestamp(cp.atSec)} — ${cp.question.prompt}`}
              aria-label={t('contentTools.tools.media_checkpoints.markerLabel', {
                time: formatTimestamp(cp.atSec),
                status: done
                  ? t('contentTools.tools.media_checkpoints.answered')
                  : t('contentTools.tools.media_checkpoints.unanswered'),
              })}
              className={`absolute top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 border border-white shadow ${ done ? 'rotate-45 bg-emerald-600' : 'rounded-full bg-amber-500' }`}
              style={{ left: `${left}%` }}
              onClick={() => applySeek(cp.atSec)}
            />
          )
        })}
        <input
          type="range"
          min={0}
          max={duration || durationSec || 1}
          step={0.1}
          value={current}
          aria-label={t('contentTools.tools.media_checkpoints.seek')}
          className="absolute inset-0 w-full cursor-pointer opacity-0"
          onChange={(e) => applySeek(Number(e.target.value))}
        />
      </div>

      <div className="flex flex-wrap items-center gap-2 text-xs text-fg-default">
        <button
          type="button"
          className="min-h-11 min-w-11 rounded-md border border-border-strong px-3 dark:border-border-default"
          onClick={togglePlay}
          disabled={pausedForCheckpoint || !src}
        >
          {playing
            ? t('contentTools.tools.media_checkpoints.pause')
            : t('contentTools.tools.media_checkpoints.play')}
        </button>
        <span className="font-mono">
          {formatTimestamp(current)} / {formatTimestamp(duration || durationSec)}
        </span>
        <label className="flex min-h-11 items-center gap-1">
          <span>{t('contentTools.tools.media_checkpoints.speed')}</span>
          <select
            value={prefs.playbackRate}
            onChange={(e) => updatePrefs({ playbackRate: Number(e.target.value) })}
            className="rounded border border-border-strong bg-surface-raised px-1 py-1 dark:border-border-default dark:bg-surface-base"
          >
            {supportedPlaybackRates().map((r) => (
              <option key={r} value={r}>
                {r}x
              </option>
            ))}
          </select>
        </label>
        <label className="flex min-h-11 items-center gap-1">
          <span>{t('contentTools.tools.media_checkpoints.volume')}</span>
          <input
            type="range"
            min={0}
            max={1}
            step={0.05}
            value={prefs.muted ? 0 : prefs.volume}
            onChange={(e) =>
              updatePrefs({ volume: Number(e.target.value), muted: Number(e.target.value) === 0 })
            }
          />
        </label>
        {captionUrl ? (
          <label className="flex min-h-11 items-center gap-1">
            <input
              type="checkbox"
              checked={prefs.captionsEnabled}
              onChange={(e) => updatePrefs({ captionsEnabled: e.target.checked })}
            />
            {t('contentTools.tools.media_checkpoints.captions')}
          </label>
        ) : null}
      </div>
      <p className="text-[11px] text-fg-muted">
        {t('contentTools.tools.media_checkpoints.shortcuts')}
      </p>
    </div>
  )
}
