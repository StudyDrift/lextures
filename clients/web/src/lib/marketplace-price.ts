/** Marketplace price helpers (plan MKT2). */

import {
  isZeroDecimalCurrency,
  majorUnitsToMinorUnits,
  maxCatalogMinorUnits,
  MAX_MARKETPLACE_PRICE_MAJOR_ZERO_DECIMAL,
  minorUnitsToMajorUnits,
  stripeMinimumMinorUnits,
} from './currency-exponent'

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

function amountPattern(currency: string): RegExp {
  return isZeroDecimalCurrency(currency) ? /^\d+$/ : /^\d+(\.\d{1,2})?$/
}

function maxMajorUnits(currency: string): number {
  return isZeroDecimalCurrency(currency) ? MAX_MARKETPLACE_PRICE_MAJOR_ZERO_DECIMAL : MAX_MARKETPLACE_PRICE_MAJOR
}

/** Convert major units (e.g. 19.99 USD or 1000 JPY) to Stripe smallest units. */
export function majorUnitsToPriceCents(amount: string, currency = 'usd'): number | null {
  const trimmed = amount.trim()
  if (!trimmed) return 0
  if (!amountPattern(currency).test(trimmed)) return null
  const value = Number.parseFloat(trimmed)
  if (!Number.isFinite(value) || value < 0) return null
  if (value > maxMajorUnits(currency)) return null
  return majorUnitsToMinorUnits(value, currency)
}

/** Convert Stripe smallest units to a major-unit string for form inputs. */
export function priceCentsToMajorUnits(priceCents: number, currency = 'usd'): string {
  if (priceCents <= 0) return ''
  const major = minorUnitsToMajorUnits(priceCents, currency)
  return isZeroDecimalCurrency(currency) ? String(Math.round(major)) : major.toFixed(2)
}

export function formatMarketplacePrice(
  priceCents: number,
  currency: string,
  locale?: string,
  freeLabel = 'Free',
): string {
  if (priceCents <= 0) return freeLabel
  const major = minorUnitsToMajorUnits(priceCents, currency)
  try {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency: currency.toUpperCase(),
    }).format(major)
  } catch {
    return isZeroDecimalCurrency(currency)
      ? `${currency.toUpperCase()} ${Math.round(major)}`
      : `${currency.toUpperCase()} ${major.toFixed(2)}`
  }
}

export function validateMarketplaceAmount(amount: string, currency = 'usd'): string | null {
  if (!amount.trim()) return null
  const cents = majorUnitsToPriceCents(amount, currency)
  if (cents === null) {
    return isZeroDecimalCurrency(currency)
      ? 'Enter a valid whole-number amount.'
      : 'Enter a valid amount with up to two decimal places.'
  }
  if (cents < 0) return 'Price cannot be negative.'
  const min = stripeMinimumMinorUnits(currency)
  if (cents > 0 && cents < min) return 'Paid courses must be at least $0.50 (or equivalent).'
  if (cents > maxCatalogMinorUnits(currency)) {
    return 'Price exceeds the maximum allowed amount.'
  }
  return null
}