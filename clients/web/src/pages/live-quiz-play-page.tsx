import { useEffect, useState, type FormEvent, type ReactNode } from 'react'
import { Link, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AnswerSurface } from '../components/live-quiz/play/answer-surface'
import { ConnectionBadge } from '../components/live-quiz/play/connection-badge'
import { CountdownRing } from '../components/live-quiz/play/countdown-ring'
import { ResultCard } from '../components/live-quiz/play/result-card'
import { StandingCard } from '../components/live-quiz/play/standing-card'
import { getAccessToken } from '../lib/auth'
import {
  joinLiveGame,
  joinLiveGameAsGuest,
  LiveQuizJoinError,
  lookupJoinCode,
  type JoinLookup,
} from '../lib/live-quiz-api'
import { validateNickname } from '../lib/live-quiz-nickname'
import {
  clearPlayerSession,
  loadPlayerSession,
  savePlayerSession,
} from '../lib/live-quiz-player-storage'
import { useLiveGame } from '../lib/live-quiz-realtime'

type Step = 'code' | 'nickname' | 'play'

export default function LiveQuizPlayPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { code: rawCode } = useParams<{ code?: string }>()
  const [searchParams] = useSearchParams()

  const [step, setStep] = useState<Step>(rawCode ? 'nickname' : 'code')
  const [codeInput, setCodeInput] = useState(rawCode ?? '')
  const [nickname, setNickname] = useState('')
  const [lookup, setLookup] = useState<JoinLookup | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const [playerId, setPlayerId] = useState<string | null>(null)
  const [playerToken, setPlayerToken] = useState<string | null>(null)
  const [courseCode, setCourseCode] = useState('')
  const [gameId, setGameId] = useState('')
  const [doubleOrNothing, setDoubleOrNothing] = useState(false)

  const game = useLiveGame({
    courseCode,
    gameId,
    role: 'player',
    playerToken: playerToken ?? undefined,
    playerId: playerId ?? undefined,
    enabled: step === 'play' && !!playerToken && !!courseCode && !!gameId,
  })

  useEffect(() => {
    if (!rawCode) return
    void (async () => {
      setBusy(true)
      setError(null)
      try {
        const info = await lookupJoinCode(rawCode.replace(/\D/g, '').slice(0, 12))
        setLookup(info)
        setCourseCode(info.courseCode)
        setGameId(info.gameId)
        const stored = loadPlayerSession(info.gameId)
        if (stored?.playerToken) {
          setPlayerId(stored.playerId)
          setPlayerToken(stored.playerToken)
          setNickname(stored.nickname)
          setStep('play')
        } else {
          setStep('nickname')
        }
      } catch (err) {
        setError(mapJoinError(err, t))
        setStep('code')
      } finally {
        setBusy(false)
      }
    })()
  }, [rawCode, t])

  async function onLookupCode(e: FormEvent) {
    e.preventDefault()
    const code = codeInput.replace(/\D/g, '').slice(0, 12)
    if (!code) {
      setError(t('liveQuiz.play.errorCodeRequired'))
      return
    }
    setBusy(true)
    setError(null)
    try {
      const info = await lookupJoinCode(code)
      setLookup(info)
      setCourseCode(info.courseCode)
      setGameId(info.gameId)
      navigate(`/play/${code}`, { replace: true })
      setStep('nickname')
    } catch (err) {
      setError(mapJoinError(err, t))
    } finally {
      setBusy(false)
    }
  }

  async function onJoin(e: FormEvent) {
    e.preventDefault()
    if (!lookup) return
    const check = validateNickname(nickname)
    if (!check.ok) {
      setError(
        check.reason === 'too_long'
          ? t('liveQuiz.play.errorNicknameLong')
          : check.reason === 'charset'
            ? t('liveQuiz.play.errorNicknameCharset')
            : t('liveQuiz.play.errorNicknameRequired'),
      )
      return
    }
    const joinCode = (codeInput || rawCode || '').replace(/\D/g, '')
    const useGuest = lookup.allowsGuests && !getAccessToken()
    if (lookup.requiresAuth && !useGuest && !getAccessToken()) {
      const from = `/play/${joinCode}`
      navigate('/login', { state: { from } })
      return
    }
    setBusy(true)
    setError(null)
    try {
      const res = useGuest
        ? await joinLiveGameAsGuest(joinCode, check.nickname)
        : await joinLiveGame(lookup.courseCode, lookup.gameId, check.nickname)
      setPlayerId(res.playerId)
      setPlayerToken(res.playerToken)
      setNickname(res.nickname)
      savePlayerSession({
        gameId: lookup.gameId,
        courseCode: lookup.courseCode,
        playerId: res.playerId,
        playerToken: res.playerToken,
        nickname: res.nickname,
        joinCode,
      })
      setStep('play')
    } catch (err) {
      setError(mapJoinError(err, t))
    } finally {
      setBusy(false)
    }
  }

  if (game.kicked || game.conn === 'kicked') {
    if (gameId) clearPlayerSession(gameId)
    return (
      <Shell>
        <h1 className="text-2xl font-semibold">{t('liveQuiz.play.kickedTitle')}</h1>
        <p className="mt-3">{t('liveQuiz.play.kicked')}</p>
        <Link to="/play" className="mt-6 inline-block text-indigo-600 underline">
          {t('liveQuiz.play.joinAnother')}
        </Link>
      </Shell>
    )
  }

  const phase = game.state?.phase
  const q = game.state?.question
  const locked =
    game.hasAnsweredCurrent ||
    phase === 'question_locked' ||
    phase === 'question_reveal' ||
    phase === 'leaderboard' ||
    phase === 'podium' ||
    phase === 'ended'
  const showResult =
    game.hasAnsweredCurrent ||
    phase === 'question_reveal' ||
    phase === 'leaderboard' ||
    phase === 'podium'
  const showStanding = phase === 'podium' || phase === 'ended'

  return (
    <Shell>
      <div className="mb-4 flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-medium text-slate-500 dark:text-neutral-400">
            {t('liveQuiz.play.brand')}
          </p>
          <h1 className="text-2xl font-semibold">
            {lookup?.kitTitle || game.state?.kitTitle || t('liveQuiz.play.title')}
          </h1>
          {nickname && step === 'play' && (
            <p className="mt-1 text-sm text-slate-600 dark:text-neutral-300">
              {t('liveQuiz.play.playingAs', { nickname })}
            </p>
          )}
        </div>
        {step === 'play' && <ConnectionBadge conn={game.conn} />}
      </div>

      {error && (
        <p className="mb-4 rounded-md bg-red-50 px-3 py-2 text-sm text-red-800 dark:bg-red-950/40 dark:text-red-100" role="alert">
          {error}
        </p>
      )}

      {step === 'code' && (
        <form onSubmit={onLookupCode} className="space-y-4">
          <label className="block text-sm font-medium">
            {t('liveQuiz.play.codeLabel')}
            <input
              inputMode="numeric"
              autoComplete="one-time-code"
              value={codeInput}
              onChange={(e) => setCodeInput(e.target.value.replace(/\D/g, '').slice(0, 12))}
              className="mt-1 min-h-14 w-full rounded-xl border border-slate-300 bg-white px-4 text-center text-3xl tracking-[0.35em] tabular-nums dark:border-neutral-700 dark:bg-neutral-900"
              placeholder="000000"
              aria-describedby="join-code-hint"
            />
          </label>
          <p id="join-code-hint" className="text-sm text-slate-500 dark:text-neutral-400">
            {t('liveQuiz.play.codeHint')}
          </p>
          <button
            type="submit"
            disabled={busy}
            className="min-h-12 w-full rounded-xl bg-indigo-600 px-4 py-3 text-base font-semibold text-white disabled:opacity-50"
          >
            {t('liveQuiz.play.continue')}
          </button>
        </form>
      )}

      {step === 'nickname' && (
        <form onSubmit={onJoin} className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-neutral-300">
            {t('liveQuiz.play.nicknameIntro', { title: lookup?.kitTitle ?? '' })}
          </p>
          <label className="block text-sm font-medium">
            {t('liveQuiz.play.nicknameLabel')}
            <input
              value={nickname}
              onChange={(e) => setNickname(e.target.value)}
              maxLength={24}
              autoComplete="nickname"
              className="mt-1 min-h-12 w-full rounded-xl border border-slate-300 bg-white px-3 text-lg dark:border-neutral-700 dark:bg-neutral-900"
            />
          </label>
          {lookup?.requiresAuth && !getAccessToken() && (
            <p className="text-sm text-amber-800 dark:text-amber-200">{t('liveQuiz.play.authRequired')}</p>
          )}
          <button
            type="submit"
            disabled={busy}
            className="min-h-12 w-full rounded-xl bg-indigo-600 px-4 py-3 text-base font-semibold text-white disabled:opacity-50"
          >
            {t('liveQuiz.play.join')}
          </button>
        </form>
      )}

      {step === 'play' && (
        <div className="space-y-5">
          {(phase === 'lobby' || phase === 'question_intro' || (!phase && game.conn === 'connecting')) && (
            <div className="rounded-xl bg-indigo-50 p-6 text-center dark:bg-indigo-950/40" role="status">
              <p className="text-xl font-semibold">{t('liveQuiz.play.lobbyTitle')}</p>
              <p className="mt-2 text-slate-600 dark:text-neutral-300">{t('liveQuiz.play.lobbyHint')}</p>
            </div>
          )}

          {phase === 'waiting_for_host' && (
            <p className="text-center text-lg" role="status">
              {t('liveQuiz.state.waitingForHost')}
            </p>
          )}

          {(phase === 'question_open' ||
            phase === 'question_locked' ||
            phase === 'question_reveal') &&
            q && (
              <>
                <div className="flex items-start justify-between gap-4">
                  <div className="min-w-0 flex-1">
                    <p className="text-sm text-slate-500 dark:text-neutral-400">
                      {t('liveQuiz.host.questionN', {
                        n: (game.state?.questionIndex ?? 0) + 1,
                        total: game.state?.questionCount ?? 0,
                      })}
                    </p>
                    <h2 className="mt-1 text-xl font-semibold leading-snug" aria-live="polite">
                      {q.prompt}
                    </h2>
                  </div>
                  {phase === 'question_open' && !game.hasAnsweredCurrent && (
                    <CountdownRing
                      deadline={game.state?.deadline}
                      timeLimitSeconds={q.timeLimitSeconds}
                    />
                  )}
                </div>

                {game.hasAnsweredCurrent && phase === 'question_open' && (
                  <p className="rounded-xl bg-slate-100 px-4 py-3 text-center font-medium dark:bg-neutral-800" role="status">
                    {t('liveQuiz.answer.received')}
                  </p>
                )}

                {phase === 'question_open' && !game.hasAnsweredCurrent && (
                  <>
                    {game.state?.powerUpsEnabled &&
                      q.pointsStyle !== 'no_points' &&
                      q.questionType !== 'poll' &&
                      q.questionType !== 'word_cloud' && (
                        <label className="flex items-center gap-2 rounded-lg border px-3 py-2 text-sm">
                          <input
                            type="checkbox"
                            checked={doubleOrNothing}
                            onChange={(e) => setDoubleOrNothing(e.target.checked)}
                          />
                          {t('liveQuiz.powerup.doubleOrNothing')}
                        </label>
                      )}
                    <AnswerSurface
                      key={`${game.state?.questionIndex}-${q.questionType}`}
                      questionType={q.questionType}
                      options={q.options}
                      locked={locked}
                      onAnswer={(payload) => {
                        game.submitAnswer(game.state!.questionIndex, payload, {
                          powerUp: doubleOrNothing ? 'double_or_nothing' : undefined,
                        })
                        setDoubleOrNothing(false)
                      }}
                    />
                  </>
                )}

                {showResult && (
                  <ResultCard
                    ack={game.lastAck}
                    explanation={q.explanation}
                    showCorrect={phase === 'question_reveal'}
                  />
                )}
              </>
            )}

          {(phase === 'leaderboard' || showStanding) && (
            <StandingCard
              playerId={playerId ?? undefined}
              nickname={nickname}
              leaderboard={game.state?.leaderboard}
              totalScore={
                game.lastAck?.totalScore ??
                game.state?.players?.find((p) => p.id === playerId)?.totalScore
              }
            />
          )}

          {searchParams.get('debug') === '1' && (
            <pre className="overflow-auto text-xs opacity-60">{JSON.stringify({ phase, conn: game.conn }, null, 2)}</pre>
          )}
        </div>
      )}
    </Shell>
  )
}

