import { useCallback, useEffect, useRef, useState } from 'react'
import { useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useScreenShare } from '../../lib/screen-share-realtime'

export default function ScreenSharePresentPage() {
  const { t } = useTranslation('common')
  const { courseCode: rawCode, sessionId: rawSession } = useParams<{
    courseCode: string
    sessionId: string
  }>()
  const [params] = useSearchParams()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const sessionId = rawSession ? decodeURIComponent(rawSession) : ''
  const joinToken = params.get('token') ?? undefined
  const videoRef = useRef<HTMLVideoElement>(null)
  const [hasStream, setHasStream] = useState(false)

  const onRemoteStream = useCallback((stream: MediaStream | null) => {
    const el = videoRef.current
    if (!el) return
    el.srcObject = stream
    setHasStream(!!stream)
    if (stream) void el.play().catch(() => undefined)
  }, [])

  const { conn, presenterId, announcement } = useScreenShare({
    courseCode,
    sessionId,
    role: 'display',
    joinToken,
    enabled: !!courseCode && !!sessionId,
    onRemoteStream,
  })

  useEffect(() => {
    const preferReduced =
      typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches
    if (preferReduced) document.documentElement.dataset.reducedMotion = 'true'
  }, [])

  return (
    <div
      className="relative min-h-screen bg-zinc-950 text-zinc-50 motion-reduce:transition-none"
      data-no-flash="true"
    >
      <video
        ref={videoRef}
        className="absolute inset-0 h-full w-full object-contain bg-black"
        autoPlay
        muted
        playsInline
        aria-label={t('screenShare.present.videoLabel')}
      />
      <div className="pointer-events-none absolute inset-x-0 top-0 flex justify-center p-4">
        {hasStream && presenterId ? (
          <p
            className="rounded-full bg-black/60 px-4 py-1.5 text-sm text-zinc-100"
            role="status"
            aria-live="polite"
          >
            {t('screenShare.present.presenting', { name: presenterId.slice(0, 8) })}
          </p>
        ) : (
          <div className="max-w-lg text-center" role="status" aria-live="polite">
            <p className="text-2xl font-semibold">{t('screenShare.present.waiting')}</p>
            <p className="mt-2 text-zinc-400">{t('screenShare.present.joinHint')}</p>
          </div>
        )}
      </div>
      {(conn === 'reconnecting' || announcement) && (
        <p className="absolute bottom-4 left-4 rounded bg-black/50 px-3 py-1 text-sm text-amber-200" role="status">
          {conn === 'reconnecting' ? t('screenShare.state.reconnecting') : announcement}
        </p>
      )}
    </div>
  )
}
