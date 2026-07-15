import { useEffect, useId, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Award, CheckCircle2, Loader2, ShieldAlert, XCircle } from 'lucide-react'
import { fetchPublicBadge, verifyBadge, type PublicBadge } from '../../lib/badges-api'

type VerifyState = 'idle' | 'loading' | 'verified' | 'revoked' | 'unverified' | 'error'

export default function PublicBadgeDetailPage() {
  const { handle = '', badgeSlug = '' } = useParams<{ handle: string; badgeSlug: string }>()
  const navigate = useNavigate()
  const titleId = useId()
  const liveId = useId()
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [badge, setBadge] = useState<PublicBadge | null>(null)
  const [verifyState, setVerifyState] = useState<VerifyState>('idle')
  const [verifyMessage, setVerifyMessage] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    void fetchPublicBadge(handle, badgeSlug)
      .then((data) => {
        if (cancelled) return
        if (data.redirectTo) {
          void navigate(`/badges/${data.redirectTo}/${badgeSlug}`, { replace: true })
          return
        }
        setBadge(data)
      })
      .catch((e) => {
        if (!cancelled) setError(e instanceof Error ? e.message : 'Badge not found.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [handle, badgeSlug, navigate])

  async function onVerify() {
    if (!badge?.shareSlug) return
    setVerifyState('loading')
    setVerifyMessage('Checking signature…')
    try {
      const result = await verifyBadge(badge.shareSlug)
      if (result.revoked) {
        setVerifyState('revoked')
        setVerifyMessage('This badge has been revoked.')
      } else if (result.verified) {
        setVerifyState('verified')
        setVerifyMessage('Verified — signature matches issuer key.')
      } else {
        setVerifyState('unverified')
        setVerifyMessage('Could not verify signature.')
      }
    } catch (e) {
      setVerifyState('error')
      setVerifyMessage(e instanceof Error ? e.message : 'Verification failed.')
    }
  }

  function shareX() {
    if (!badge) return
    const url = window.location.href
    const text = `I earned the “${badge.name}” micro-credential — verified by Lextures`
    window.open(
      `https://twitter.com/intent/tweet?text=${encodeURIComponent(text)}&url=${encodeURIComponent(url)}`,
      '_blank',
      'noopener,noreferrer',
    )
  }

  return (
    <div className="min-h-screen bg-slate-50 text-slate-900 dark:bg-slate-950 dark:text-slate-100">
      <main className="mx-auto max-w-2xl px-4 py-10">
        <p className="mb-4 text-sm">
          <Link to={`/badges/${encodeURIComponent(handle)}`} className="text-indigo-600 hover:underline dark:text-indigo-400">
            ← All badges
          </Link>
        </p>

        {loading ? (
          <p className="inline-flex items-center gap-2 text-sm text-slate-600">
            <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
            Loading badge…
          </p>
        ) : null}

        {error ? (
          <p role="alert" className="text-sm text-red-700 dark:text-red-300">
            {error}
          </p>
        ) : null}

        {badge ? (
          <article className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900">
            <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-indigo-100 text-indigo-700 dark:bg-indigo-950 dark:text-indigo-300">
              <Award className="h-9 w-9" aria-hidden />
            </div>
            <h1 id={titleId} className="text-2xl font-semibold">
              {badge.name}
            </h1>
            <p className="mt-1 text-sm text-slate-500">
              Awarded to {badge.recipientDisplayName ?? handle}
              {badge.issuerName ? ` · Issued by ${badge.issuerName}` : ''}
            </p>
            <p className="mt-1 text-xs text-slate-500">
              Issued {new Date(badge.issuedAt).toLocaleDateString()}
              {badge.courseTitle ? ` · ${badge.courseTitle}` : ''}
            </p>
            {badge.description ? (
              <p className="mt-4 text-sm text-slate-700 dark:text-slate-300">{badge.description}</p>
            ) : null}
            {badge.criteriaNarrative ? (
              <section className="mt-4">
                <h2 className="text-sm font-semibold">Criteria</h2>
                <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">{badge.criteriaNarrative}</p>
              </section>
            ) : null}

            <div className="mt-6 flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => void onVerify()}
                className="rounded-lg bg-indigo-600 px-3 py-2 text-sm font-medium text-white hover:bg-indigo-500"
                disabled={verifyState === 'loading'}
              >
                Verify
              </button>
              {badge.verifyUrl ? (
                <a
                  href={badge.verifyUrl}
                  className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium hover:bg-slate-50 dark:border-slate-600 dark:hover:bg-slate-800"
                  target="_blank"
                  rel="noreferrer"
                >
                  Download signed JSON
                </a>
              ) : null}
              <button
                type="button"
                onClick={shareX}
                className="rounded-lg border border-slate-300 px-3 py-2 text-sm font-medium hover:bg-slate-50 dark:border-slate-600 dark:hover:bg-slate-800"
              >
                Share to X
              </button>
            </div>

            <p
              id={liveId}
              role="status"
              aria-live="polite"
              className="mt-4 flex items-center gap-2 text-sm"
            >
              {verifyState === 'verified' ? (
                <>
                  <CheckCircle2 className="h-4 w-4 text-emerald-600" aria-hidden />
                  <span className="text-emerald-700 dark:text-emerald-400">{verifyMessage}</span>
                </>
              ) : null}
              {verifyState === 'revoked' ? (
                <>
                  <ShieldAlert className="h-4 w-4 text-amber-600" aria-hidden />
                  <span className="text-amber-700 dark:text-amber-400">{verifyMessage}</span>
                </>
              ) : null}
              {verifyState === 'unverified' || verifyState === 'error' ? (
                <>
                  <XCircle className="h-4 w-4 text-red-600" aria-hidden />
                  <span className="text-red-700 dark:text-red-400">{verifyMessage}</span>
                </>
              ) : null}
              {verifyState === 'loading' ? (
                <>
                  <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
                  <span>{verifyMessage}</span>
                </>
              ) : null}
            </p>
          </article>
        ) : null}
      </main>
    </div>
  )
}
