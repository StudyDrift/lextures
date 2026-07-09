import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { checkEntitlement, fetchMyEntitlements } from '../../lib/billing-api'
import { authorizedFetch } from '../../lib/api'
import { marketplaceCoursePath } from '../../lib/marketplace-api'

const POLL_ATTEMPTS = 20
const POLL_INTERVAL_MS = 1000

export default function CheckoutSuccessPage() {
  const { t } = useTranslation('billing')
  const [params] = useSearchParams()
  const courseId = params.get('course_id') ?? ''
  const courseCode = params.get('course_code') ?? ''
  const slug = params.get('slug') ?? ''
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
      if (attempts >= POLL_ATTEMPTS) {
        if (!cancelled) setStatus('timeout')
        return
      }
      window.setTimeout(() => void poll(), POLL_INTERVAL_MS)
    }

    void poll()
    return () => {
      cancelled = true
    }
  }, [courseId])

  const continueTo = courseCode
    ? marketplaceCoursePath(courseCode)
    : courseId
      ? `/courses/${encodeURIComponent(courseId)}`
      : '/'
  const fallbackTo = slug
    ? `/marketplace/${encodeURIComponent(slug)}`
    : courseCode
      ? marketplaceCoursePath(courseCode)
      : '/me/billing'

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col items-center justify-center px-4 text-center">
      {status === 'verifying' ? (
        <>
          <Loader2 className="h-10 w-10 motion-safe:animate-spin text-indigo-600" aria-hidden />
          <h1 className="mt-4 text-2xl font-semibold">{t('billing.checkout.success.verifying.title')}</h1>
          <p className="mt-2 text-slate-600 dark:text-neutral-400" aria-live="polite">
            {t('billing.checkout.success.verifying.description')}
          </p>
        </>
      ) : null}
      {status === 'ready' ? (
        <>
          <h1 className="text-2xl font-semibold text-emerald-700 dark:text-emerald-300">
            {t('billing.checkout.success.ready.title')}
          </h1>
          <p className="mt-2 text-slate-600 dark:text-neutral-400">
            {t('billing.checkout.success.ready.description')}
          </p>
          <Link
            to={continueTo}
            className="mt-6 inline-flex rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700"
          >
            {t('billing.checkout.success.ready.continue')}
          </Link>
        </>
      ) : null}
      {status === 'timeout' ? (
        <>
          <h1 className="text-2xl font-semibold">{t('billing.checkout.success.timeout.title')}</h1>
          <p className="mt-2 text-slate-600 dark:text-neutral-400" role="status">
            {t('billing.checkout.success.timeout.description')}
          </p>
          <Link to={fallbackTo} className="mt-6 text-sm font-medium text-indigo-600 hover:underline">
            {t('billing.checkout.success.timeout.billingLink')}
          </Link>
        </>
      ) : null}
    </main>
  )
}
