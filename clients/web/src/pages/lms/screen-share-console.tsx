import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  canCaptureEntireScreen,
  createScreenShareSession,
  displaySurfaceOf,
  endScreenShareSession,
  type IceServersPayload,
  type ScreenShareSession,
} from '../../lib/screen-share-api'
import { useScreenShare } from '../../lib/screen-share-realtime'
import { getJwtSubject } from '../../lib/auth'

type Props = {
  courseCode: string
  canHost: boolean
}

export function ScreenShareConsole({ courseCode, canHost }: Props) {
  const { t } = useTranslation('common')
  const viewerId = getJwtSubject()
  const [session, setSession] = useState<ScreenShareSession | null>(null)
  const [joinToken, setJoinToken] = useState<string | undefined>()
  const [ice, setIce] = useState<RTCIceServer[]>([])
  const [busy, setBusy] = useState(false)
  const [warnSurface, setWarnSurface] = useState(false)
  const [pendingStream, setPendingStream] = useState<MediaStream | null>(null)
  const videoRef = useRef<HTMLVideoElement>(null)
  const captureOk = canCaptureEntireScreen()

  const onRemoteStream = useCallback((stream: MediaStream | null) => {
    const el = videoRef.current
    if (!el) return
    el.srcObject = stream
    if (stream) void el.play().catch(() => undefined)
  }, [])

  const {
    conn,
    presenterId,
    announcement,
    requestPresent,
    publishLocalStream,
    stopLocalShare,
    applyTurn,
    error,
  } = useScreenShare({
    courseCode,
    sessionId: session?.id ?? '',
    role: canHost ? 'host' : 'viewer',
    joinToken,
    iceServers: ice,
    enabled: !!session?.id,
    onRemoteStream,
  })

  useEffect(() => {
    if (ice.length) applyTurn({ iceServers: ice, ttlSeconds: 0 } as IceServersPayload)
  }, [ice, applyTurn])

  const startSession = async () => {
    if (!canHost) return
    setBusy(true)
    try {
      const res = await createScreenShareSession(courseCode, { policy: 'request' })
      setSession(res.session)
      setJoinToken(res.joinToken)
      setIce(res.turn.iceServers ?? [])
    } catch (e) {
      console.error(e)
    } finally {
      setBusy(false)
    }
  }

  const endSession = async () => {
    if (!session) return
    setBusy(true)
    try {
      stopLocalShare()
      await endScreenShareSession(courseCode, session.id)
      setSession(null)
      setJoinToken(undefined)
    } finally {
      setBusy(false)
    }
  }

  const beginShare = async () => {
    if (!captureOk) return
    try {
      const stream = await navigator.mediaDevices.getDisplayMedia({
        video: { displaySurface: 'monitor' } as MediaTrackConstraints,
        audio: false,
      })
      const track = stream.getVideoTracks()[0]
      const surface = displaySurfaceOf(track)
      if (surface && surface !== 'monitor') {
        setPendingStream(stream)
        setWarnSurface(true)
        return
      }
      requestPresent()
      await publishLocalStream(stream)
    } catch {
      /* user cancelled picker */
    }
  }

  const continueDespiteWarn = async () => {
    if (!pendingStream) return
    setWarnSurface(false)
    requestPresent()
    await publishLocalStream(pendingStream)
    setPendingStream(null)
  }

  const reshare = () => {
    pendingStream?.getTracks().forEach((tr) => tr.stop())
    setPendingStream(null)
    setWarnSurface(false)
    void beginShare()
  }

  const isSharing = !!presenterId && !!viewerId && presenterId === viewerId
  const displayPath = session
    ? `/courses/${encodeURIComponent(courseCode)}/screen-share/${encodeURIComponent(session.id)}/present${
        joinToken ? `?token=${encodeURIComponent(joinToken)}` : ''
      }`
    : ''

  return (
    <section className="space-y-4" aria-labelledby="screen-share-heading">
      <div>
        <h2 id="screen-share-heading" className="text-lg font-semibold text-zinc-900 dark:text-zinc-50">
          {t('screenShare.console.title')}
        </h2>
        <p className="text-sm text-zinc-600 dark:text-zinc-400">{t('screenShare.console.subtitle')}</p>
      </div>

      {!session && canHost && (
        <button
          type="button"
          className="rounded-md bg-zinc-900 px-3 py-2 text-sm text-white disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900"
          disabled={busy}
          onClick={() => void startSession()}
        >
          {t('screenShare.console.startSession')}
        </button>
      )}

      {session && (
        <div className="space-y-3">
          <p className="text-sm text-zinc-600 dark:text-zinc-400" role="status" aria-live="polite">
            {t('screenShare.state.conn', { status: conn })}
            {presenterId
              ? ` · ${t('screenShare.state.presenter', { id: presenterId.slice(0, 8) })}`
              : ` · ${t('screenShare.state.waitingPresenter')}`}
          </p>
          {announcement && (
            <p className="sr-only" role="status" aria-live="polite">
              {announcement}
            </p>
          )}
          {error && <p className="text-sm text-red-600">{error}</p>}

          <div className="flex flex-wrap gap-2">
            {captureOk ? (
              !isSharing ? (
                <button
                  type="button"
                  className="rounded-md bg-emerald-700 px-3 py-2 text-sm text-white"
                  onClick={() => void beginShare()}
                >
                  {t('screenShare.start.share')}
                </button>
              ) : (
                <button
                  type="button"
                  className="rounded-md bg-red-700 px-3 py-2 text-sm text-white"
                  onClick={() => stopLocalShare()}
                >
                  {t('screenShare.present.stop')}
                </button>
              )
            ) : (
              <p className="text-sm text-zinc-500">{t('screenShare.error.unsupportedCapture')}</p>
            )}
            {canHost && (
              <button
                type="button"
                className="rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-600"
                disabled={busy}
                onClick={() => void endSession()}
              >
                {t('screenShare.console.endSession')}
              </button>
            )}
            <Link className="rounded-md border border-zinc-300 px-3 py-2 text-sm dark:border-zinc-600" to={displayPath} target="_blank">
              {t('screenShare.console.openDisplay')}
            </Link>
          </div>

          {isSharing && (
            <div
              className="flex items-center gap-2 rounded-md border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-900 dark:border-red-800 dark:bg-red-950 dark:text-red-100"
              role="status"
            >
              <span aria-hidden="true">●</span>
              {t('screenShare.present.sharingBar', { course: courseCode })}
            </div>
          )}

          {warnSurface && (
            <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm dark:border-amber-700 dark:bg-amber-950" role="alertdialog">
              <p>{t('screenShare.consent.notEntireScreen')}</p>
              <div className="mt-2 flex gap-2">
                <button type="button" className="rounded bg-zinc-900 px-2 py-1 text-white" onClick={() => void continueDespiteWarn()}>
                  {t('screenShare.consent.continue')}
                </button>
                <button type="button" className="rounded border px-2 py-1" onClick={reshare}>
                  {t('screenShare.consent.reshare')}
                </button>
              </div>
            </div>
          )}

          <video
            ref={videoRef}
            className="aspect-video w-full max-w-3xl rounded-md bg-black"
            autoPlay
            playsInline
            muted
            controls
            aria-label={t('screenShare.present.videoLabel')}
          />
        </div>
      )}
    </section>
  )
}
