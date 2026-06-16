import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Route } from 'lucide-react'
import { fetchMyPaths, type PathProgress } from '../../lib/learning-paths-api'

export function DashboardLearningPathsCard() {
  const [paths, setPaths] = useState<PathProgress[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let active = true
    fetchMyPaths()
      .then((rows) => active && setPaths(rows.slice(0, 3)))
      .catch(() => active && setPaths([]))
      .finally(() => active && setLoading(false))
    return () => {
      active = false
    }
  }, [])

  if (loading) {
    return (
      <section className="rounded-lg border bg-card p-4" aria-label="My Learning Paths">
        <div className="h-20 motion-safe:animate-pulse rounded bg-muted" aria-hidden />
      </section>
    )
  }

  if (paths.length === 0) return null

  return (
    <section className="rounded-lg border bg-card p-4" aria-labelledby="dashboard-paths-heading">
      <div className="flex items-center justify-between gap-2">
        <h2 id="dashboard-paths-heading" className="flex items-center gap-2 text-sm font-semibold">
          <Route className="size-4" aria-hidden />
          My Learning Paths
        </h2>
        <Link to="/my-paths" className="text-xs font-medium text-primary hover:underline">
          View all
        </Link>
      </div>
      <ul className="mt-3 space-y-3">
        {paths.map((path) => (
          <li key={path.pathId}>
            <div className="flex items-center justify-between gap-2 text-sm">
              <span className="font-medium">{path.pathTitle}</span>
              <span className="text-muted-foreground">{path.progressLabel}</span>
            </div>
            <div
              className="mt-1 h-1.5 overflow-hidden rounded-full bg-muted"
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={path.percent}
              aria-label={path.progressLabel}
            >
              <div className="h-full rounded-full bg-primary" style={{ width: `${path.percent}%` }} />
            </div>
          </li>
        ))}
      </ul>
    </section>
  )
}
