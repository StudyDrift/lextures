import { useEffect, useRef, useState } from 'react'
import { getAccessToken } from './auth'

const apiBase = import.meta.env.VITE_API_URL ?? ''

export type LiveGameRole = 'host' | 'projector' | 'player'

export type LiveGamePhase =
  | 'lobby'
  | 'question_intro'
  | 'question_open'
  | 'question_locked'
  | 'question_reveal'
  | 'leaderboard'
  | 'podium'
  | 'ended'
  | 'waiting_for_host'

export type LiveGameStateFrame = {
  type: 'state'
  seq: number
  gameId: string
  phase: LiveGamePhase
  status: string
  questionIndex: number
  joinCode: string
  kitTitle: string
  pacing: string
  players: Array<{
    id: string
    nickname: string
    totalScore: number
    streak: number
    connected: boolean
    renamedByHost?: boolean
    isGuest?: boolean
  }>
  questionCount: number
  namesMuted?: boolean
  lobbyLocked?: boolean
  allowGuests?: boolean
  openedAt?: string
  deadline?: string
  answerCount?: number
  distribution?: Record<string, number>
  leaderboard?: Array<{
    rank: number
    playerId: string
    nickname: string
    totalScore: number
  }>
  leaderboardPrivacy?: 'names' | 'nicknames' | 'hidden'
  podium?: Array<{
    rank: number
    playerId: string
    nickname: string
    totalScore: number
  }>
  you?: { rank: number; totalScore: number; streak: number }
  scoringProfile?: string
  powerUpsEnabled?: boolean
  question?: {
    index: number
    questionType: string
    prompt: string
    options: Array<{ id: string; text: string }>
    timeLimitSeconds: number
    pointsStyle: string
    correctOptionIds?: string[]
    explanation?: string | null
  }
}

export type PointsBreakdown = {
  base: number
  speedBonus: number
  streakBonus: number
  styleMultiplier: number
  powerUp?: string
  powerUpFactor: number
  total: number
}

export type AnswerAck = {
  type: 'answer_ack'
  ok: boolean
  questionIndex?: number
  isCorrect?: boolean
  points?: number
  pointsBreakdown?: PointsBreakdown
  responseMs?: number
  streak?: number
  totalScore?: number
  rank?: number
  duplicate?: boolean
  late?: boolean
  alreadyAnswered?: boolean
  error?: string
}

export type ConnStatus = 'connecting' | 'connected' | 'reconnecting' | 'ended' | 'kicked' | 'disconnected'

export type LiveAnswerPayload =
  | { optionId: string }
  | { optionIds: string[] }
  | { text: string }
  | { value: number }
  | { order: string[] }

export type LivePowerUpKind = 'double_or_nothing' | 'shield'

function wsURL(courseCode: string, gameId: string): string {
  const base = apiBase || (typeof window !== 'undefined' ? window.location.origin : '')
  const u = new URL(
    `/api/v1/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/ws`,
    base,
  )
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:'
  return u.toString()
}

export type UseLiveGameOpts = {
  courseCode: string
  gameId: string
  role: LiveGameRole
  playerToken?: string
  playerId?: string
  enabled?: boolean
}

