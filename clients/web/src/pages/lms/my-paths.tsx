import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { CheckCircle2, Route } from 'lucide-react'
import { fetchMyPaths, type PathProgress } from '../../lib/learning-paths-api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { LmsPage } from './lms-page'
import { EmptyState } from '../../components/ui/empty-state'

function PathProgressCard({ path }: { path: PathProgress }) {
  const label = path.progressLabel || `${path.percent}% — ${path.completedCourses} of ${path.totalCourses} complete`
  return (
    <article className="rounded-lg border bg-card p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h2 className="font-semibold">{path.pathTitle}</h2>
          {path.slug ? (
            <Link to={`/paths/${path.slug}`} className="text-sm text-primary hover:underline">
              View path details
            </Link>
          ) : null}
        </div>
        {path.completedAt ? (
          <CheckCircle2 className="size-5 shrink-0 text-emerald-600" aria-label="Path completed" />
        ) : null}
      </div>
      <div className="mt-4">
        <div
          className="h-2 overflow-hidden rounded-full bg-muted"
          role="progressbar"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={path.percent}
          aria-label={label}
        >
          <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${path.percent}%` }} />
        </div>
        <p className="mt-2 text-sm text-muted-foreground">{label}</p>
      </div>
      {path.courses.length > 0 ? (
        <ol className="mt-4 space-y-2 text-sm" aria-label="Courses in path">
          {path.courses.map((course) => (
            <li key={course.courseId} className="flex items-center gap-2">
              {course.completed ? (
                <CheckCircle2 className="size-4 text-emerald-600" aria-label="Completed" />
              ) : (
                <span className="size-4 rounded-full border" aria-hidden />
              )}
              <Link to={`/courses/${course.courseCode}`} className="hover:underline">
                {course.title}
              </Link>
            </li>
          ))}
        </ol>
      ) : null}
      {path.justCompleted ? (
        <p className="mt-3 rounded-md bg-amber-50 px-3 py-2 text-sm font-medium text-amber-900 dark:bg-amber-950/40 dark:text-amber-100">
          Congratulations — you completed this learning path!
        </p>
      ) : null}
    </article>
  )
}

export default function MyPathsPage() {
  const { ffLearningPaths, loading: featuresLoading } = usePlatformFeatures()
  const [paths, setPaths] = useState<PathProgress[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (featuresLoading || !ffLearningPaths) {
      setLoading(false)
      return
    }
    let active = true
    fetchMyPaths()
      .then((rows) => active && setPaths(rows))
      .catch(() => active && setError('Could not load your learning paths.'))
      .finally(() => active && setLoading(false))
    return () => {
      active = false
    }
  }, [ffLearningPaths, featuresLoading])

  if (!ffLearningPaths && !featuresLoading) {
    return (
      <LmsPage title="My Learning Paths">
        <EmptyState icon={Route} title="Learning paths are not enabled" body="Contact your administrator." />
      </LmsPage>
    )
  }

  return (
    <LmsPage title="My Learning Paths">
      {loading ? (
        <div className="h-32 motion-safe:animate-pulse rounded-lg border bg-card" aria-hidden />
      ) : error ? (
        <p className="text-sm text-destructive" role="alert">
          {error}
        </p>
      ) : paths.length === 0 ? (
        <EmptyState
          icon={Route}
          title="No learning paths yet"
          body="Browse public paths and enroll to start a curated course sequence."
          primaryAction={{ label: 'Browse learning paths', to: '/paths' }}
        />
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          {paths.map((path) => (
            <PathProgressCard key={path.pathId} path={path} />
          ))}
        </div>
      )}
    </LmsPage>
  )
}
