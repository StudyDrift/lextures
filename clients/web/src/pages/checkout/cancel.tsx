import { useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Link, useSearchParams } from 'react-router-dom'
import {
  displayCouponCode,
  normalizeCouponInput,
  rememberPendingCoupon,
} from '../../lib/marketplace-coupon'

export default function CheckoutCancelPage() {
  const { t } = useTranslation('billing')
  const [params] = useSearchParams()
  const slug = params.get('slug') ?? ''
  const couponRaw = params.get('coupon') ?? ''
  const coupon = couponRaw ? normalizeCouponInput(couponRaw) : ''

  // FR-16: keep the applied code so the learner can retry (sessionStorage + link).
  useEffect(() => {
    if (slug && coupon) {
      rememberPendingCoupon(slug, coupon)
    }
  }, [slug, coupon])

  const backTo = (() => {
    if (!slug) return '/courses'
    const base = `/marketplace/${encodeURIComponent(slug)}`
    if (coupon) return `${base}?coupon=${encodeURIComponent(coupon)}`
    return base
  })()
  const backLabel = slug
    ? t('billing.checkout.cancel.backToCourse')
    : t('billing.checkout.cancel.backToCourses')

  const displayCode = coupon ? displayCouponCode(coupon) : ''

  return (
    <main className="mx-auto flex min-h-screen max-w-lg flex-col items-center justify-center px-4 text-center">
      <h1 className="text-2xl font-semibold">{t('billing.checkout.cancel.title')}</h1>
      <p className="mt-2 text-fg-muted">{t('billing.checkout.cancel.description')}</p>
      {displayCode ? (
        <p className="mt-2 text-sm text-fg-muted" data-testid="checkout-cancel-coupon" role="status">
          {t('billing.checkout.cancel.couponKept', { code: displayCode })}
        </p>
      ) : null}
      <Link
        to={backTo}
        className="mt-6 inline-flex rounded-lg border border-border-strong px-4 py-2 text-sm font-medium hover:bg-surface-base dark:border-border-default dark:hover:bg-surface-raised"
      >
        {backLabel}
      </Link>
    </main>
  )
}
