import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import type { AnswerAck } from '../../../lib/live-quiz-realtime'
import { quizAnswerFeedbackClass } from '../../../lib/delight-motion'
import { usePrefersReducedMotion } from '../../../lib/motion'
import { usePlatformFeatures } from '../../../context/platform-features-context'
import { useHaptics } from '../../../lib/control-motion'
import { DelightMoment } from '../../ui/delight-moment'

export function ResultCard({
  ack,
  explanation,
  showCorrect,
  seriousContext,
}: {
  ack: AnswerAck | null
  explanation?: string | null
  showCorrect?: boolean
  /** Exam / proctored / reduced-distraction — suppress flourish (FR-8). */
  seriousContext?: boolean
}) {
  const { t } = useTranslation('common')
  const { ffMotionDelight } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const { trigger } = useHaptics()

  const lateOnly = Boolean(ack?.late && !ack?.ok && !ack?.alreadyAnswered && !ack?.duplicate)
  const visible = Boolean(ack && (ack.ok || ack.alreadyAnswered || ack.duplicate))
  const correct = Boolean(ack?.isCorrect)
  const points = ack?.points ?? 0
  const bd = ack?.pointsBreakdown
  const isPollLike = Boolean(
    ack && ack.isCorrect === false && points === 0 && !ack.duplicate && bd == null,
  )

  useEffect(() => {
    if (!visible || isPollLike || lateOnly) return
    // Non-blocking haptic (FR-3 / FR-7) — never delays advance.
    trigger(correct ? 'success' : 'error')
  }, [visible, isPollLike, lateOnly, correct, trigger])

  if (lateOnly) {
    return (
      <div
        className="rounded-xl bg-amber-50 p-4 text-amber-900 dark:bg-amber-950/40 dark:text-amber-100"
        role="status"
        aria-live="assertive"
      >
        {t('liveQuiz.answer.late')}
      </div>
    )
  }

  if (!visible || !ack) return null

  const feedbackClass = isPollLike
    ? ''
    : quizAnswerFeedbackClass(correct ? 'correct' : 'incorrect', {
        enabled: ffMotionDelight !== false,
        reduceMotion,
        seriousContext,
      })

  const card = (
    <div
      className={
        (correct
          ? 'rounded-xl bg-emerald-50 p-4 text-emerald-950 dark:bg-emerald-950/40 dark:text-emerald-50'
          : 'rounded-xl bg-slate-100 p-4 text-slate-900 dark:bg-neutral-800 dark:text-neutral-50') +
        (feedbackClass ? ` ${feedbackClass}` : '')
      }
      role="status"
      aria-live="assertive"
      data-testid="quiz-answer-feedback"
      data-feedback={isPollLike ? 'received' : correct ? 'correct' : 'incorrect'}
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
      {bd && (bd.total > 0 || correct) ? (
        <p className="mt-2 text-sm opacity-90" aria-label={t('liveQuiz.score.breakdownAria')}>
          {t('liveQuiz.score.breakdown', {
            base: bd.base,
            speed: bd.speedBonus,
            streak: bd.streakBonus,
            total: bd.total,
          })}
          {bd.styleMultiplier === 2 ? ` ${t('liveQuiz.score.doubleApplied')}` : ''}
          {bd.powerUp === 'double_or_nothing' ? ` ${t('liveQuiz.powerup.donApplied')}` : ''}
        </p>
      ) : null}
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

  if (isPollLike || !correct) return card

  return (
    <DelightMoment
      active
      kind="correct"
      // Card already has assertive status text; keep burst without duplicating copy.
      announcement=""
      seriousContext={seriousContext}
      showBurst={!seriousContext}
    >
      {card}
    </DelightMoment>
  )
}
