import { useEffect, useState } from 'react'
import { Trophy } from 'lucide-react'
import { fetchCourseLeaderboard, type CourseLeaderboard } from '../../lib/gamification-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

type Props = {
  courseCode: string
}

export function LeaderboardWidget({ courseCode }: Props) {
  const { ffGamification, loading: featuresLoading } = usePlatformFeatures()
  const [board, setBoard] = useState<CourseLeaderboard | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (featuresLoading || !ffGamification || !courseCode) return
    let cancelled = false
    void fetchCourseLeaderboard(courseCode)
      .then((b) => {
        if (!cancelled) {
          setBoard(b)
          setError(null)
        }
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Could not load leaderboard.')
      })
    return () => {
      cancelled = true
    }
  }, [featuresLoading, ffGamification, courseCode])

  if (featuresLoading || !ffGamification || error) return null
  if (!board || (board.topEntries.length === 0 && !board.currentUser)) return null

  return (
    <aside
      aria-label="Course leaderboard"
      className="mt-6 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-neutral-700 dark:bg-neutral-900"
    >
      <div className="flex items-center gap-2">
        <Trophy className="h-4 w-4 text-amber-600 dark:text-amber-400" aria-hidden />
        <h2 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Leaderboard</h2>
      </div>
      <div className="mt-3 overflow-x-auto">
        <table className="w-full min-w-[16rem] text-left text-sm">
          <thead>
            <tr className="border-b border-slate-200 text-xs uppercase tracking-wide text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
              <th scope="col" className="py-2 pe-2">
                Rank
              </th>
              <th scope="col" className="py-2 pe-2">
                Learner
              </th>
              <th scope="col" className="py-2 text-end">
                XP
              </th>
            </tr>
          </thead>
          <tbody>
            {board.topEntries.map((row) => (
              <tr
                key={row.userId}
                className={
                  row.isCurrentUser
                    ? 'bg-amber-50 font-semibold dark:bg-amber-950/30'
                    : 'border-b border-slate-100 dark:border-neutral-800'
                }
              >
                <td className="py-2 pe-2 tabular-nums">{row.rank}</td>
                <td className="py-2 pe-2">{row.displayName}</td>
                <td className="py-2 text-end tabular-nums">{row.xpEarned.toLocaleString()}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {board.currentUser &&
      !board.topEntries.some((e) => e.isCurrentUser) ? (
        <p className="mt-3 border-t border-slate-200 pt-3 text-xs text-slate-600 dark:border-neutral-700 dark:text-neutral-400">
          Your rank: #{board.currentUser.rank} · {board.currentUser.xpEarned.toLocaleString()} XP
        </p>
      ) : null}
    </aside>
  )
}
