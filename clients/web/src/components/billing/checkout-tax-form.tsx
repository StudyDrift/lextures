import { useCallback, useEffect, useId, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { formatMoney } from '../../lib/billing-api'
import { fetchTaxQuote, validateTaxID, type TaxAddress, type TaxQuote } from '../../lib/tax-api'

type Props = {
  courseId: string
  onQuoteChange?: (quote: TaxQuote | null) => void
}

const COUNTRY_OPTIONS = [
  { code: 'US', label: 'United States' },
  { code: 'GB', label: 'United Kingdom' },
  { code: 'DE', label: 'Germany' },
  { code: 'FR', label: 'France' },
  { code: 'CA', label: 'Canada' },
  { code: 'AU', label: 'Australia' },
]

export function CheckoutTaxForm({ courseId, onQuoteChange }: Props) {
  const countryId = useId()
  const regionId = useId()
  const vatId = useId()
  const [country, setCountry] = useState('')
  const [region, setRegion] = useState('')
  const [vatNumber, setVatNumber] = useState('')
  const [isBusiness, setIsBusiness] = useState(false)
  const [quote, setQuote] = useState<TaxQuote | null>(null)
  const [status, setStatus] = useState<'idle' | 'computing' | 'computed' | 'error'>('idle')
  const [error, setError] = useState<string | null>(null)
  const [vatMessage, setVatMessage] = useState<string | null>(null)

  const address = useCallback((): TaxAddress => ({
    country,
    region: region || undefined,
  }), [country, region])

  const computeQuote = useCallback(async () => {
    if (!country) {
      setQuote(null)
      setStatus('idle')
      onQuoteChange?.(null)
      return
    }
    setStatus('computing')
    setError(null)
    try {
      const result = await fetchTaxQuote({
        courseId,
        address: address(),
        taxId: isBusiness ? vatNumber : undefined,
      })
      setQuote(result)
      setStatus('computed')
      onQuoteChange?.(result)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Tax calculation failed.')
      setStatus('error')
      setQuote(null)
      onQuoteChange?.(null)
    }
  }, [address, courseId, country, isBusiness, onQuoteChange, vatNumber])

  useEffect(() => {
    const timer = setTimeout(() => void computeQuote(), 400)
    return () => clearTimeout(timer)
  }, [computeQuote])

  async function handleValidateVAT() {
    if (!vatNumber.trim() || !country) return
    try {
      const result = await validateTaxID({ courseId, address: address(), taxId: vatNumber })
      setVatMessage(result.message ?? (result.valid ? 'Tax ID accepted.' : 'Invalid tax ID.'))
    } catch (e) {
      setVatMessage(e instanceof Error ? e.message : 'Validation failed.')
    }
  }

  return (
    <div className="space-y-4 rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
      <h3 className="text-base font-medium text-slate-900 dark:text-neutral-100">Billing address</h3>
      <p className="text-sm text-slate-600 dark:text-neutral-400">
        Enter your country so we can calculate applicable tax before checkout.
      </p>

      <div className="grid gap-4 sm:grid-cols-2">
        <div>
          <label htmlFor={countryId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
            Country
          </label>
          <select
            id={countryId}
            value={country}
            onChange={(e) => setCountry(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
            required
          >
            <option value="">Select country</option>
            {COUNTRY_OPTIONS.map((c) => (
              <option key={c.code} value={c.code}>
                {c.label}
              </option>
            ))}
          </select>
        </div>
        <div>
          <label htmlFor={regionId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
            State / region
          </label>
          <input
            id={regionId}
            type="text"
            value={region}
            onChange={(e) => setRegion(e.target.value)}
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
            autoComplete="address-level1"
          />
        </div>
      </div>

      <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
        <input
          type="checkbox"
          checked={isBusiness}
          onChange={(e) => setIsBusiness(e.target.checked)}
          className="rounded border-slate-300"
        />
        I&apos;m a business — add VAT / GST ID
      </label>

      {isBusiness ? (
        <div>
          <label htmlFor={vatId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
            VAT / GST ID
          </label>
          <div className="mt-1 flex gap-2">
            <input
              id={vatId}
              type="text"
              value={vatNumber}
              onChange={(e) => setVatNumber(e.target.value)}
              className="flex-1 rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
            />
            <button
              type="button"
              onClick={() => void handleValidateVAT()}
              className="rounded-lg border border-slate-300 px-3 py-2 text-sm hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
            >
              Validate
            </button>
          </div>
          {vatMessage ? (
            <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400" role="status">
              {vatMessage}
            </p>
          ) : null}
        </div>
      ) : null}

      {error ? (
        <p role="alert" className="text-sm text-red-700 dark:text-red-300">
          {error}
        </p>
      ) : null}

      {status === 'computing' ? (
        <p className="flex items-center gap-2 text-sm text-slate-600 dark:text-neutral-400" role="status">
          <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
          Computing tax…
        </p>
      ) : null}

      {quote && status === 'computed' ? (
        <div className="rounded-lg bg-slate-50 p-4 dark:bg-neutral-950" aria-live="polite">
          <dl className="space-y-2 text-sm">
            {quote.lines.map((line) => (
              <div key={line.label} className="flex justify-between">
                <dt className="text-slate-600 dark:text-neutral-400">{line.label}</dt>
                <dd className="font-medium text-slate-900 dark:text-neutral-100">
                  {formatMoney(line.amountCents, quote.currency)}
                </dd>
              </div>
            ))}
            <div className="flex justify-between border-t border-slate-200 pt-2 dark:border-neutral-800">
              <dt className="font-medium text-slate-900 dark:text-neutral-100">Total</dt>
              <dd className="font-semibold text-slate-900 dark:text-neutral-100">
                {formatMoney(quote.totalCents, quote.currency)}
              </dd>
            </div>
          </dl>
          {quote.reverseCharge ? (
            <p className="mt-2 text-xs text-emerald-700 dark:text-emerald-300">Reverse charge applies.</p>
          ) : null}
        </div>
      ) : null}
    </div>
  )
}