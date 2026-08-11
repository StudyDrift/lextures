import { PriceBadge } from './course-card'
import { COURSES_COPY } from '../../lib/courses-copy'
import {
  displayHandoffCoupon,
  enrollHandoffUrl,
  formatMarketplacePrice,
} from '../../lib/marketplace-api'
import type { PublicMarketplaceCourse } from '../../lib/marketplace-api'

type EnrollPanelProps = {
  course: PublicMarketplaceCourse
  sticky?: boolean
  /** Unvalidated coupon from the page URL (MKTC.5 FR-12/FR-13). */
  coupon?: string | null
}

export function EnrollPanel({ course, sticky, coupon }: EnrollPanelProps) {
  const priceLabel = formatMarketplacePrice(course.priceCents, course.priceCurrency, COURSES_COPY.free)
  const ctaLabel =
    course.priceCents <= 0 ? COURSES_COPY.enrollFree : COURSES_COPY.enrollPaid(priceLabel)
  const safeCoupon = displayHandoffCoupon(coupon)
  const href = enrollHandoffUrl(course.slug || course.courseCode, {
    coupon: safeCoupon || undefined,
  })

  return (
    <div
      className={
        sticky
          ? 'fixed inset-x-0 bottom-0 z-40 border-t px-4 py-3 lg:hidden'
          : 'border p-5'
      }
      style={{
        backgroundColor: 'var(--panel)',
        borderColor: 'var(--line-card)',
        borderRadius: sticky ? undefined : 'var(--radius-card)',
        boxShadow: sticky ? '0 -4px 24px rgba(38,58,60,0.08)' : 'var(--shadow-panel)',
      }}
      data-testid="www-enroll-panel"
    >
      <div className={sticky ? 'mx-auto flex max-w-[960px] items-center justify-between gap-4' : 'flex flex-col gap-4'}>
        <div className={sticky ? 'min-w-0' : undefined}>
          <PriceBadge
            priceCents={course.priceCents}
            priceCurrency={course.priceCurrency}
            listPriceCents={course.listPriceCents}
            className="text-lg"
          />
          {safeCoupon ? (
            <p
              className={sticky ? 'mt-0.5 truncate text-[12px]' : 'mt-1 text-[13px]'}
              style={{ color: 'var(--text-soft)' }}
              data-testid="www-coupon-will-apply"
            >
              {COURSES_COPY.couponWillApply(safeCoupon)}
            </p>
          ) : null}
        </div>
        <div className={sticky ? 'flex shrink-0 items-center gap-3' : 'flex flex-col gap-2'}>
          <a href={href} className="btn-primary" aria-label={ctaLabel} data-testid="www-enroll-cta">
            {ctaLabel}
          </a>
          {!sticky && (
            <a
              href={href}
              className="text-center text-[13px] no-underline"
              style={{ color: 'var(--text-soft)' }}
            >
              {COURSES_COPY.viewOnLextures}
            </a>
          )}
        </div>
      </div>
    </div>
  )
}
