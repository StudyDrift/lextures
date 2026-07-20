import { useCallback, useEffect, useRef, useState } from 'react'
import { getAccessToken } from './auth'
import type { IceServersPayload } from './screen-share-api'

const apiBase = import.meta.env.VITE_API_URL ?? ''

export type ScreenShareRole = 'host' | 'presenter' | 'viewer' | 'display'

export type ScreenShareConn = 'connecting' | 'connected' | 'reconnecting' | 'ended' | 'disconnected' | 'error'

function wsURL(courseCode: string, sessionId: string): string {
  const base = apiBase || (typeof window !== 'undefined' ? window.location.origin : '')
  const u = new URL(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/screen-share/sessions/${encodeURIComponent(sessionId)}/ws`,
    base,
  )
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  return u.toString()
}

export type UseScreenShareOpts = {
  courseCode: string
  sessionId: string
  role: ScreenShareRole
  joinToken?: string
  iceServers?: RTCIceServer[]
  enabled?: boolean
  onRemoteStream?: (stream: MediaStream | null) => void
}

export function useScreenShare(opts: UseScreenShareOpts) {
  const { courseCode, sessionId, role, joinToken, iceServers, enabled = true, onRemoteStream } = opts
  const [conn, setConn] = useState<ScreenShareConn>('connecting')
  const [presenterId, setPresenterId] = useState<string | null>(null)
  const [selfRole, setSelfRole] = useState<ScreenShareRole>(role)
  const [error, setError] = useState<string | null>(null)
  const [announcement, setAnnouncement] = useState('')
  const wsRef = useRef<WebSocket | null>(null)
  const pcRef = useRef<RTCPeerConnection | null>(null)
  const localStreamRef = useRef<MediaStream | null>(null)
  const iceRef = useRef<RTCIceServer[]>(iceServers ?? [])
  const closedRef = useRef(false)
  const retryRef = useRef(0)
  const onRemoteStreamRef = useRef(onRemoteStream)
  onRemoteStreamRef.current = onRemoteStream

  useEffect(() => {
    if (iceServers) iceRef.current = iceServers
  }, [iceServers])

  const teardownPC = useCallback(() => {
    pcRef.current?.close()
    pcRef.current = null
    onRemoteStreamRef.current?.(null)
  }, [])

  const ensurePC = useCallback(() => {
    if (pcRef.current) return pcRef.current
    const pc = new RTCPeerConnection({ iceServers: iceRef.current })
    pc.onicecandidate = (ev) => {
      if (!ev.candidate || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return
      wsRef.current.send(JSON.stringify({ type: 'ice-candidate', candidate: ev.candidate.toJSON() }))
    }
    pc.ontrack = (ev) => {
      const stream = ev.streams[0] ?? new MediaStream([ev.track])
      onRemoteStreamRef.current?.(stream)
    }
    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'failed') {
        void pc.restartIce()
      }
    }
    pcRef.current = pc
    return pc
  }, [])

  const send = useCallback((frame: Record<string, unknown>) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(frame))
    }
  }, [])

  const stopLocalShare = useCallback(() => {
    localStreamRef.current?.getTracks().forEach((t) => t.stop())
    localStreamRef.current = null
    send({ type: 'present-stop' })
    teardownPC()
    setAnnouncement('Sharing stopped')
  }, [send, teardownPC])

  const publishLocalStream = useCallback(
    async (stream: MediaStream) => {
      localStreamRef.current = stream
      stream.getVideoTracks()[0]?.addEventListener('ended', () => {
        stopLocalShare()
      })
      const pc = ensurePC()
      for (const track of stream.getTracks()) {
        pc.addTrack(track, stream)
      }
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)
      send({ type: 'offer', sdp: offer.sdp })
      setAnnouncement('You are sharing your entire screen')
    },
    [ensurePC, send, stopLocalShare],
  )

  const requestPresent = useCallback(() => {
    send({ type: 'present-request' })
  }, [send])

  useEffect(() => {
    if (!enabled || !courseCode || !sessionId) return
    closedRef.current = false
    let timer: ReturnType<typeof setTimeout> | undefined

    const connect = () => {
      if (closedRef.current) return
      setConn(retryRef.current > 0 ? 'reconnecting' : 'connecting')
      const token = getAccessToken()
      if (!token) {
        setConn('error')
        setError('Not signed in')
        return
      }
      const ws = new WebSocket(wsURL(courseCode, sessionId))
      wsRef.current = ws
      ws.onopen = () => {
        ws.send(JSON.stringify({ authToken: token, role, joinToken }))
      }
      ws.onmessage = (ev) => {
        let frame: Record<string, unknown>
        try {
          frame = JSON.parse(String(ev.data)) as Record<string, unknown>
        } catch {
          return
        }
        const type = String(frame.type ?? '')
        switch (type) {
          case 'joined': {
            setConn('connected')
            retryRef.current = 0
            if (typeof frame.selfRole === 'string') setSelfRole(frame.selfRole as ScreenShareRole)
            if (frame.presenterId) {
              setPresenterId(String(frame.presenterId))
              setAnnouncement('A presenter is live')
            } else {
              setPresenterId(null)
            }
            break
          }
          case 'present-changed': {
            const pid = frame.presenterId == null ? null : String(frame.presenterId)
            setPresenterId(pid)
            setAnnouncement(pid ? 'Presenter changed' : 'Sharing ended')
            if (!pid) {
              teardownPC()
              onRemoteStreamRef.current?.(null)
            }
            break
          }
          case 'present-grant':
            setAnnouncement('You may present now')
            break
          case 'present-revoke':
            stopLocalShare()
            setAnnouncement('Presenting revoked')
            break
          case 'offer': {
            void (async () => {
              const pc = ensurePC()
              await pc.setRemoteDescription({ type: 'offer', sdp: String(frame.sdp ?? '') })
              const answer = await pc.createAnswer()
              await pc.setLocalDescription(answer)
              send({ type: 'answer', sdp: answer.sdp })
            })()
            break
          }
          case 'answer': {
            void (async () => {
              const pc = ensurePC()
              await pc.setRemoteDescription({ type: 'answer', sdp: String(frame.sdp ?? '') })
            })()
            break
          }
          case 'ice-candidate': {
            void (async () => {
              const pc = ensurePC()
              if (frame.candidate) {
                try {
                  await pc.addIceCandidate(frame.candidate as RTCIceCandidateInit)
                } catch {
                  /* ignore late candidates */
                }
              }
            })()
            break
          }
          case 'error': {
            const code = String(frame.code ?? '')
            setError(String(frame.message ?? code))
            if (code === 'ended') setConn('ended')
            else setConn('error')
            break
          }
          default:
            break
        }
      }
      ws.onclose = () => {
        teardownPC()
        if (closedRef.current) {
          setConn('disconnected')
          return
        }
        const delay = Math.min(30_000, 1000 * 2 ** retryRef.current)
        retryRef.current += 1
        setConn('reconnecting')
        timer = setTimeout(connect, delay)
      }
      ws.onerror = () => {
        ws.close()
      }
    }

    connect()
    return () => {
      closedRef.current = true
      if (timer) clearTimeout(timer)
      wsRef.current?.close()
      wsRef.current = null
      stopLocalShare()
    }
  }, [courseCode, sessionId, role, joinToken, enabled, ensurePC, send, stopLocalShare, teardownPC])

  const applyTurn = useCallback((turn: IceServersPayload) => {
    iceRef.current = turn.iceServers ?? []
  }, [])

  return {
    conn,
    presenterId,
    selfRole,
    error,
    announcement,
    requestPresent,
    publishLocalStream,
    stopLocalShare,
    applyTurn,
  }
}