export function useLiveGame(opts: UseLiveGameOpts) {
  const { courseCode, gameId, role, playerToken, playerId, enabled = true } = opts
  const [state, setState] = useState<LiveGameStateFrame | null>(null)
  const [conn, setConn] = useState<ConnStatus>('connecting')
  const [pending, setPending] = useState(false)
  const [lastAck, setLastAck] = useState<AnswerAck | null>(null)
  const [kicked, setKicked] = useState(false)
  const wsRef = useRef<WebSocket | null>(null)
  const seqRef = useRef(0)
  const retryRef = useRef(0)
  const closedRef = useRef(false)
  const answeredIndexRef = useRef<number | null>(null)
  const statePhaseRef = useRef<LiveGamePhase | null>(null)
  const questionIndexRef = useRef<number | null>(null)
  const kickedRef = useRef(false)

  useEffect(() => {
    if (!enabled || !courseCode || !gameId) return
    if (role === 'player' && !playerToken) return
    closedRef.current = false
    let timer: ReturnType<typeof setTimeout> | undefined

    function connect() {
      if (closedRef.current || kickedRef.current) return
      setConn(retryRef.current > 0 ? 'reconnecting' : 'connecting')
      const tok = getAccessToken()
      // Guest players (IQ.9) connect with playerToken only.
      if (!tok && !(role === 'player' && playerToken)) {
        timer = setTimeout(connect, 1500)
        return
      }
      const ws = new WebSocket(wsURL(courseCode, gameId))
      wsRef.current = ws
      ws.onopen = () => {
        ws.send(
          JSON.stringify({
            authToken: tok || '',
            role,
            playerToken: playerToken || undefined,
          }),
        )
        if (seqRef.current > 0) {
          ws.send(JSON.stringify({ type: 'catchup', afterSeq: seqRef.current }))
        } else if (role === 'player') {
          ws.send(JSON.stringify({ type: 'hello', resumeSeq: 0 }))
        }
      }
      ws.onmessage = (ev) => {
        let msg: { type?: string; seq?: number; playerId?: string } & Partial<LiveGameStateFrame> &
          Partial<AnswerAck>
        try {
          msg = JSON.parse(String(ev.data)) as typeof msg
        } catch {
          return
        }
        if (msg.type === 'kicked') {
          kickedRef.current = true
          setKicked(true)
          setConn('kicked')
          closedRef.current = true
          ws.close()
          return
        }
        if (msg.type === 'answer_ack') {
          const ack = msg as AnswerAck
          setLastAck(ack)
          if (ack.ok && typeof ack.questionIndex === 'number') {
            answeredIndexRef.current = ack.questionIndex
          }
          setPending(false)
          return
        }
        if (msg.type === 'state') {
          const frame = msg as LiveGameStateFrame
          if (typeof frame.seq === 'number') seqRef.current = frame.seq
          if (
            questionIndexRef.current != null &&
            frame.questionIndex !== questionIndexRef.current &&
            frame.phase === 'question_open'
          ) {
            answeredIndexRef.current = null
            setLastAck(null)
          }
          questionIndexRef.current = frame.questionIndex
          statePhaseRef.current = frame.phase
          setState(frame)
          setPending(false)
          setConn(frame.phase === 'ended' ? 'ended' : 'connected')
          retryRef.current = 0
        }
      }
      ws.onclose = () => {
        if (closedRef.current || kickedRef.current) return
        if (statePhaseRef.current === 'ended') {
          setConn('ended')
          return
        }
        retryRef.current += 1
        const delay = Math.min(8000, 500 * 2 ** Math.min(retryRef.current, 4))
        setConn('reconnecting')
        timer = setTimeout(connect, delay)
      }
      ws.onerror = () => {
        ws.close()
      }
    }

    connect()

    function onVisibility() {
      if (document.visibilityState !== 'visible') return
      const ws = wsRef.current
      if (ws && ws.readyState === WebSocket.OPEN && seqRef.current > 0) {
        ws.send(JSON.stringify({ type: 'catchup', afterSeq: seqRef.current }))
      }
    }
    document.addEventListener('visibilitychange', onVisibility)

    return () => {
      closedRef.current = true
      document.removeEventListener('visibilitychange', onVisibility)
      if (timer) clearTimeout(timer)
      wsRef.current?.close()
      wsRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- reconnect on identity only
  }, [courseCode, gameId, role, playerToken, enabled])

  function send(type: string, extra?: Record<string, unknown>) {
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    setPending(true)
    ws.send(JSON.stringify({ type, ...extra }))
  }

  function submitAnswer(
    questionIndex: number,
    answer: LiveAnswerPayload,
    opts?: { powerUp?: LivePowerUpKind },
  ) {
    if (answeredIndexRef.current === questionIndex) return
    if (conn === 'kicked' || conn === 'ended') return
    answeredIndexRef.current = questionIndex
    send('answer', {
      questionIndex,
      answer,
      powerUp: opts?.powerUp,
      clientSentAt: new Date().toISOString(),
    })
  }

  function claimPowerUp(questionIndex: number, kind: LivePowerUpKind) {
    if (conn === 'kicked' || conn === 'ended') return
    const ws = wsRef.current
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({ type: 'powerup', questionIndex, kind }))
  }

  const hasAnsweredCurrent =
    state != null &&
    answeredIndexRef.current === state.questionIndex &&
    (state.phase === 'question_open' ||
      state.phase === 'question_locked' ||
      state.phase === 'question_reveal')

  return {
    state,
    conn,
    pending,
    lastAck,
    kicked,
    playerId,
    hasAnsweredCurrent,
    submitAnswer,
    claimPowerUp,
    open: () => send('open'),
    lock: () => send('lock'),
    reveal: () => send('reveal'),
    next: () => send('next'),
    skip: () => send('skip'),
    pause: () => send('pause'),
    resume: () => send('resume'),
    end: () => send('end'),
    kick: (id: string) => send('kick', { playerId: id }),
    ban: (id: string) => send('ban', { playerId: id }),
    rename: (id: string, nickname?: string) => send('rename', { playerId: id, nickname }),
    muteNames: (muted: boolean) => send('mute_names', { muted }),
    lockLobby: (locked: boolean) => send('lock_lobby', { locked }),
  }
}
