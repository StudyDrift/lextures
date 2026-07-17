import { useEffect, useId, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Award, Loader2, ShieldCheck } from 'lucide-react'
import { fetchPublicBadges, type PublicBadge } from '../../lib/badges-api'

export default function PublicBadgeListPage() {
  const { handle = '' } = useParams<{ handle: string }>()
  const navigate = useNavigate()
  const titleId = useId()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [status, setStatus] = useState<string>('loading')
  const [displayName, setDisplayName] = useState('')
  const [badges, setBadges] = useState<PublicBadge[]>([])
  const [resolvedHandle, setResolvedHandle] = useState(handle)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    void fetchPublicBadges(handle)
      .then((data) => {
        if (cancelled) return
        if (data.redirectTo) {
          void navigate(`/badges/${data.redirectTo}`, { replace: true })
          return
        }
        setResolvedHandle(data.handle)
        setDisplayName(data.displayName)
        setBadges(data.badges ?? [])
        setStatus(data.status)
      })
      .catch((e) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load badge page.')
          setStatus('error')
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [handle, navigate])

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <main className="mx-auto max-w-3xl px-4 py-10">
        <header className="mb-8">
          <p className="text-xs font-medium uppercase tracking-wide text-indigo-600 dark:text-indigo-400">
            Verified by Lextures
          </p>
          <h1 id={titleId} className="mt-1 flex items-center gap-2 text-2xl font-semibold">
            <Award className="h-7 w-7 text-indigo-600" aria-hidden />
            {displayName || resolvedHandle || 'Badge backpack'}
          </h1>
          {resolvedHandle ? (
            <p className="mt-1 text-sm text-slate-500">@{resolvedHandle}</p>
          ) : null}
        </header>

        {loading ? (
          <p className="inline-flex items-center gap-2 text-sm text-slate-600">
            <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
            Loading badges…
          </p>
        ) : null}

        {error ? (
          <p role="alert" className="text-sm text-red-700 dark:text-red-300">
            {error}
          </p>
        ) : null}

        {!loading && status === 'private' ? (
          <p className="rounded-xl border border-slate-200 bg-white p-6 text-sm dark:border-slate-700 dark:bg-slate-900">
            This badge page is private.
          </p>
        ) : null}

        {!loading && status === 'ok' && badges.length === 0 ? (
          <p className="text-sm text-slate-600 dark:text-slate-300">No public badges yet.</p>
        ) : null}

        <ul className="grid gap-4 sm:grid-cols-2">
          {badges.map((b) => (
            <li key={b.id}>
              <Link
                to={`/badges/${encodeURIComponent(resolvedHandle)}/${encodeURIComponent(b.slug)}`}
                className="block rounded-xl border border-slate-200 bg-white p-4 shadow-sm transition-[background-color,color,border-color] hover:border-indigo-300 dark:border-slate-700 dark:bg-slate-900 dark:hover:border-indigo-600"
              >
                <h2 className="font-semibold">{b.name}</h2>
                <p className="mt-1 text-xs text-slate-500">
                  Issued {new Date(b.issuedAt).toLocaleDateString()}
                  {b.courseTitle ? ` · ${b.courseTitle}` : ''}
                </p>
                {b.description ? (
                  <p className="mt-2 line-clamp-3 text-sm text-slate-600 dark:text-slate-300">{b.description}</p>
                ) : null}
                <p className="mt-3 inline-flex items-center gap-1 text-xs font-medium text-emerald-700 dark:text-emerald-400">
                  <ShieldCheck className="h-3.5 w-3.5" aria-hidden />
                  Verifiable
                </p>
              </Link>
            </li>
          ))}
        </ul>
      </main>
    </div>
  )
}
