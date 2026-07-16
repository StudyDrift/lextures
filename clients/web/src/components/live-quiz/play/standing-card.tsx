import { useTranslation } from 'react-i18next'

export function StandingCard({
  playerId,
  nickname,
  leaderboard,
  totalScore,
}: {
  playerId?: string
  nickname?: string
  leaderboard?: Array<{ rank: number; playerId: string; nickname: string; totalScore: number }>
  totalScore?: number
}) {
  const { t } = useTranslation('common')
  const me = leaderboard?.find((r) => r.playerId === playerId)
  const rank = me?.rank
  const top3 = (leaderboard ?? []).slice(0, 3)

  return (
    <div className="space-y-6 text-center">
      <h2 className="text-2xl font-semibold">{t('liveQuiz.standing.title')}</h2>
      {rank != null ? (
        <p className="text-4xl font-bold tabular-nums" aria-live="polite">
          {t('liveQuiz.standing.youPlaced', { rank })}
        </p>
      ) : (
        <p className="text-lg text-slate-600 dark:text-neutral-300">
          {nickname ? t('liveQuiz.standing.thanksNamed', { nickname }) : t('liveQuiz.standing.thanks')}
        </p>
      )}
      {typeof totalScore === 'number' && (
        <p className="text-lg tabular-nums">{t('liveQuiz.standing.score', { score: totalScore })}</p>
      )}
      {top3.length > 0 && (
        <ol className="mx-auto max-w-sm space-y-2 text-start">
          {top3.map((row) => (
            <li
              key={row.playerId}
              className={
                row.playerId === playerId
                  ? 'flex items-center justify-between rounded-lg bg-indigo-50 px-3 py-2 font-medium dark:bg-indigo-950/40'
                  : 'flex items-center justify-between rounded-lg bg-slate-100 px-3 py-2 dark:bg-neutral-800'
              }
            >
              <span>
                <span className="tabular-nums">{row.rank}.</span> {row.nickname}
              </span>
              <span className="tabular-nums">{row.totalScore}</span>
            </li>
          ))}
        </ol>
      )}
    </div>
  )
}
