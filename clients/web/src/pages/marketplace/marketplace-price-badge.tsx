import { formatMarketplacePrice } from '../../lib/marketplace-price'

type Props = {
  priceCents: number
  priceCurrency: string
  listPriceCents?: number | null
  freeLabel?: string
  /** Text alternative for struck list price, e.g. "was $40.00" (MKTC.5 a11y). */
  wasLabel?: string | null
  locale?: string
  className?: string
  'data-testid'?: string
}

/** Free / price badge with optional list-price strikethrough (plan MKT3 FR-3 / MKTC.5). */
export function MarketplacePriceBadge({
  priceCents,
  priceCurrency,
  listPriceCents,
  freeLabel = 'Free',
  wasLabel,
  locale,
  className = '',
  'data-testid': testId = 'marketplace-price',
}: Props) {
  const price = formatMarketplacePrice(priceCents, priceCurrency, locale, freeLabel)
  const showStrike =
    listPriceCents != null && listPriceCents > priceCents && (priceCents > 0 || wasLabel != null)
  const listPrice = showStrike
    ? formatMarketplacePrice(listPriceCents!, priceCurrency, locale, freeLabel)
    : null

  return (
    <span
      className={`inline-flex flex-wrap items-baseline gap-1.5 font-semibold text-fg-default ${className}`}
      data-testid={testId}
    >
      {listPrice ? (
        <>
          <span
            className="text-sm font-normal text-fg-muted line-through dark:text-fg-muted"
            aria-hidden="true"
          >
            {listPrice}
          </span>
          {wasLabel ? <span className="sr-only">{wasLabel}</span> : null}
        </>
      ) : null}
      <span>{price}</span>
    </span>
  )
}