function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="min-h-screen bg-gradient-to-b from-slate-50 to-indigo-50 px-4 py-8 text-slate-900 dark:from-neutral-950 dark:to-neutral-900 dark:text-neutral-50">
      <div className="mx-auto w-full max-w-lg">{children}</div>
    </div>
  )
}

function mapJoinError(err: unknown, t: (key: string) => string): string {
  if (err instanceof LiveQuizJoinError) {
    switch (err.message) {
      case 'not_found':
        return t('liveQuiz.play.errorNotFound')
      case 'rate_limited':
        return t('liveQuiz.play.errorRateLimited')
      case 'nickname_taken':
        return t('liveQuiz.play.errorNicknameTaken')
      case 'nickname_invalid':
        return t('liveQuiz.play.errorNicknameCharset')
      case 'nickname_denied':
        return t('liveQuiz.moderation.nicknameDenied')
      case 'lobby_locked':
        return t('liveQuiz.safety.lobbyLocked')
      case 'banned':
        return t('liveQuiz.safety.banned')
      case 'one_session':
        return t('liveQuiz.safety.oneSession')
      case 'game_ended':
        return t('liveQuiz.play.errorGameEnded')
      case 'auth_required':
        return t('liveQuiz.play.authRequired')
      default:
        return t('liveQuiz.play.errorJoin')
    }
  }
  return t('liveQuiz.play.errorJoin')
}
