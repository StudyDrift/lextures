import { useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useEffect, useState } from 'react'
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
    <div className="min-h-screen bg-zinc-950 px-8 py-10 text-zinc-50 motion-reduce:transition-none">
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
                {q.options.map((opt) => {
                  const isCorrect = showCorrect && correct.has(opt.id)
                  return (
                    <li
                      key={opt.id}
                      className={
                        isCorrect
                          ? 'rounded-lg border-2 border-emerald-400 bg-emerald-950/40 px-4 py-5 text-2xl'
                          : 'rounded-lg border border-zinc-700 bg-zinc-900 px-4 py-5 text-2xl'
                      }
                    >
                      {opt.text}
                      {showCorrect && state?.distribution?.[opt.id] != null && (
                        <span className="ml-3 text-lg text-zinc-400">
                          ({state.distribution[opt.id]})
                        </span>
                      )}
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
          <>
            <h1 className="text-5xl font-semibold">{t('liveQuiz.present.podium')}</h1>
            <ol className="space-y-3 text-3xl">
              {(state?.leaderboard ?? []).map((row) => (
                <li key={row.playerId}>
                  #{row.rank} {row.nickname} — {row.totalScore}
                </li>
              ))}
            </ol>
          </>
        )}
      </div>
    </div>
  )
}
