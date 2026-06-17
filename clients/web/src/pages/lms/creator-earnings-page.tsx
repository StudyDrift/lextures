import { useCallback, useEffect, useId, useState } from 'react'
import { DollarSign, Link2, Wallet } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createAffiliateCode,
  fetchAffiliateCodes,
  fetchCreatorEarnings,
  fetchCreatorLedger,
  formatMoney,
  startConnectOnboarding,
  type AffiliateCode,
  type EarningsSummary,
  type LedgerEntry,
} from '../../lib/revenue-share-api'

export default function CreatorEarningsPage() {
  const { ffRevenueShare, loading: featuresLoading } = usePlatformFeatures()
  const [summary, setSummary] = useState<EarningsSummary | null>(null)
  const [ledger, setLedger] = useState<LedgerEntry[]>([])
  const [codes, setCodes] = useState<AffiliateCode[]>([])
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)
  const pendingId = useId()
  const paidId = useId()
  const tableCaptionId = useId()

  const load = useCallback(async () => {
    setError(null)
    try {
      const [s, l, c] = await Promise.all([
        fetchCreatorEarnings(),
        fetchCreatorLedger(),
        fetchAffiliateCodes(),
      ])
      setSummary(s)
      setLedger(l)
      setCodes(c)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Could not load earnings.')
    }
  }, [])

  useEffect(() => {
    if (!featuresLoading && ffRevenueShare) void load()
  }, [featuresLoading, ffRevenueShare, load])

  async function handleConnect() {
    setBusy(true)
    setError(null)
    try {
      const url = await startConnectOnboarding()
      window.location.href = url
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Connect onboarding failed.')
      setBusy(false)
    }
  }

  async function handleNewCode() {
    setBusy(true)
    setError(null)
    try {
      await createAffiliateCode()
      await load()
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Could not create referral link.')
    } finally {
      setBusy(false)
    }
  }

  if (featuresLoading) {
    return <p className="p-6 text-sm text-slate-600">Loading…</p>
  }

  if (!ffRevenueShare) {
    return (
      <div className="p-6">
        <p className="text-sm text-slate-600">Creator earnings are not enabled on this platform.</p>
      </div>
    )
  }

  const currency = summary?.currency ?? 'usd'
  const hasEarnings = (summary?.pendingCents ?? 0) > 0 || (summary?.paidCents ?? 0) > 0 || ledger.length > 0

  return (
    <div className="mx-auto max-w-4xl space-y-8 p-6">
      <header>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-neutral-100">Creator earnings</h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Track sales revenue, affiliate commissions, and payouts.
        </p>
      </header>

      {error ? (
        <div role="alert" className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {error}
        </div>
      ) : null}

      {!summary?.connectConfigured ? (
        <section
          aria-label="Stripe Connect onboarding"
          className="rounded-xl border border-amber-200 bg-amber-50 p-5 dark:border-amber-900 dark:bg-amber-950"
        >
          <p className="text-sm text-amber-900 dark:text-amber-100">
            Connect your bank account via Stripe to receive payouts.
          </p>
          <button
            type="button"
            disabled={busy}
            onClick={() => void handleConnect()}
            className="mt-3 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
          >
            Set up payouts
          </button>
        </section>
      ) : null}

      <section aria-labelledby="earnings-summary-heading" className="grid gap-4 sm:grid-cols-2">
        <h2 id="earnings-summary-heading" className="sr-only">
          Earnings summary
        </h2>
        <div className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex items-center gap-2 text-sm text-slate-500">
            <Wallet className="h-4 w-4" aria-hidden="true" />
            Pending balance
          </div>
          <data value={summary?.pendingCents ?? 0} className="mt-2 block text-3xl font-bold" id={pendingId}>
            {formatMoney(summary?.pendingCents ?? 0, currency)}
          </data>
        </div>
        <div className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex items-center gap-2 text-sm text-slate-500">
            <DollarSign className="h-4 w-4" aria-hidden="true" />
            Paid out
          </div>
          <data value={summary?.paidCents ?? 0} className="mt-2 block text-3xl font-bold" id={paidId}>
            {formatMoney(summary?.paidCents ?? 0, currency)}
          </data>
        </div>
      </section>

      {!hasEarnings ? (
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          No earnings yet — publish a course and start selling!
        </p>
      ) : (
        <section aria-labelledby="ledger-heading">
          <h2 id="ledger-heading" className="text-lg font-semibold">
            Recent activity
          </h2>
          <div className="mt-3 overflow-x-auto rounded-xl border border-slate-200 dark:border-neutral-800">
            <table className="min-w-full text-sm" aria-describedby={tableCaptionId}>
              <caption id={tableCaptionId} className="sr-only">
                Earnings ledger with date, type, and amount
              </caption>
              <thead className="bg-slate-50 text-start dark:bg-neutral-900">
                <tr>
                  <th scope="col" className="px-4 py-2 font-medium">
                    Date
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    Type
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    Amount
                  </th>
                  <th scope="col" className="px-4 py-2 font-medium">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody>
                {ledger.map((row) => (
                  <tr key={row.id} className="border-t border-slate-100 dark:border-neutral-800">
                    <td className="px-4 py-2">{new Date(row.createdAt).toLocaleDateString()}</td>
                    <td className="px-4 py-2 capitalize">{row.entryType}</td>
                    <td className="px-4 py-2">
                      <data value={row.amountCents}>{formatMoney(row.amountCents, row.currency)}</data>
                    </td>
                    <td className="px-4 py-2 capitalize">{row.status}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      <section aria-labelledby="affiliate-heading">
        <div className="flex items-center justify-between gap-4">
          <h2 id="affiliate-heading" className="text-lg font-semibold">
            Referral links
          </h2>
          <button
            type="button"
            disabled={busy}
            onClick={() => void handleNewCode()}
            className="inline-flex items-center gap-1 rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
          >
            <Link2 className="h-4 w-4" aria-hidden="true" />
            New link
          </button>
        </div>
        {codes.length === 0 ? (
          <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
            Generate a referral link to earn commission when others purchase through your link.
          </p>
        ) : (
          <ul className="mt-3 space-y-3">
            {codes.map((code) => (
              <li
                key={code.id}
                className="rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900"
              >
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <code className="text-xs text-slate-600 dark:text-neutral-400">{code.url}</code>
                  <button
                    type="button"
                    className="text-sm font-medium text-indigo-600 hover:underline"
                    onClick={() => void navigator.clipboard.writeText(code.url)}
                  >
                    Copy
                  </button>
                </div>
                <p className="mt-2 text-xs text-slate-500">
                  {code.clickCount} clicks · {code.conversions} conversions
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  )
}
