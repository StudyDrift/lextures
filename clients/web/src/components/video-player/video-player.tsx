import Hls from 'hls.js'
import { useEffect, useRef, useState, useCallback } from 'react'
import { CaptionSettings } from '../media/caption-settings'
import {
  captionStyleVars,
  loadCaptionPreferences,
  saveCaptionPreferences,
  type CaptionPreferences,
} from '../../lib/caption-preferences'

export interface TranscodeStatus {
  status: 'queued' | 'processing' | 'done' | 'failed'
  master_playlist_url?: string
  poster_url?: string
  renditions?: string[]
  error?: string
  job_id?: string
}

interface VideoPlayerProps {
  /** HLS master playlist URL (when transcoding is done). */
  masterPlaylistUrl?: string
  /** Fallback raw MP4/WebM URL (served while transcoding or as fallback). */
  fallbackSrc?: string
  /** Poster image URL. */
  posterUrl?: string
  /** Current transcode status — drives processing state UI. */
  transcodeStatus?: TranscodeStatus
  /** ARIA label for the video element. */
  ariaLabel?: string
  /** Authenticated URL for WebVTT track (plan 12.4). */
  captionTrackSrc?: string
  /** BCP-47 language for the track element. */
  captionLang?: string
  className?: string
}

type QualityLevel = { height: number; label: string; index: number }

