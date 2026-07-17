import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useEffect, useState } from 'react'
import { LiveQuizLeaderboard } from '../../components/live-quiz/leaderboard'
import { shapeForIndex } from '../../components/live-quiz/play/answer-shape-meta'
import { ShapeIcon } from '../../components/live-quiz/play/answer-shapes'
import { useLiveGame } from '../../lib/live-quiz-realtime'

function PresentCountdown({ deadline }: { deadline?: string }) {
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
  return (
    <p
      className="text-6xl font-bold tabular-nums"
      aria-live={left <= 5 ? 'assertive' : 'polite'}
      aria-atomic="true"
    >
      {left}
    </p>
  )
}

export default function LiveQuizPresentPage() {
  const { t } = useTranslation('common')
  const { courseCode: rawCode, gameId: rawGameId } = useParams<{
    courseCode: string
    gameId: string
  }>()
  const courseCode = rawCode ? decodeURIComponent(rawCode) : ''
  const gameId = rawGameId ? decodeURIComponent(rawGameId) : ''
  const [highContrast, setHighContrast] = useState(false)
  const { state, conn } = useLiveGame({
    courseCode,
    gameId,
    role: 'projector',
    enabled: !!courseCode && !!gameId,
  })

  const phase = state?.phase ?? 'lobby'
  const q = state?.question
  const correct = new Set(q?.correctOptionIds ?? [])
  const showCorrect =
    phase === 'question_reveal' || phase === 'leaderboard' || phase === 'podium'

  return (
    <div
      className={
        highContrast
          ? 'min-h-screen bg-black px-8 py-10 text-white motion-reduce:transition-none motion-reduce:animate-none'
          : 'min-h-screen bg-zinc-950 px-8 py-10 text-zinc-50 motion-reduce:transition-none motion-reduce:animate-none'
      }
      data-no-flash="true"
    >
      <div className="mb-4 flex justify-end">
        <label className="flex items-center gap-2 text-sm text-zinc-300">
          <input
            type="checkbox"
            checked={highContrast}
            onChange={(e) => setHighContrast(e.target.checked)}
          />
          {t('liveQuiz.a11y.highContrast')}
        </label>
      </div>
      <div className="mx-auto flex min-h-[80vh] max-w-5xl flex-col justify-center gap-8">
        {conn === 'reconnecting' && (
          <p className="text-amber-300" role="status">
            {t('liveQuiz.state.reconnecting')}
          </p>
        )}
        {phase === 'waiting_for_host' && (
          <p className="text-3xl" role="status">
            {t('liveQuiz.state.waitingForHost')}
          </p>
        )}

        {phase === 'lobby' && (
          <>
            <p className="text-xl text-zinc-400">{state?.kitTitle}</p>
            <h1 className="text-4xl font-semibold">{t('liveQuiz.present.joinHeadline')}</h1>
            <p className="text-xl text-zinc-300">{t('liveQuiz.present.joinHint')}</p>
            <p className="text-7xl font-bold tracking-[0.3em] tabular-nums">{state?.joinCode}</p>
            <p className="text-2xl text-zinc-400">
              {t('liveQuiz.host.players', { count: state?.players?.length ?? 0 })}
            </p>
          </>
        )}

        {(phase === 'question_open' ||
          phase === 'question_locked' ||
          phase === 'question_intro' ||
          phase === 'question_reveal') &&
          q && (
            <>
              <div className="flex items-start justify-between gap-6">
                <div>
                  <p className="text-lg text-zinc-400">
                    {t('liveQuiz.host.questionN', {
                      n: (state?.questionIndex ?? 0) + 1,
                      total: state?.questionCount ?? 0,
                    })}
                  </p>
                  <h1 className="mt-2 text-4xl font-semibold leading-tight">{q.prompt}</h1>
                </div>
                {phase === 'question_open' && <PresentCountdown deadline={state?.deadline} />}
              </div>
              <ul className="grid gap-3 sm:grid-cols-2">
                {q.options.map((opt, i) => {
                  const isCorrect = showCorrect && correct.has(opt.id)
                  const shape = shapeForIndex(i)
                  return (
                    <li
                      key={opt.id}
                      className={
                        isCorrect
                          ? `flex items-center gap-3 rounded-lg border-2 border-emerald-400 bg-emerald-950/40 px-4 py-5 ${highContrast ? 'text-3xl' : 'text-2xl'}`
                          : `flex items-center gap-3 rounded-lg border border-zinc-700 bg-zinc-900 px-4 py-5 ${highContrast ? 'text-3xl' : 'text-2xl'}`
                      }
                    >
                      <ShapeIcon shape={shape} className="h-8 w-8 shrink-0" />
                      <span>
                        {opt.text}
                        {showCorrect && state?.distribution?.[opt.id] != null && (
                          <span className="ms-3 text-lg text-zinc-400">
                            ({state.distribution[opt.id]})
                          </span>
                        )}
                      </span>
                    </li>
                  )
                })}
              </ul>
              {(phase === 'question_open' || phase === 'question_locked') && (
                <p className="text-xl text-zinc-400">
                  {t('liveQuiz.host.answerCount', { count: state?.answerCount ?? 0 })}
                </p>
              )}
            </>
          )}

        {(phase === 'podium' || phase === 'ended' || phase === 'leaderboard') && (
          <div className="w-full max-w-3xl space-y-6 text-start">
            <h1 className="text-center text-5xl font-semibold">
              {phase === 'leaderboard' ? t('liveQuiz.leaderboard.title') : t('liveQuiz.present.podium')}
            </h1>
            <div className="text-2xl">
              <LiveQuizLeaderboard
                rows={state?.leaderboard ?? state?.podium ?? []}
                privacy={state?.leaderboardPrivacy ?? 'names'}
                variant={phase === 'leaderboard' ? 'list' : 'podium'}
              />
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
