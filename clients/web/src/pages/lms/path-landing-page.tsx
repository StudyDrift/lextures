import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { BookOpen, Lock, Sparkles } from 'lucide-react'
import {
  enrollInPath,
  fetchCatalogPathDetail,
  formatCents,
  formatDurationMinutes,
  type LearningPathDetail,
} from '../../lib/learning-paths-api'
import { getAccessToken } from '../../lib/auth'

export default function PathLandingPage() {
  const { slug = '' } = useParams<{ slug: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<LearningPathDetail | null>(null)
  const [status, setStatus] = useState<'loading' | 'ready' | 'notfound' | 'error'>('loading')
  const [enrolling, setEnrolling] = useState(false)
  const [enrollError, setEnrollError] = useState('')
  const [celebrate, setCelebrate] = useState(false)

  useEffect(() => {
    let active = true
    setStatus('loading')
    fetchCatalogPathDetail(slug)
      .then((d) => {
        if (!active) return
        if (!d) {
          setStatus('notfound')
          return
        }
        setDetail(d)
        setStatus('ready')
        document.title = `${d.path.title} — Learning Path`
      })
      .catch(() => active && setStatus('error'))
    return () => {
      active = false
    }
  }, [slug])

  async function handleEnroll() {
    if (!detail) return
    if (!getAccessToken()) {
      navigate('/login', { state: { from: `/paths/${slug}` } })
      return
    }
    setEnrolling(true)
    setEnrollError('')
    try {
      const result = await enrollInPath(detail.path.id)
      if (result.progress.justCompleted) {
        setCelebrate(true)
      } else {
        navigate('/my-paths')
      }
    } catch (e) {
      setEnrollError(e instanceof Error ? e.message : 'Could not enroll')
    } finally {
      setEnrolling(false)
    }
  }

  if (status === 'loading') {
    return (
      <main className="mx-auto max-w-3xl px-4 py-12">
        <div className="h-40 motion-safe:animate-pulse rounded-lg border bg-card" aria-hidden />
      </main>
    )
  }

  if (status === 'notfound') {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16 text-center">
        <h1 className="text-2xl font-semibold">Learning path not found</h1>
        <p className="mt-2 text-muted-foreground">This path is not available or is not public.</p>
      </main>
    )
  }

  if (status === 'error' || !detail) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16 text-center">
        <h1 className="text-2xl font-semibold">Something went wrong</h1>
        <p className="mt-2 text-muted-foreground">Please try again later.</p>
      </main>
    )
  }

  if (celebrate) {
    return (
      <main className="mx-auto max-w-lg px-4 py-20 text-center">
        <Sparkles className="mx-auto size-12 text-amber-500" aria-hidden />
        <h1 className="mt-4 text-3xl font-bold">Path complete!</h1>
        <p className="mt-2 text-muted-foreground">
          You earned the <strong>{detail.path.title}</strong> certificate.
        </p>
        <Link to="/my-paths" className="mt-6 inline-flex rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground">
          View my paths
        </Link>
      </main>
    )
  }

  const bundle = detail.path.bundlePriceCents
  const savings =
    bundle != null && detail.individualTotalCents > bundle
      ? detail.individualTotalCents - bundle
      : 0

  return (
    <div className="min-h-screen bg-slate-50/50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      <header className="border-b bg-white dark:border-neutral-800 dark:bg-neutral-900/50">
        <div className="mx-auto max-w-3xl px-4 py-10">
          <p className="text-sm font-medium text-indigo-600 dark:text-indigo-400">Learning path</p>
          <h1 className="mt-1 text-3xl font-bold tracking-tight">{detail.path.title}</h1>
          {detail.path.description ? (
            <p className="mt-3 text-base text-muted-foreground">{detail.path.description}</p>
          ) : null}
          <dl className="mt-6 flex flex-wrap gap-4 text-sm text-muted-foreground">
            <div>
              <dt className="sr-only">Course count</dt>
              <dd>{detail.courses.length} courses</dd>
            </div>
            <div>
              <dt className="sr-only">Total duration</dt>
              <dd>{formatDurationMinutes(detail.totalDurationMinutes)}</dd>
            </div>
          </dl>
          {detail.skillTags.length > 0 ? (
            <ul className="mt-4 flex flex-wrap gap-2" aria-label="Skills covered">
              {detail.skillTags.map((tag) => (
                <li key={tag} className="rounded-full bg-slate-100 px-3 py-1 text-xs font-medium dark:bg-neutral-800">
                  {tag}
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-4 py-8">
        <section aria-labelledby="path-courses-heading">
          <h2 id="path-courses-heading" className="text-lg font-semibold">
            Courses in this path
          </h2>
          <ol className="mt-4 space-y-3">
            {detail.courses.map((course, index) => (
              <li
                key={course.courseId}
                className="flex gap-3 rounded-lg border bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900/40"
              >
                <span
                  className="flex size-8 shrink-0 items-center justify-center rounded-full bg-indigo-100 text-sm font-semibold text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300"
                  aria-hidden
                >
                  {index + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <h3 className="font-medium">{course.title}</h3>
                  <p className="text-sm text-muted-foreground">{course.courseCode}</p>
                  <p className="mt-1 text-sm text-muted-foreground">
                    {formatDurationMinutes(course.durationMinutes)}
                    {course.listPriceCents != null ? ` · ${formatCents(course.listPriceCents)}` : ''}
                  </p>
                  {course.skillTags.length > 0 ? (
                    <ul className="mt-2 flex flex-wrap gap-1">
                      {course.skillTags.slice(0, 4).map((tag) => (
                        <li key={tag} className="text-xs text-muted-foreground">
                          #{tag}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </div>
                {index > 0 ? (
                  <Lock className="size-4 shrink-0 text-muted-foreground" aria-label="Recommended after prior course" />
                ) : (
                  <BookOpen className="size-4 shrink-0 text-muted-foreground" aria-hidden />
                )}
              </li>
            ))}
          </ol>
        </section>

        <aside className="mt-8 rounded-xl border bg-white p-6 dark:border-neutral-800 dark:bg-neutral-900/40">
          <h2 className="text-lg font-semibold">Enroll in this path</h2>
          <div className="mt-3 space-y-1">
            {bundle != null ? (
              <p className="text-2xl font-bold">{formatCents(bundle)}</p>
            ) : (
              <p className="text-2xl font-bold">Free</p>
            )}
            {detail.individualTotalCents > 0 ? (
              <p className="text-sm text-muted-foreground">
                Individual courses: {formatCents(detail.individualTotalCents)}
              </p>
            ) : null}
            {savings > 0 ? (
              <p className="text-sm font-medium text-emerald-600 dark:text-emerald-400">
                Save {formatCents(savings)} with the bundle
              </p>
            ) : null}
          </div>
          {enrollError ? (
            <p className="mt-3 text-sm text-destructive" role="alert">
              {enrollError}
            </p>
          ) : null}
          <button
            type="button"
            className="mt-4 w-full rounded-md bg-primary px-4 py-2.5 text-sm font-medium text-primary-foreground disabled:opacity-60"
            onClick={() => void handleEnroll()}
            disabled={enrolling}
          >
            {enrolling ? 'Enrolling…' : bundle != null ? `Enroll for ${formatCents(bundle)}` : 'Enroll for free'}
          </button>
        </aside>
      </main>
    </div>
  )
}
