import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { HostSafetyPanel } from '../../components/live-quiz/host-safety-panel'
import { LiveQuizLeaderboard } from '../../components/live-quiz/leaderboard'
import {
  assignLiveGameTeams,
  endLiveGame,
  fetchLiveGame,
  startPacedLiveGame,
  type LiveGame,
} from '../../lib/live-quiz-api'
import { useLiveGame } from '../../lib/live-quiz-realtime'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { toastMutationError } from '../../lib/lms-toast'
import { LmsPage } from './lms-page'

function Countdown({ deadline }: { deadline?: string }) {
  const [left, setLeft] = useState<number | null>(null)
  useEffect(() => {
    if (!deadline) {
      setLeft(null)
      return
    }
    const tick = () => {
      const ms = new Date(deadline).getTime() - Date.now()
      setLeft(Math.max(0, Math.ceil(ms / 1000)))
    }
    tick()
    const id = setInterval(tick, 250)
    return () => clearInterval(id)
  }, [deadline])
  if (left == null) return null
  const assertive = left <= 5
  return (
    <p
      className="text-3xl font-semibold tabular-nums"
      aria-live={assertive ? 'assertive' : 'polite'}
      aria-atomic="true"
    >
      {left}s
    </p>
  )
}

export default function LiveQuizHostPage() {
  const { t } = useTranslation('common')
  const navigate = useNavigate()
  const { courseCode: rawCode, gameId: rawGameId } = useParams<{
    courseCode: string
    gameId: string
  }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const gameId = rawGameId ? decodeURIComponent(rawGameId) : ''
  const { ffInteractiveQuizzes, ffIqLiveHosting } = usePlatformFeatures()
  const [bootstrap, setBootstrap] = useState<LiveGame | null>(null)
  const [error, setError] = useState<string | null>(null)

  const game = useLiveGame({ courseCode, gameId, role: 'host', enabled: !!courseCode && !!gameId })

  const load = useCallback(async () => {
    if (!courseCode || !gameId) return
    try {
      if (!ffInteractiveQuizzes || !ffIqLiveHosting) {
        setError(t('liveQuiz.error.disabled'))
        return
      }
      setBootstrap(await fetchLiveGame(courseCode, gameId))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('liveQuiz.host.errorLoad'))
    }
  }, [courseCode, gameId, ffInteractiveQuizzes, ffIqLiveHosting, t])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.code !== 'Space' || e.repeat) return
      const tag = (e.target as HTMLElement | null)?.tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'BUTTON') return
      e.preventDefault()
      const phase = game.state?.phase
      if (!phase || game.pending) return
      if (phase === 'lobby' || phase === 'question_intro') game.open()
      else if (phase === 'question_open') game.lock()
      else if (phase === 'question_locked') game.reveal()
      else if (phase === 'question_reveal' || phase === 'leaderboard') game.next()
      else if (phase === 'podium') void handleEnd()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [game.state?.phase, game.pending])

  async function handleEnd() {
    try {
      game.end()
      await endLiveGame(courseCode, gameId)
      navigate(
        `/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/report`,
      )
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleAutoBalanceTeams() {
    try {
      await assignLiveGameTeams(courseCode, gameId, { autoBalance: true })
      await load()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  async function handleStartPaced() {
    try {
      await startPacedLiveGame(courseCode, gameId)
      await load()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : String(err))
    }
  }

  const state = game.state
  const phase = state?.phase ?? bootstrap?.phase ?? 'lobby'
  const mode = bootstrap?.mode ?? 'live_classic'
  const joinCode = state?.joinCode || bootstrap?.joinCode || ''
  const players = state?.players ?? bootstrap?.players ?? []
  const kitTitle = state?.kitTitle || bootstrap?.kitTitle || ''
  const presentPath = `/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/present`

  if (error) {
    return (
      <LmsPage title={t('liveQuiz.host.title')}>
        <p className="text-destructive">{error}</p>
        <Link to={`/courses/${encodeURIComponent(courseCode)}/live-quizzes`} className="underline">
          {t('liveQuiz.kit.backToGallery')}
        </Link>
      </LmsPage>
    )
  }

  return (
    <LmsPage title={t('liveQuiz.host.title')}>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <p className="text-sm text-muted-foreground">{kitTitle}</p>
          <h1 className="text-2xl font-semibold">{t('liveQuiz.host.title')}</h1>
          {(game.conn === 'reconnecting' || phase === 'waiting_for_host') && (
            <p className="mt-1 text-amber-700 dark:text-amber-300" role="status">
              {game.conn === 'reconnecting'
                ? t('liveQuiz.state.reconnecting')
                : t('liveQuiz.state.waitingForHost')}
            </p>
          )}
        </div>
        <div className="flex flex-wrap gap-2">
          <Link
            to={presentPath}
            target="_blank"
            rel="noreferrer"
            className="rounded-md border px-3 py-2 text-sm"
          >
            {t('liveQuiz.host.openPresent')}
          </Link>
          <button
            type="button"
            className="rounded-md border border-destructive px-3 py-2 text-sm text-destructive"
            onClick={() => void handleEnd()}
          >
            {t('liveQuiz.host.endGame')}
          </button>
        </div>
      </div>

      {phase === 'lobby' && (
        <section className="space-y-4">
          <div>
            <p className="text-sm text-muted-foreground">{t('liveQuiz.host.joinCodeLabel')}</p>
            <p className="text-5xl font-bold tracking-widest tabular-nums" aria-live="polite">
              {joinCode}
            </p>
            {joinCode ? (
              <p className="mt-2 text-sm text-muted-foreground">
                {t('liveQuiz.host.playerJoinUrl')}{' '}
                <Link className="text-primary underline" to={`/play/${joinCode}`}>
                  /play/{joinCode}
                </Link>
              </p>
            ) : null}
          </div>
          <HostSafetyPanel
            courseCode={courseCode}
            gameId={gameId}
            players={players}
            namesMuted={!!state?.namesMuted || !!bootstrap?.namesMuted}
            lobbyLocked={!!state?.lobbyLocked || !!bootstrap?.lobbyLocked}
            onKick={(id) => game.kick(id)}
            onRename={(id) => game.rename(id)}
            onMuteNames={(muted) => game.muteNames(muted)}
            onLockLobby={(locked) => game.lockLobby(locked)}
          />
          {mode === 'team' ? (
            <button
              type="button"
              className="rounded-md border px-4 py-2 text-sm disabled:opacity-50"
              disabled={players.length === 0}
              onClick={() => void handleAutoBalanceTeams()}
            >
              {t('liveQuiz.team.autoBalance', { defaultValue: 'Auto-balance teams' })}
            </button>
          ) : null}
          {mode === 'student_paced' ? (
            <button
              type="button"
              className="rounded-md bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50"
              disabled={players.length === 0}
              onClick={() => void handleStartPaced()}
            >
              {t('liveQuiz.paced.start', { defaultValue: 'Start student-paced game' })}
            </button>
          ) : (
            <button
              type="button"
              className="rounded-md bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50"
              disabled={game.pending || players.length === 0}
              onClick={() => game.open()}
            >
              {t('liveQuiz.host.start')}
            </button>
          )}
        </section>
      )}

      {phase !== 'lobby' &&
        phase !== 'ended' &&
        phase !== 'podium' &&
        phase !== 'leaderboard' && (
        <section className="space-y-4">
          <div className="flex flex-wrap items-end justify-between gap-4">
            <div>
              <p className="text-sm text-muted-foreground">
                {t('liveQuiz.host.questionN', {
                  n: (state?.questionIndex ?? 0) + 1,
                  total: state?.questionCount ?? 0,
                })}
              </p>
              <h2 className="text-xl font-semibold">{state?.question?.prompt}</h2>
              <p className="text-sm text-muted-foreground">
                {t('liveQuiz.host.answerCount', { count: state?.answerCount ?? 0 })}
              </p>
            </div>
            <Countdown deadline={state?.deadline} />
          </div>
          <div className="flex flex-wrap gap-2">
            {phase === 'question_open' && (
              <button
                type="button"
                className="rounded-md border px-3 py-2 text-sm disabled:opacity-50"
                disabled={game.pending}
                onClick={() => game.lock()}
              >
                {t('liveQuiz.host.lock')}
              </button>
            )}
            {(phase === 'question_locked' || phase === 'question_open') && (
              <button
                type="button"
                className="rounded-md border px-3 py-2 text-sm disabled:opacity-50"
                disabled={game.pending}
                onClick={() => game.reveal()}
              >
                {t('liveQuiz.host.reveal')}
              </button>
            )}
            {phase === 'question_reveal' && (
              <button
                type="button"
                className="rounded-md bg-primary px-3 py-2 text-sm text-primary-foreground disabled:opacity-50"
                disabled={game.pending}
                onClick={() => game.next()}
              >
                {t('liveQuiz.host.next')}
              </button>
            )}
            <button
              type="button"
              className="rounded-md border px-3 py-2 text-sm disabled:opacity-50"
              disabled={game.pending}
              onClick={() => game.skip()}
            >
              {t('liveQuiz.host.skip')}
            </button>
          </div>
          {phase === 'question_reveal' && state?.question?.correctOptionIds && (
            <p className="text-sm">
              {t('liveQuiz.host.correct')}: {state.question.correctOptionIds.join(', ')}
            </p>
          )}
          <HostSafetyPanel
            courseCode={courseCode}
            gameId={gameId}
            players={players}
            namesMuted={!!state?.namesMuted || !!bootstrap?.namesMuted}
            lobbyLocked={!!state?.lobbyLocked || !!bootstrap?.lobbyLocked}
            onKick={(id) => game.kick(id)}
            onRename={(id) => game.rename(id)}
            onMuteNames={(muted) => game.muteNames(muted)}
            onLockLobby={(locked) => game.lockLobby(locked)}
          />
        </section>
      )}

      {(phase === 'leaderboard' || phase === 'podium' || phase === 'ended') && (
        <section className="space-y-3">
          <h2 className="text-xl font-semibold">
            {phase === 'leaderboard' ? t('liveQuiz.leaderboard.title') : t('liveQuiz.host.podium')}
          </h2>
          <LiveQuizLeaderboard
            rows={state?.leaderboard ?? state?.podium ?? []}
            privacy={state?.leaderboardPrivacy ?? 'names'}
            variant={phase === 'leaderboard' ? 'list' : 'podium'}
          />
          {phase === 'leaderboard' && (
            <button
              type="button"
              className="rounded-md bg-primary px-4 py-2 text-primary-foreground disabled:opacity-50"
              disabled={game.pending}
              onClick={() => game.next()}
            >
              {t('liveQuiz.host.next')}
            </button>
          )}
          {phase === 'podium' && (
            <button
              type="button"
              className="rounded-md bg-primary px-4 py-2 text-primary-foreground"
              onClick={() => void handleEnd()}
            >
              {t('liveQuiz.host.endGame')}
            </button>
          )}
          {phase === 'ended' && (
            <div className="flex flex-wrap gap-3">
              <button
                type="button"
                className="rounded-md bg-primary px-4 py-2 text-primary-foreground"
                onClick={() =>
                  navigate(
                    `/courses/${encodeURIComponent(courseCode)}/live-quizzes/games/${encodeURIComponent(gameId)}/report`,
                  )
                }
              >
                {t('liveQuiz.report.viewReport')}
              </button>
              <button
                type="button"
                className="underline"
                onClick={() => navigate(`/courses/${encodeURIComponent(courseCode)}/live-quizzes`)}
              >
                {t('liveQuiz.kit.backToGallery')}
              </button>
            </div>
          )}
        </section>
      )}
    </LmsPage>
  )
}
