import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { checkEntitlement, fetchMyEntitlements } from '../../lib/billing-api'
import { authorizedFetch } from '../../lib/api'

export default function CheckoutSuccessPage() {
  const [params] = useSearchParams()
  const courseId = params.get('course_id') ?? ''
  const [status, setStatus] = useState<'verifying' | 'ready' | 'timeout'>('verifying')

  useEffect(() => {
    let cancelled = false
    let attempts = 0

    async function poll() {
      attempts += 1
      try {
        const meRes = await authorizedFetch('/api/v1/me')
        if (!meRes.ok) {
          throw new Error('not signed in')
        }
        const me = (await meRes.json()) as { id: string }
        if (courseId) {
          const entitled = await checkEntitlement(me.id, courseId)
          if (entitled) {
            if (!cancelled) setStatus('ready')
            return
          }
        } else {
          const items = await fetchMyEntitlements()
          if (items.length > 0) {
            if (!cancelled) setStatus('ready')
            return
          }
        }
      } catch {
        // keep polling briefly
      }
      if (attempts >= 10) {
        if (!cancelled) setStatus('timeout')
        return
      }
      window.setTimeout(() => void poll(), 1000)
    }

    void poll()
    return () => {
      cancelled = true
    }
  }, [courseId])

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col items-center justify-center px-4 text-center">
      {status === 'verifying' ? (
        <>
          <Loader2 className="h-10 w-10 animate-spin text-indigo-600" aria-hidden />
          <h1 className="mt-4 text-2xl font-semibold">Verifying payment…</h1>
          <p className="mt-2 text-slate-600 dark:text-neutral-400">
            This usually takes a few seconds. Please keep this page open.
          </p>
        </>
      ) : null}
      {status === 'ready' ? (
        <>
          <h1 className="text-2xl font-semibold text-emerald-700 dark:text-emerald-300">Payment confirmed</h1>
          <p className="mt-2 text-slate-600 dark:text-neutral-400">Your access is ready. Start learning!</p>
          <Link
            to={courseId ? `/courses/${courseId}` : '/'}
            className="mt-6 inline-flex rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            Continue
          </Link>
        </>
      ) : null}
      {status === 'timeout' ? (
        <>
          <h1 className="text-2xl font-semibold">Still processing</h1>
          <p className="mt-2 text-slate-600 dark:text-neutral-400">
            Your payment may still be processing. Check billing settings in a moment.
          </p>
          <Link to="/me/billing" className="mt-6 text-sm font-medium text-indigo-600 hover:underline">
            Open billing settings
          </Link>
        </>
      ) : null}
    </main>
  )
}