export function VideoPlayer({
  masterPlaylistUrl,
  fallbackSrc,
  posterUrl,
  transcodeStatus,
  ariaLabel = 'Video player',
  captionTrackSrc,
  captionLang = 'en',
  className = '',
}: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  const [playing, setPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [levels, setLevels] = useState<QualityLevel[]>([])
  const [currentLevel, setCurrentLevel] = useState<number>(-1) // -1 = auto
  const [captionPrefs, setCaptionPrefs] = useState<CaptionPreferences>(() => loadCaptionPreferences())
  const [settingsOpen, setSettingsOpen] = useState(false)

  const status = transcodeStatus?.status

  // Initialize hls.js or native HLS
  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    // Clean up previous instance
    if (hlsRef.current) {
      hlsRef.current.destroy()
      hlsRef.current = null
    }

    if (masterPlaylistUrl) {
      if (Hls.isSupported()) {
        const hls = new Hls({ startLevel: -1 }) // -1 = auto ABR
        hls.loadSource(masterPlaylistUrl)
        hls.attachMedia(video)
        hls.on(Hls.Events.MANIFEST_PARSED, (_, data) => {
          const qs: QualityLevel[] = data.levels.map((l, i) => ({
            height: l.height,
            label: l.height ? `${l.height}p` : `Level ${i + 1}`,
            index: i,
          }))
          setLevels(qs)
        })
        hls.on(Hls.Events.LEVEL_SWITCHED, (_, data) => {
          setCurrentLevel(data.level)
        })
        hlsRef.current = hls
      } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
        // Native HLS on Safari / iOS
        video.src = masterPlaylistUrl
        if (posterUrl) video.poster = posterUrl
      }
    } else if (fallbackSrc) {
      video.src = fallbackSrc
      if (posterUrl) video.poster = posterUrl
    }

    if (posterUrl && !video.poster) {
      video.poster = posterUrl
    }

    return () => {
      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }
    }
  }, [masterPlaylistUrl, fallbackSrc, posterUrl])

  // Video event listeners
  useEffect(() => {
    const video = videoRef.current
    if (!video) return
    const onPlay = () => setPlaying(true)
    const onPause = () => setPlaying(false)
    const onTimeUpdate = () => setCurrentTime(video.currentTime)
    const onDurationChange = () => setDuration(video.duration)
    video.addEventListener('play', onPlay)
    video.addEventListener('pause', onPause)
    video.addEventListener('timeupdate', onTimeUpdate)
    video.addEventListener('durationchange', onDurationChange)
    return () => {
      video.removeEventListener('play', onPlay)
      video.removeEventListener('pause', onPause)
      video.removeEventListener('timeupdate', onTimeUpdate)
      video.removeEventListener('durationchange', onDurationChange)
    }
  }, [])

  useEffect(() => {
    saveCaptionPreferences(captionPrefs)
    const video = videoRef.current
    if (!video?.textTracks?.length) return
    for (let i = 0; i < video.textTracks.length; i++) {
      const track = video.textTracks[i]
      if (track.kind === 'captions' || track.kind === 'subtitles') {
        track.mode = captionPrefs.enabled && captionTrackSrc ? 'showing' : 'hidden'
      }
    }
  }, [captionPrefs, captionTrackSrc])

  const captionStatus =
    captionPrefs.enabled && captionTrackSrc ? 'Captions on' : 'Captions off'

  const toggleCaptions = useCallback(() => {
    setCaptionPrefs((p) => ({ ...p, enabled: !p.enabled }))
  }, [])

  const togglePlay = useCallback(() => {
    const video = videoRef.current
    if (!video) return
    if (video.paused) {
      void video.play()
    } else {
      video.pause()
    }
  }, [])

  const seek = useCallback((delta: number) => {
    const video = videoRef.current
    if (!video) return
    video.currentTime = Math.max(0, Math.min(video.duration, video.currentTime + delta))
  }, [])

  const toggleFullscreen = useCallback(() => {
    const video = videoRef.current
    if (!video) return
    if (document.fullscreenElement) {
      void document.exitFullscreen()
    } else {
      void video.requestFullscreen()
    }
  }, [])

  const setQuality = useCallback((level: number) => {
    if (hlsRef.current) {
      hlsRef.current.currentLevel = level
      setCurrentLevel(level)
    }
  }, [])

  // Keyboard controls (Space = play/pause, ←/→ = seek 10s, F = fullscreen)
  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case ' ':
        case 'k':
          e.preventDefault()
          togglePlay()
          break
        case 'ArrowLeft':
          e.preventDefault()
          seek(-10)
          break
        case 'ArrowRight':
          e.preventDefault()
          seek(10)
          break
        case 'f':
        case 'F':
          e.preventDefault()
          toggleFullscreen()
          break
        case 'c':
        case 'C':
          if (captionTrackSrc) {
            e.preventDefault()
            toggleCaptions()
          }
          break
      }
    },
    [togglePlay, seek, toggleFullscreen, toggleCaptions, captionTrackSrc],
  )

  const formatTime = (s: number) => {
    if (!isFinite(s)) return '0:00'
    const m = Math.floor(s / 60)
    const sec = Math.floor(s % 60)
    return `${m}:${sec.toString().padStart(2, '0')}`
  }

  // Processing state
  if (status === 'queued' || status === 'processing') {
    return (
      <div
        className={`flex flex-col items-center justify-center rounded-lg bg-gray-100 p-8 text-center ${className}`}
        role="status"
        aria-live="polite"
        aria-label="video.processing"
      >
        <div
          className="mb-3 h-8 w-8 animate-spin rounded-full border-4 border-blue-500 border-t-transparent"
          aria-hidden="true"
        />
        <p className="text-sm text-gray-600">Video is processing — check back in a few minutes</p>
        {fallbackSrc && (
          // eslint-disable-next-line jsx-a11y/media-has-caption -- processing-state fallback; captions unavailable until transcode completes (plan 8.4)
          <video
            ref={videoRef}
            src={fallbackSrc}
            poster={posterUrl}
            controls
            playsInline
            className="mt-4 w-full max-w-2xl rounded"
            aria-label={ariaLabel}
          />
        )}
      </div>
    )
  }

  if (status === 'failed') {
    return (
      <div
        className={`flex flex-col items-center justify-center rounded-lg bg-red-50 p-8 text-center ${className}`}
        role="alert"
        aria-label="video.failed"
      >
        <p className="text-sm font-medium text-red-700">
          Video processing failed — contact support
        </p>
        {fallbackSrc && (
          // eslint-disable-next-line jsx-a11y/media-has-caption -- failed-state fallback; no caption file available (plan 8.4)
          <video
            ref={videoRef}
            src={fallbackSrc}
            poster={posterUrl}
            controls
            playsInline
            className="mt-4 w-full max-w-2xl rounded"
            aria-label={ariaLabel}
          />
        )}
      </div>
    )
  }

  const cueStyle = captionStyleVars(captionPrefs)

  return (
    <div
      className={`relative flex flex-col rounded-lg bg-black ${className}`}
      style={cueStyle}
      onKeyDown={handleKeyDown}
      tabIndex={0}
      role="group"
      aria-label={ariaLabel}
    >
      <p className="sr-only" role="status" aria-live="polite">
        {captionStatus}
      </p>
      {/* eslint-disable-next-line jsx-a11y/media-has-caption -- track injected when captionTrackSrc is set */}
      <video
        ref={videoRef}
        poster={posterUrl}
        playsInline
        className="video-with-captions w-full rounded-t-lg"
        aria-label={ariaLabel}
        tabIndex={-1}
      >
        {captionTrackSrc ? (
          <track
            kind="captions"
            src={captionTrackSrc}
            srcLang={captionLang}
            label="Captions"
            default={captionPrefs.enabled}
          />
        ) : null}
      </video>
      <CaptionSettings
        open={settingsOpen}
        prefs={captionPrefs}
        onChange={setCaptionPrefs}
        onClose={() => setSettingsOpen(false)}
      />

      {/* Custom controls */}
      <div className="flex flex-wrap items-center gap-2 rounded-b-lg bg-gray-900 px-3 py-2 text-white">
        <button
          onClick={togglePlay}
          aria-label={playing ? 'Pause' : 'Play'}
          className="lex-icon-hit rounded hover:bg-gray-700"
        >
          <span className={playing ? '' : 'ms-px'} aria-hidden>
            {playing ? '⏸' : '▶'}
          </span>
        </button>

        <button
          onClick={() => seek(-10)}
          aria-label="Seek back 10 seconds"
          className="rounded p-1 text-sm hover:bg-gray-700"
        >
          ↩10
        </button>
        <button
          onClick={() => seek(10)}
          aria-label="Seek forward 10 seconds"
          className="rounded p-1 text-sm hover:bg-gray-700"
        >
          10↪
        </button>

        <span className="text-xs tabular-nums" aria-live="off">
          {formatTime(currentTime)} / {formatTime(duration)}
        </span>

        {captionTrackSrc ? (
          <>
            <button
              type="button"
              onClick={toggleCaptions}
              aria-pressed={captionPrefs.enabled}
              aria-label="Toggle captions"
              className="rounded px-2 py-0.5 text-sm font-semibold hover:bg-gray-700"
            >
              CC
            </button>
            <button
              type="button"
              onClick={() => setSettingsOpen((o) => !o)}
              aria-label="Caption settings"
              aria-haspopup="dialog"
              className="rounded px-2 py-0.5 text-xs hover:bg-gray-700"
            >
              Caption settings
            </button>
          </>
        ) : null}

        {/* Quality picker (FR-9) */}
        {levels.length > 0 && (
          <select
            value={currentLevel}
            onChange={(e) => setQuality(Number(e.target.value))}
            className="ms-auto rounded bg-gray-700 px-1 py-0.5 text-xs"
            aria-label="video.qualityLabel"
          >
            <option value={-1}>Auto</option>
            {levels.map((l) => (
              <option key={l.index} value={l.index}>
                {l.label}
              </option>
            ))}
          </select>
        )}

        <button
          onClick={toggleFullscreen}
          aria-label="Fullscreen"
          className="rounded p-1 hover:bg-gray-700"
        >
          ⛶
        </button>
      </div>
    </div>
  )
}
