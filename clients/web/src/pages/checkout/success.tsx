import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { checkEntitlement, fetchMyEntitlements } from '../../lib/billing-api'
import { authorizedFetch } from '../../lib/api'
import { marketplaceCoursePath } from '../../lib/marketplace-api'
import { clearPendingCoupon, displayCouponCode } from '../../lib/marketplace-coupon'
import { formatMarketplacePrice } from '../../lib/marketplace-price'

const POLL_ATTEMPTS = 20
const POLL_INTERVAL_MS = 1000

export default function CheckoutSuccessPage() {
  const { t, i18n } = useTranslation('billing')
  const [params] = useSearchParams()
  const courseId = params.get('course_id') ?? ''
  const courseCode = params.get('course_code') ?? ''
  const slug = params.get('slug') ?? ''
  const couponRaw = params.get('coupon') ?? params.get('coupon_code') ?? ''
  const coupon = couponRaw ? displayCouponCode(couponRaw) : ''
  const discountCentsRaw = params.get('discount_cents') ?? params.get('saved_cents') ?? ''
  const discountCents = discountCentsRaw ? Number.parseInt(discountCentsRaw, 10) : NaN
  const currency = (params.get('currency') ?? 'usd').toLowerCase()
  const [status, setStatus] = useState<'verifying' | 'ready' | 'timeout'>('verifying')

  useEffect(() => {
    if (slug) clearPendingCoupon(slug)
  }, [slug])

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

  const savedLabel =
    Number.isFinite(discountCents) && discountCents > 0
      ? formatMarketplacePrice(discountCents, currency, i18n.language, '')
      : null

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col items-center justify-center px-4 text-center">
      {status === 'verifying' ? (
        <>
          <Loader2 className="h-10 w-10 motion-safe:animate-spin text-accent-fg" aria-hidden />
          <h1 className="mt-4 text-2xl font-semibold">
            {t('billing.checkout.success.verifying.title')}
          </h1>
          <p className="mt-2 text-fg-muted" aria-live="polite">
            {t('billing.checkout.success.verifying.description')}
          </p>
        </>
      ) : null}
      {status === 'ready' ? (
        <>
          <h1 className="text-2xl font-semibold text-success-fg">
            {t('billing.checkout.success.ready.title')}
          </h1>
          <p className="mt-2 text-fg-muted">{t('billing.checkout.success.ready.description')}</p>
          {coupon || savedLabel ? (
            <p
              className="mt-3 text-sm text-fg-muted"
              data-testid="checkout-success-coupon"
              role="status"
            >
              {coupon && savedLabel
                ? t('billing.checkout.success.ready.couponSaved', {
                    code: coupon,
                    amount: savedLabel,
                  })
                : coupon
                  ? t('billing.checkout.success.ready.couponApplied', { code: coupon })
                  : t('billing.checkout.success.ready.saved', { amount: savedLabel })}
            </p>
          ) : null}
          <Link
            to={continueTo}
            className="mt-6 inline-flex rounded-lg bg-accent-solid px-4 py-2 text-sm font-medium text-fg-on-accent hover:opacity-90"
          >
            {t('billing.checkout.success.ready.continue')}
          </Link>
        </>
      ) : null}
      {status === 'timeout' ? (
        <>
          <h1 className="text-2xl font-semibold">{t('billing.checkout.success.timeout.title')}</h1>
          <p className="mt-2 text-fg-muted" role="status">
            {t('billing.checkout.success.timeout.description')}
          </p>
          <Link to={fallbackTo} className="mt-6 text-sm font-medium text-accent-fg hover:underline">
            {t('billing.checkout.success.timeout.billingLink')}
          </Link>
        </>
      ) : null}
    </main>
  )
}
