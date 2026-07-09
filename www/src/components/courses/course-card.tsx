import { formatMarketplacePrice, type PublicMarketplaceCourse } from '../../lib/marketplace-api'
import { COURSES_COPY } from '../../lib/courses-copy'

type PriceBadgeProps = {
  priceCents: number
  priceCurrency: string
  listPriceCents?: number | null
  className?: string
}

export function PriceBadge({ priceCents, priceCurrency, listPriceCents, className = '' }: PriceBadgeProps) {
  const free = priceCents <= 0
  const price = formatMarketplacePrice(priceCents, priceCurrency, COURSES_COPY.free)
  const list =
    listPriceCents != null && listPriceCents > priceCents
      ? formatMarketplacePrice(listPriceCents, priceCurrency, COURSES_COPY.free)
      : null

  return (
    <span className={`inline-flex items-baseline gap-2 ${className}`}>
      <span
        className="text-[15px] font-semibold"
        style={{ color: free ? '#4fa894' : 'var(--ink-nav)' }}
      >
        {price}
      </span>
      {list && (
        <span className="text-[13px] line-through" style={{ color: 'var(--text-soft)' }}>
          {list}
        </span>
      )}
    </span>
  )
}

type RatingStarsProps = {
  average: number | null
  count: number
}

export function RatingStars({ average, count }: RatingStarsProps) {
  if (average == null) return null
  const label = COURSES_COPY.ratingLabel(average, count)
  return (
    <span className="inline-flex items-center gap-1.5 text-[13px]" style={{ color: 'var(--text-soft)' }}>
      <span aria-hidden className="font-semibold" style={{ color: '#f49b44' }}>
        ★ {average.toFixed(1)}
      </span>
      {count > 0 && <span aria-hidden>({count.toLocaleString()})</span>}
      <span className="sr-only">{label}</span>
    </span>
  )
}

export function CourseHeroPlaceholder({ title }: { title: string }) {
  const initial = (title.trim()[0] || 'L').toUpperCase()
  return (
    <div
      className="flex h-full w-full items-center justify-center"
      style={{
        background: 'linear-gradient(135deg, rgba(106,197,176,0.35), rgba(242,104,78,0.28))',
      }}
      aria-hidden
    >
      <span className="font-display text-4xl font-semibold" style={{ color: 'var(--ink-nav)' }}>
        {initial}
      </span>
    </div>
  )
}

type CourseCardProps = {
  course: PublicMarketplaceCourse
}

export function CourseCard({ course }: CourseCardProps) {
  const href = `/courses/${encodeURIComponent(course.slug || course.courseCode)}`
  const price = formatMarketplacePrice(course.priceCents, course.priceCurrency, COURSES_COPY.free)
  const accessibleName = `${course.title}, ${price}`

  return (
    <li>
      <a
        href={href}
        className="group flex h-full flex-col overflow-hidden border no-underline transition-shadow"
        style={{
          backgroundColor: 'var(--panel)',
          borderColor: 'var(--line-card)',
          borderRadius: 'var(--radius-card)',
          boxShadow: 'var(--shadow-panel)',
        }}
        aria-label={accessibleName}
      >
        <div className="aspect-[16/9] w-full overflow-hidden" style={{ backgroundColor: 'var(--paper)' }}>
          {course.heroImageUrl ? (
            <img
              src={course.heroImageUrl}
              alt=""
              loading="lazy"
              className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.02]"
            />
          ) : (
            <CourseHeroPlaceholder title={course.title} />
          )}
        </div>
        <div className="flex flex-1 flex-col gap-2 p-4">
          <div className="flex flex-wrap gap-1.5">
            {course.category && (
              <span
                className="rounded-full px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide"
                style={{ backgroundColor: 'rgba(106,197,176,0.14)', color: '#4fa894' }}
              >
                {course.category}
              </span>
            )}
            {course.level && (
              <span
                className="rounded-full px-2 py-0.5 text-[11px] font-medium capitalize"
                style={{ backgroundColor: 'rgba(38,58,60,0.06)', color: 'var(--text-soft)' }}
              >
                {course.level}
              </span>
            )}
          </div>
          <h3
            className="font-display text-[17px] font-semibold leading-snug"
            style={{ color: 'var(--ink-nav)' }}
          >
            {course.title}
          </h3>
          {course.instructorName && (
            <p className="text-[13px]" style={{ color: 'var(--text-soft)' }}>
              {course.instructorName}
            </p>
          )}
          <div className="mt-auto flex flex-wrap items-center justify-between gap-2 pt-2">
            <div className="flex flex-col gap-1">
              <RatingStars average={course.averageRating} count={course.ratingCount} />
              <span className="text-[12px]" style={{ color: 'var(--text-soft)' }}>
                {COURSES_COPY.students(course.enrollmentCount)}
              </span>
            </div>
            <PriceBadge
              priceCents={course.priceCents}
              priceCurrency={course.priceCurrency}
              listPriceCents={course.listPriceCents}
            />
          </div>
        </div>
      </a>
    </li>
  )
}
