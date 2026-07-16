import { useTranslation } from 'react-i18next'
import type { ConnStatus } from '../../../lib/live-quiz-realtime'

export function ConnectionBadge({ conn }: { conn: ConnStatus }) {
  const { t } = useTranslation('common')
  if (conn === 'connected') return null
  const label =
    conn === 'reconnecting'
      ? t('liveQuiz.state.reconnecting')
      : conn === 'kicked'
        ? t('liveQuiz.play.kicked')
        : conn === 'ended'
          ? t('liveQuiz.play.gameEnded')
          : conn === 'disconnected'
            ? t('liveQuiz.play.disconnected')
            : t('liveQuiz.play.connecting')
  const tone =
    conn === 'reconnecting' || conn === 'connecting'
      ? 'bg-amber-500/20 text-amber-800 dark:text-amber-200'
      : conn === 'kicked'
        ? 'bg-red-500/20 text-red-800 dark:text-red-200'
        : 'bg-slate-500/20 text-slate-700 dark:text-slate-200'
  return (
    <p className={`rounded-md px-3 py-1.5 text-sm ${tone}`} role="status" aria-live="polite">
      {label}
    </p>
  )
}
