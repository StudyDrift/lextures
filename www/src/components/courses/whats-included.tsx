import { BookOpen, Clock, Layers } from 'lucide-react'
import { COURSES_COPY } from '../../lib/courses-copy'
import type { MarketplaceWhatsIncluded } from '../../lib/marketplace-api'

export function WhatsIncluded({ data }: { data: MarketplaceWhatsIncluded }) {
  const items = [
    {
      icon: Layers,
      label: COURSES_COPY.modules(data.moduleCount),
    },
    {
      icon: BookOpen,
      label: COURSES_COPY.items(data.itemCount),
    },
  ]
  if (data.estimatedDurationMinutes != null && data.estimatedDurationMinutes > 0) {
    items.push({
      icon: Clock,
      label: COURSES_COPY.duration(data.estimatedDurationMinutes),
    })
  }

  return (
    <section aria-labelledby="whats-included-heading">
      <h2
        id="whats-included-heading"
        className="font-display text-[22px] font-semibold"
        style={{ color: 'var(--ink-nav)' }}
      >
        {COURSES_COPY.whatsIncluded}
      </h2>
      <ul className="mt-4 grid gap-3 sm:grid-cols-3">
        {items.map(({ icon: Icon, label }) => (
          <li
            key={label}
            className="flex items-center gap-3 border px-4 py-3"
            style={{
              borderColor: 'var(--line-card)',
              borderRadius: 'var(--radius-card)',
              backgroundColor: 'var(--panel)',
            }}
          >
            <Icon className="h-5 w-5 shrink-0" style={{ color: '#4fa894' }} aria-hidden />
            <span className="text-[14px] font-medium" style={{ color: 'var(--ink-nav)' }}>
              {label}
            </span>
          </li>
        ))}
      </ul>
    </section>
  )
}
