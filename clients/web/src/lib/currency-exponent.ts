/** ISO 4217 currencies where Stripe's smallest unit is the major unit. */
export const ZERO_DECIMAL_CURRENCIES = new Set(['jpy'])

export function isZeroDecimalCurrency(currency: string): boolean {
  return ZERO_DECIMAL_CURRENCIES.has(currency.toLowerCase().trim())
}

export function minorUnitFactor(currency: string): number {
  return isZeroDecimalCurrency(currency) ? 1 : 100
}

export function minorUnitsToMajorUnits(minor: number, currency: string): number {
  return minor / minorUnitFactor(currency)
}

export function majorUnitsToMinorUnits(major: number, currency: string): number {
  return Math.round(major * minorUnitFactor(currency))
}

export const MAX_MARKETPLACE_PRICE_MAJOR_ZERO_DECIMAL = 99_999

export function stripeMinimumMinorUnits(_currency: string): number {
  return 50
}

export function maxCatalogMinorUnits(currency: string): number {
  return isZeroDecimalCurrency(currency) ? 99_999 : 9_999_999
}