import { Button } from './button'
import { cx } from './utils'

export type PaginationProps = {
  page: number
  pageCount: number
  onPageChange: (page: number) => void
  /** Accessible labels — required (no hardcoded English in defaults for product i18n). */
  labels: {
    nav: string
    previous: string
    next: string
    page: (n: number) => string
  }
  className?: string
}

export function Pagination({
  page,
  pageCount,
  onPageChange,
  labels,
  className = '',
}: PaginationProps) {
  const safeCount = Math.max(1, pageCount)
  const safePage = Math.min(Math.max(1, page), safeCount)

  const pages = visiblePages(safePage, safeCount)

  return (
    <nav aria-label={labels.nav} className={cx('inline-flex items-center gap-1', className)}>
      <Button
        variant="secondary"
        size="sm"
        disabled={safePage <= 1}
        onClick={() => onPageChange(safePage - 1)}
        aria-label={labels.previous}
      >
        ‹
      </Button>
      {pages.map((p, i) =>
        p === '…' ? (
          <span key={`e-${i}`} className="px-1 text-fg-muted" aria-hidden>
            …
          </span>
        ) : (
          <Button
            key={p}
            variant={p === safePage ? 'primary' : 'ghost'}
            size="sm"
            aria-label={labels.page(p)}
            aria-current={p === safePage ? 'page' : undefined}
            onClick={() => onPageChange(p)}
          >
            {p}
          </Button>
        ),
      )}
      <Button
        variant="secondary"
        size="sm"
        disabled={safePage >= safeCount}
        onClick={() => onPageChange(safePage + 1)}
        aria-label={labels.next}
      >
        ›
      </Button>
    </nav>
  )
}

function visiblePages(page: number, count: number): Array<number | '…'> {
  if (count <= 7) return Array.from({ length: count }, (_, i) => i + 1)
  const set = new Set([1, count, page, page - 1, page + 1, page - 2, page + 2])
  const sorted = [...set].filter((n) => n >= 1 && n <= count).sort((a, b) => a - b)
  const out: Array<number | '…'> = []
  let prev = 0
  for (const n of sorted) {
    if (prev && n - prev > 1) out.push('…')
    out.push(n)
    prev = n
  }
  return out
}
