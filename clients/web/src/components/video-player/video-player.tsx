import Hls from 'hls.js'
import { useEffect, useRef, useState, useCallback } from 'react'

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
  className?: string
}

type QualityLevel = { height: number; label: string; index: number }

export function VideoPlayer({
  masterPlaylistUrl,
  fallbackSrc,
  posterUrl,
  transcodeStatus,
  ariaLabel = 'Video player',
  className = '',
}: VideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  const [playing, setPlaying] = useState(false)
  const [currentTime, setCurrentTime] = useState(0)
  const [duration, setDuration] = useState(0)
  const [levels, setLevels] = useState<QualityLevel[]>([])
  const [currentLevel, setCurrentLevel] = useState<number>(-1) // -1 = auto

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
      }
    },
    [togglePlay, seek, toggleFullscreen],
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

  return (
    <div className={`relative flex flex-col rounded-lg bg-black ${className}`} onKeyDown={handleKeyDown} tabIndex={0} role="group" aria-label={ariaLabel}>
      <video
        ref={videoRef}
        poster={posterUrl}
        playsInline
        className="w-full rounded-t-lg"
        aria-label={ariaLabel}
        tabIndex={-1}
      >
        <track kind="captions" src="" default />
      </video>

      {/* Custom controls */}
      <div className="flex items-center gap-2 rounded-b-lg bg-gray-900 px-3 py-2 text-white">
        <button
          onClick={togglePlay}
          aria-label={playing ? 'Pause' : 'Play'}
          className="rounded p-1 hover:bg-gray-700"
        >
          {playing ? '⏸' : '▶'}
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

        {/* Quality picker (FR-9) */}
        {levels.length > 0 && (
          <select
            value={currentLevel}
            onChange={(e) => setQuality(Number(e.target.value))}
            className="ml-auto rounded bg-gray-700 px-1 py-0.5 text-xs"
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
