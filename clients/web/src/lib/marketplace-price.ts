/** Marketplace price helpers (plan MKT2). */

export const MAX_MARKETPLACE_PRICE_MAJOR = 99_999.99

export const MARKETPLACE_CURRENCIES = [
  { code: 'usd', label: 'USD — US Dollar' },
  { code: 'eur', label: 'EUR — Euro' },
  { code: 'gbp', label: 'GBP — British Pound' },
  { code: 'cad', label: 'CAD — Canadian Dollar' },
  { code: 'aud', label: 'AUD — Australian Dollar' },
  { code: 'jpy', label: 'JPY — Japanese Yen' },
  { code: 'chf', label: 'CHF — Swiss Franc' },
  { code: 'sek', label: 'SEK — Swedish Krona' },
  { code: 'nok', label: 'NOK — Norwegian Krone' },
  { code: 'dkk', label: 'DKK — Danish Krone' },
  { code: 'nzd', label: 'NZD — New Zealand Dollar' },
  { code: 'sgd', label: 'SGD — Singapore Dollar' },
  { code: 'hkd', label: 'HKD — Hong Kong Dollar' },
  { code: 'mxn', label: 'MXN — Mexican Peso' },
] as const

export type MarketplaceCurrencyCode = (typeof MARKETPLACE_CURRENCIES)[number]['code']

/** Convert major units (e.g. 19.99) to integer cents. */
export function majorUnitsToPriceCents(amount: string): number | null {
  const trimmed = amount.trim()
  if (!trimmed) return 0
  if (!/^\d+(\.\d{1,2})?$/.test(trimmed)) return null
  const value = Number.parseFloat(trimmed)
  if (!Number.isFinite(value) || value < 0) return null
  if (value > MAX_MARKETPLACE_PRICE_MAJOR) return null
  return Math.round(value * 100)
}

/** Convert integer cents to a major-unit string for form inputs. */
export function priceCentsToMajorUnits(priceCents: number): string {
  if (priceCents <= 0) return ''
  return (priceCents / 100).toFixed(2)
}

export function formatMarketplacePrice(
  priceCents: number,
  currency: string,
  locale?: string,
  freeLabel = 'Free',
): string {
  if (priceCents <= 0) return freeLabel
  try {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency: currency.toUpperCase(),
    }).format(priceCents / 100)
  } catch {
    return `${currency.toUpperCase()} ${(priceCents / 100).toFixed(2)}`
  }
}

export function validateMarketplaceAmount(amount: string): string | null {
  if (!amount.trim()) return null
  const cents = majorUnitsToPriceCents(amount)
  if (cents === null) return 'Enter a valid amount with up to two decimal places.'
  if (cents < 0) return 'Price cannot be negative.'
  if (cents > 0 && cents < 50) return 'Paid courses must be at least $0.50 (or equivalent).'
  if (cents > Math.round(MAX_MARKETPLACE_PRICE_MAJOR * 100)) {
    return 'Price exceeds the maximum allowed amount.'
  }
  return null
}
