import { formatMarketplacePrice } from '../../lib/marketplace-price'

type Props = {
  priceCents: number
  priceCurrency: string
  listPriceCents?: number | null
  freeLabel?: string
  locale?: string
  className?: string
  'data-testid'?: string
}

/** Free / price badge with optional list-price strikethrough (plan MKT3 FR-3). */
export function MarketplacePriceBadge({
  priceCents,
  priceCurrency,
  listPriceCents,
  freeLabel = 'Free',
  locale,
  className = '',
  'data-testid': testId = 'marketplace-price',
}: Props) {
  const price = formatMarketplacePrice(priceCents, priceCurrency, locale, freeLabel)
  const showStrike =
    listPriceCents != null && listPriceCents > priceCents && priceCents > 0
  const listPrice = showStrike
    ? formatMarketplacePrice(listPriceCents, priceCurrency, locale, freeLabel)
    : null

  return (
    <span
      className={`inline-flex items-baseline gap-1.5 font-semibold text-slate-900 dark:text-neutral-100 ${className}`}
      data-testid={testId}
    >
      {listPrice ? (
        <span
          className="text-sm font-normal text-slate-500 line-through dark:text-neutral-400"
          aria-hidden="true"
        >
          {listPrice}
        </span>
      ) : null}
      <span>{price}</span>
    </span>
  )
}
