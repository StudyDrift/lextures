import { useCallback, useEffect, useId, useState } from 'react'
import { Link } from 'react-router-dom'
import { ExternalLink, Loader2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { fetchMyEntitlements, fetchMyTransactions, formatMoney, openBillingPortal, type Entitlement, type Transaction } from '../../lib/billing-api'
import { invoiceDownloadUrl } from '../../lib/tax-api'
import { authorizedFetch } from '../../lib/api'
import { LmsPage } from './lms-page'

type MeProfile = { id: string; email: string }

function entitlementLabel(e: Entitlement): string {
  switch (e.entitlementType) {
    case 'course_purchase':
      return 'Course purchase'
    case 'subscription_monthly':
      return 'Monthly subscription'
    case 'subscription_annual':
      return 'Annual subscription'
    default:
      return e.entitlementType
  }
}

export default function BillingSettingsPage() {
  const titleId = useId()
  const { ffStripeBilling, ffPaymentsEnabled, ffCourseMarketplace, loading: featuresLoading } = usePlatformFeatures()
  const [entitlements, setEntitlements] = useState<Entitlement[]>([])
  const [transactions, setTransactions] = useState<Transaction[]>([])
  const [me, setMe] = useState<MeProfile | null>(null)
  const [loading, setLoading] = useState(false)
  const [portalLoading, setPortalLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [items, txItems, meRes] = await Promise.all([
        fetchMyEntitlements(),
        ffPaymentsEnabled ? fetchMyTransactions() : Promise.resolve([]),
        authorizedFetch('/api/v1/me'),
      ])
      setEntitlements(items)
      setTransactions(txItems)
      if (meRes.ok) {
        setMe((await meRes.json()) as MeProfile)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load billing settings.')
    } finally {
      setLoading(false)
    }
  }, [ffPaymentsEnabled])

  useEffect(() => {
    if (featuresLoading || (!ffStripeBilling && !ffPaymentsEnabled)) return
    void load()
  }, [featuresLoading, ffStripeBilling, ffPaymentsEnabled, load])

  const activeSubscription = entitlements.find((e) => e.entitlementType.startsWith('subscription'))

  async function handleManageSubscription() {
    setPortalLoading(true)
    setError(null)
    try {
      const url = await openBillingPortal(`${window.location.origin}/me/billing`)
      window.open(url, '_blank', 'noopener,noreferrer')
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not open billing portal.')
    } finally {
      setPortalLoading(false)
    }
  }

  if (featuresLoading) {
    return <p>Loading…</p>
  }

  if (!ffStripeBilling && !ffPaymentsEnabled) {
    return (
      <LmsPage title="Billing">
        <p role="alert">Billing is not enabled for this institution.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Billing">
      <div className="mx-auto max-w-3xl space-y-6">
        <header>
          <h1 id={titleId} className="text-2xl font-semibold text-fg-default">
            Billing
          </h1>
          <p className="mt-2 text-sm text-fg-muted">
            Manage your subscription, payment method, and purchase history.
          </p>
        </header>

        {error ? (
          <p
            role="alert"
            className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-200"
          >
            {error}
          </p>
        ) : null}

        <section className="rounded-xl border border-border-default bg-surface-raised p-5 shadow-sm dark:border-border-subtle dark:bg-surface-raised">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-lg font-medium text-fg-default">Subscription</h2>
              {activeSubscription ? (
                <p className="mt-1 text-sm text-emerald-700 dark:text-emerald-300">
                  Active — {entitlementLabel(activeSubscription)}
                </p>
              ) : (
                <p className="mt-1 text-sm text-fg-muted">No active subscription</p>
              )}
            </div>
            <button
              type="button"
              onClick={() => void handleManageSubscription()}
              disabled={portalLoading}
              className="inline-flex items-center gap-2 rounded-lg bg-accent-solid px-4 py-2 text-sm font-medium text-white hover:bg-accent disabled:opacity-60"
            >
              {portalLoading ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
              Manage subscription
              <ExternalLink className="h-4 w-4" aria-hidden />
            </button>
          </div>
        </section>

        <section className="rounded-xl border border-border-default bg-surface-raised p-5 shadow-sm dark:border-border-subtle dark:bg-surface-raised">
          <h2 className="text-lg font-medium text-fg-default">Purchase history</h2>
          {ffCourseMarketplace ? (
            <p className="mt-1 text-sm text-fg-muted">
              <Link
                to="/me/purchases"
                className="font-medium text-accent-fg hover:text-indigo-500 dark:text-indigo-300"
              >
                View marketplace purchases
              </Link>
            </p>
          ) : null}
          {loading ? (
            <p className="mt-4 text-sm text-fg-muted">Loading…</p>
          ) : transactions.length > 0 ? (
            <div className="mt-4 overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border-default text-fg-muted dark:border-border-default dark:text-fg-muted">
                    <th className="py-2 pr-4 font-medium">Provider</th>
                    <th className="py-2 pr-4 font-medium">Amount</th>
                    <th className="py-2 pr-4 font-medium">Date</th>
                    <th className="py-2 font-medium">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {transactions.map((tx) => (
                    <tr key={tx.id} className="border-b border-border-subtle">
                      <td className="py-3 pr-4 capitalize">{tx.provider}</td>
                      <td className="py-3 pr-4">{formatMoney(tx.amountCents, tx.currency)}</td>
                      <td className="py-3 pr-4">{new Date(tx.createdAt).toLocaleDateString()}</td>
                      <td className="py-3 capitalize">{tx.status}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : entitlements.length === 0 ? (
            <p className="mt-4 text-sm text-fg-muted">No purchases yet.</p>
          ) : (
            <div className="mt-4 overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-border-default text-fg-muted dark:border-border-default dark:text-fg-muted">
                    <th className="py-2 pr-4 font-medium">Type</th>
                    <th className="py-2 pr-4 font-medium">Amount</th>
                    <th className="py-2 pr-4 font-medium">Tax</th>
                    <th className="py-2 pr-4 font-medium">Valid from</th>
                    <th className="py-2 pr-4 font-medium">Status</th>
                    <th className="py-2 font-medium">Invoice</th>
                  </tr>
                </thead>
                <tbody>
                  {entitlements.map((e) => (
                    <tr key={e.id} className="border-b border-border-subtle">
                      <td className="py-3 pr-4">{entitlementLabel(e)}</td>
                      <td className="py-3 pr-4">{formatMoney(e.amountPaidCents, e.currency)}</td>
                      <td className="py-3 pr-4">
                        {e.taxAmountCents ? formatMoney(e.taxAmountCents, e.currency) : '—'}
                      </td>
                      <td className="py-3 pr-4">{new Date(e.validFrom).toLocaleDateString()}</td>
                      <td className="py-3 pr-4 capitalize">{e.status}</td>
                      <td className="py-3">
                        {e.invoiceId ? (
                          <a
                            href={invoiceDownloadUrl(e.invoiceId)}
                            className="text-sm font-medium text-accent-fg hover:underline"
                            onClick={async (ev) => {
                              ev.preventDefault()
                              const res = await authorizedFetch(`/api/v1/invoices/${e.invoiceId}`)
                              if (!res.ok) return
                              const blob = await res.blob()
                              const url = URL.createObjectURL(blob)
                              const a = document.createElement('a')
                              a.href = url
                              a.download = `invoice-${e.invoiceId}.pdf`
                              a.click()
                              URL.revokeObjectURL(url)
                            }}
                          >
                            Download
                          </a>
                        ) : (
                          '—'
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {me ? (
            <p className="mt-4 text-xs text-fg-subtle">Signed in as {me.email}</p>
          ) : null}
        </section>
      </div>
    </LmsPage>
  )
}
