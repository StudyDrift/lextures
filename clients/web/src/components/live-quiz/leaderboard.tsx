import { useTranslation } from 'react-i18next'

export type LeaderboardRow = {
  rank: number
  playerId: string
  nickname: string
  totalScore: number
}

export function LiveQuizLeaderboard({
  rows,
  privacy = 'names',
  variant = 'list',
  highlightPlayerId,
  youRank,
}: {
  rows: LeaderboardRow[]
  privacy?: 'names' | 'nicknames' | 'hidden'
  variant?: 'list' | 'podium'
  highlightPlayerId?: string
  youRank?: number
}) {
  const { t } = useTranslation('common')
  const reduceMotion =
    typeof window !== 'undefined' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches

  if (privacy === 'hidden') {
    return (
      <p className="text-sm text-muted-foreground" role="status">
        {t('liveQuiz.leaderboard.hidden')}
      </p>
    )
  }

  const labelFor = (row: LeaderboardRow, index: number) => {
    if (!row.nickname) {
      return t('liveQuiz.leaderboard.anonymous', { n: index + 1 })
    }
    return row.nickname
  }

  if (variant === 'podium') {
    const top3 = rows.slice(0, 3)
    return (
      <div
        className="space-y-4"
        role="region"
        aria-label={t('liveQuiz.leaderboard.podiumAria')}
      >
        <ol
          className={
            reduceMotion
              ? 'flex flex-wrap items-end justify-center gap-4'
              : 'flex flex-wrap items-end justify-center gap-4 motion-safe:animate-in motion-safe:fade-in'
          }
        >
          {top3.map((row, i) => {
            const heights = ['h-28', 'h-36', 'h-24']
            const order = [1, 0, 2]
            const place = order.indexOf(i) >= 0 ? i : i
            return (
              <li
                key={row.playerId || `p-${i}`}
                className={`flex w-28 flex-col items-center justify-end rounded-t-lg bg-muted/60 px-2 py-3 ${heights[place] ?? 'h-24'}`}
              >
                <span className="text-xs uppercase tracking-wide text-muted-foreground">
                  {t('liveQuiz.leaderboard.place', { rank: row.rank })}
                </span>
                <span className="mt-1 text-center text-sm font-semibold">{labelFor(row, i)}</span>
                <span className="mt-1 tabular-nums text-lg font-bold">{row.totalScore}</span>
              </li>
            )
          })}
        </ol>
        {rows.length > 3 ? (
          <LiveQuizLeaderboard
            rows={rows.slice(3)}
            privacy={privacy}
            variant="list"
            highlightPlayerId={highlightPlayerId}
          />
        ) : null}
      </div>
    )
  }

  return (
    <div className="space-y-2">
      {typeof youRank === 'number' ? (
        <p className="text-sm" aria-live="polite">
          {t('liveQuiz.leaderboard.yourRank', { rank: youRank })}
        </p>
      ) : null}
      <ol className="space-y-1" aria-label={t('liveQuiz.leaderboard.listAria')}>
        {rows.map((row, i) => {
          const mine = highlightPlayerId && row.playerId === highlightPlayerId
          return (
            <li
              key={row.playerId || `row-${i}`}
              className={
                mine
                  ? 'flex items-baseline justify-between rounded-md bg-primary/10 px-2 py-1.5 font-medium'
                  : 'flex items-baseline justify-between px-2 py-1'
              }
            >
              <span>
                <span className="me-2 tabular-nums text-muted-foreground">#{row.rank}</span>
                {labelFor(row, i)}
              </span>
              <span className="tabular-nums">{row.totalScore}</span>
            </li>
          )
        })}
      </ol>
    </div>
  )
}
