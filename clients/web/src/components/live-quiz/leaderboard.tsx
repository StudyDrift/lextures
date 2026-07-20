import { useTranslation } from 'react-i18next'
import { AnimatedList } from '../ui/animated-list'
import { useCountUp } from '../../lib/use-count-up'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { usePrefersReducedMotion } from '../../lib/motion'
import { motion } from '../../lib/motion'

export type LeaderboardRow = {
  rank: number
  playerId: string
  nickname: string
  totalScore: number
}

function ScoreCell({ score, animate }: { score: number; animate: boolean }) {
  const { formatted } = useCountUp(score, { enabled: animate })
  return <span className="tabular-nums lx-delight-count-up">{formatted}</span>
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
  const { ffMotionDelight, ffMotionLists } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const delightOn = ffMotionDelight !== false && !reduceMotion
  const listsOn = ffMotionLists !== false

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
        <ol className="flex flex-wrap items-end justify-center gap-4">
          {top3.map((row, i) => {
            const heights = ['h-28', 'h-36', 'h-24']
            const place = i
            const delay = delightOn ? motion.staggerDelay(i) : 0
            return (
              <li
                key={row.playerId || `p-${i}`}
                className={`flex w-28 flex-col items-center justify-end rounded-t-lg bg-muted/60 px-2 py-3 ${heights[place] ?? 'h-24'} ${delightOn ? 'lx-delight-badge-in' : ''}`}
                style={delightOn ? { animationDelay: `${delay}ms` } : undefined}
              >
                <span className="text-xs uppercase tracking-wide text-muted-foreground">
                  {t('liveQuiz.leaderboard.place', { rank: row.rank })}
                </span>
                <span className="mt-1 text-center text-sm font-semibold">{labelFor(row, i)}</span>
                <span className="mt-1 text-lg font-bold">
                  <ScoreCell score={row.totalScore} animate={delightOn} />
                </span>
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
      <AnimatedList
        as="ol"
        items={rows}
        getKey={(row) => row.playerId || `row-${row.rank}`}
        enabled={listsOn && delightOn}
        aria-label={t('liveQuiz.leaderboard.listAria')}
        className="space-y-1"
      >
        {(row, meta) => {
          const mine = highlightPlayerId && row.playerId === highlightPlayerId
          return (
            <div
              className={
                (mine
                  ? 'flex items-baseline justify-between rounded-md bg-primary/10 px-2 py-1.5 font-medium'
                  : 'flex items-baseline justify-between px-2 py-1') +
                (meta.className ? ` ${meta.className}` : '')
              }
            >
              <span>
                <span className="me-2 tabular-nums text-muted-foreground">#{row.rank}</span>
                {labelFor(row, meta.index)}
              </span>
              <ScoreCell score={row.totalScore} animate={delightOn} />
            </div>
          )
        }}
      </AnimatedList>
    </div>
  )
}
