import { useTranslation } from 'react-i18next'
import type { AnswerAck } from '../../../lib/live-quiz-realtime'

export function ResultCard({
  ack,
  explanation,
  showCorrect,
}: {
  ack: AnswerAck | null
  explanation?: string | null
  showCorrect?: boolean
}) {
  const { t } = useTranslation('common')
  if (!ack?.ok && !ack?.alreadyAnswered && !ack?.duplicate) {
    if (ack?.late) {
      return (
        <div className="rounded-xl bg-amber-50 p-4 text-amber-900 dark:bg-amber-950/40 dark:text-amber-100" role="status" aria-live="assertive">
          {t('liveQuiz.answer.late')}
        </div>
      )
    }
    return null
  }

  const correct = !!ack.isCorrect
  const points = ack.points ?? 0
  const isPollLike = ack.isCorrect === false && points === 0 && !ack.duplicate

  return (
    <div
      className={
        correct
          ? 'rounded-xl bg-emerald-50 p-4 text-emerald-950 dark:bg-emerald-950/40 dark:text-emerald-50'
          : 'rounded-xl bg-slate-100 p-4 text-slate-900 dark:bg-neutral-800 dark:text-neutral-50'
      }
      role="status"
      aria-live="assertive"
    >
      <p className="text-lg font-semibold">
        {isPollLike
          ? t('liveQuiz.answer.received')
          : correct
            ? t('liveQuiz.answer.correct')
            : t('liveQuiz.answer.incorrect')}
      </p>
      {!isPollLike && (
        <p className="mt-1 text-2xl font-bold tabular-nums">
          {t('liveQuiz.answer.points', { points })}
        </p>
      )}
      {typeof ack.streak === 'number' && ack.streak > 0 && (
        <p className="mt-1 text-sm">{t('liveQuiz.answer.streak', { streak: ack.streak })}</p>
      )}
      {typeof ack.rank === 'number' && (
        <p className="mt-1 text-sm">{t('liveQuiz.answer.rank', { rank: ack.rank })}</p>
      )}
      {showCorrect && explanation ? (
        <p className="mt-3 text-sm opacity-90">{explanation}</p>
      ) : null}
    </div>
  )
}
