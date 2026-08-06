import { Pause, Play, RotateCcw, Volume2 } from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { IconSwap } from '../ui/icon-swap'
import {
  fetchReadingPreferences,
  fetchMyAccommodationSummary,
  patchReadingPreferences,
} from '../../lib/reading-preferences-api'
import { readAloudFeatureEnabled } from '../../lib/platform-features'
import { formatTTSSpeedLabel, normalizeTTSSpeed, TTS_SPEED_OPTIONS, type TTSSpeed } from '../../lib/tts/speed-options'
import { screenReaderLikelyActive, useTTS } from '../../lib/tts/useTTS'

const AUTO_PLAY_DELAY_MS = 3000

type ReadAloudControlsProps = {
  /** BCP 47 language hint for voice selection */
  lang?: string
}

export function ReadAloudControls({ lang = 'en-US' }: ReadAloudControlsProps) {
  const enabled = readAloudFeatureEnabled()
  const [prefsLoaded, setPrefsLoaded] = useState(false)
  const [speed, setSpeed] = useState<TTSSpeed>(1)
  const [voiceName, setVoiceName] = useState<string | null>(null)
  const [accommodationTts, setAccommodationTts] = useState(false)
  const [expanded, setExpanded] = useState(false)

  const persistSpeed = useCallback(async (next: TTSSpeed) => {
    setSpeed(next)
    try {
      await patchReadingPreferences({ ttsSpeed: next })
    } catch {
      /* silent — local speed still applies */
    }
  }, [])

  const tts = useTTS({
    lang,
    speed,
    voiceName,
    onSpeedChange: persistSpeed,
  })
  const playRef = useRef(tts.play)
  useEffect(() => {
    playRef.current = tts.play
  }, [tts.play])

  useEffect(() => {
    if (!enabled) return
    let cancelled = false
    void (async () => {
      try {
        const [prefs, acc] = await Promise.all([
          fetchReadingPreferences(),
          fetchMyAccommodationSummary(),
        ])
        if (cancelled) return
        setSpeed(normalizeTTSSpeed(prefs.ttsSpeed))
        setVoiceName(prefs.ttsVoiceName)
        const accTts = acc.accommodations.some((a) => a.ttsEnabled)
        setAccommodationTts(accTts)
        setPrefsLoaded(true)
        if (accTts && !screenReaderLikelyActive()) {
          window.setTimeout(() => {
            if (!cancelled) playRef.current()
          }, AUTO_PLAY_DELAY_MS)
        }
      } catch {
        if (!cancelled) setPrefsLoaded(true)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [enabled])

  const voiceOptions = useMemo(
    () =>
      tts.voices.map((v) => ({
        name: v.name,
        label: `${v.name} (${v.lang})`,
      })),
    [tts.voices],
  )

  if (!enabled) return null

  const playing = tts.status === 'playing'
  const paused = tts.status === 'paused'

  const onVoiceChange = async (name: string) => {
    const next = name === '' ? null : name
    setVoiceName(next)
    tts.setVoice(next)
    try {
      await patchReadingPreferences({ ttsVoiceName: next })
    } catch {
      /* keep local selection */
    }
  }

  return (
    <>
      <button
        type="button"
        data-read-aloud-trigger
        onClick={() => {
          setExpanded(true)
          tts.toggle()
        }}
        className="inline-flex items-center gap-1.5 rounded-lg border border-border-default bg-surface-raised px-3 py-2 text-xs font-medium text-fg-muted shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base dark:border-border-default dark:bg-surface-overlay dark:text-fg-default dark:hover:bg-neutral-700"
        aria-label={playing ? 'Pause read aloud' : 'Read aloud'}
        aria-pressed={playing}
      >
        <Volume2 className="h-3.5 w-3.5" aria-hidden />
        Read aloud
      </button>

      {(expanded || playing || paused) && (
        <div
          data-read-aloud-controls
          role="toolbar"
          aria-label="Read aloud controls"
          className="fixed inset-x-0 bottom-0 z-40 border-t border-border-default bg-white/95 px-4 py-3 shadow-lg backdrop-blur dark:border-border-default/95"
        >
          <div className="mx-auto flex max-w-3xl flex-wrap items-center gap-3">
            {accommodationTts && prefsLoaded ? (
              <p className="w-full text-xs text-accent-fg">
                Reading enabled by your accommodation plan
              </p>
            ) : null}
            {tts.unavailableMessage ? (
              <p className="w-full text-xs text-amber-800 dark:text-amber-200" role="status">
                {tts.unavailableMessage}
              </p>
            ) : null}
            <button
              type="button"
              aria-label={playing ? 'Pause' : 'Play'}
              aria-pressed={playing}
              aria-keyshortcuts="Alt+P"
              onClick={() => tts.toggle()}
              className="inline-flex items-center gap-1.5 rounded-lg bg-accent-solid px-3 py-2 text-sm font-semibold text-white hover:bg-indigo-500"
            >
              <IconSwap active={playing} activeIcon={Pause} inactiveIcon={Play} iconClassName="h-4 w-4" />
              {playing ? 'Pause' : 'Play'}
            </button>
            <button
              type="button"
              aria-label="Restart read aloud"
              onClick={() => tts.restart()}
              className="inline-flex items-center gap-1.5 rounded-lg border border-border-strong px-3 py-2 text-sm font-semibold text-fg-default hover:bg-surface-base dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
            >
              <RotateCcw className="h-4 w-4" aria-hidden />
              Restart
            </button>
            <label className="inline-flex items-center gap-2 text-sm text-fg-default">
              <span className="sr-only">Reading speed</span>
              <span aria-hidden>Speed</span>
              <select
                aria-label="Reading speed"
                value={String(speed)}
                onChange={(e) => {
                  const next = normalizeTTSSpeed(Number(e.target.value))
                  void persistSpeed(next)
                  tts.setSpeed(next)
                }}
                className="rounded-lg border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-overlay"
              >
                {TTS_SPEED_OPTIONS.map((s) => (
                  <option key={s} value={String(s)}>
                    {formatTTSSpeedLabel(s)}
                  </option>
                ))}
              </select>
            </label>
            {voiceOptions.length > 0 ? (
              <label className="inline-flex min-w-0 flex-1 items-center gap-2 text-sm text-fg-default">
                <span className="shrink-0">Voice</span>
                <select
                  aria-label="Voice"
                  value={voiceName ?? ''}
                  onChange={(e) => void onVoiceChange(e.target.value)}
                  className="min-w-0 flex-1 rounded-lg border border-border-strong bg-surface-raised px-2 py-1.5 text-sm dark:border-border-default dark:bg-surface-overlay"
                >
                  <option value="">System default</option>
                  {voiceOptions.map((v) => (
                    <option key={v.name} value={v.name}>
                      {v.label}
                    </option>
                  ))}
                </select>
              </label>
            ) : null}
            {tts.sentenceCount > 0 ? (
              <span className="text-xs text-fg-muted" aria-live="polite">
                Sentence {Math.min(tts.sentenceIndex + 1, tts.sentenceCount)} of {tts.sentenceCount}
              </span>
            ) : null}
          </div>
        </div>
      )}
    </>
  )
}
