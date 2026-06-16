import { useCallback, useEffect, useId, useState } from 'react'
import { ExternalLink, Loader2 } from 'lucide-react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { fetchMyEntitlements, formatMoney, openBillingPortal, type Entitlement } from '../../lib/billing-api'
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
  const { ffStripeBilling, loading: featuresLoading } = usePlatformFeatures()
  const [entitlements, setEntitlements] = useState<Entitlement[]>([])
  const [me, setMe] = useState<MeProfile | null>(null)
  const [loading, setLoading] = useState(false)
  const [portalLoading, setPortalLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const [items, meRes] = await Promise.all([
        fetchMyEntitlements(),
        authorizedFetch('/api/v1/me'),
      ])
      setEntitlements(items)
      if (meRes.ok) {
        setMe((await meRes.json()) as MeProfile)
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load billing settings.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    if (featuresLoading || !ffStripeBilling) return
    void load()
  }, [featuresLoading, ffStripeBilling, load])

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

  if (!ffStripeBilling) {
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
          <h1 id={titleId} className="text-2xl font-semibold text-slate-900 dark:text-neutral-100">
            Billing
          </h1>
          <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
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

        <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-lg font-medium text-slate-900 dark:text-neutral-100">Subscription</h2>
              {activeSubscription ? (
                <p className="mt-1 text-sm text-emerald-700 dark:text-emerald-300">
                  Active — {entitlementLabel(activeSubscription)}
                </p>
              ) : (
                <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">No active subscription</p>
              )}
            </div>
            <button
              type="button"
              onClick={() => void handleManageSubscription()}
              disabled={portalLoading}
              className="inline-flex items-center gap-2 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-60"
            >
              {portalLoading ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
              Manage subscription
              <ExternalLink className="h-4 w-4" aria-hidden />
            </button>
          </div>
        </section>

        <section className="rounded-xl border border-slate-200 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
          <h2 className="text-lg font-medium text-slate-900 dark:text-neutral-100">Purchase history</h2>
          {loading ? (
            <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">Loading…</p>
          ) : entitlements.length === 0 ? (
            <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">No purchases yet.</p>
          ) : (
            <div className="mt-4 overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
                    <th className="py-2 pr-4 font-medium">Type</th>
                    <th className="py-2 pr-4 font-medium">Amount</th>
                    <th className="py-2 pr-4 font-medium">Valid from</th>
                    <th className="py-2 font-medium">Status</th>
                  </tr>
                </thead>
                <tbody>
                  {entitlements.map((e) => (
                    <tr key={e.id} className="border-b border-slate-100 dark:border-neutral-800">
                      <td className="py-3 pr-4">{entitlementLabel(e)}</td>
                      <td className="py-3 pr-4">{formatMoney(e.amountPaidCents, e.currency)}</td>
                      <td className="py-3 pr-4">{new Date(e.validFrom).toLocaleDateString()}</td>
                      <td className="py-3 capitalize">{e.status}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {me ? (
            <p className="mt-4 text-xs text-slate-500 dark:text-neutral-500">Signed in as {me.email}</p>
          ) : null}
        </section>
      </div>
    </LmsPage>
  )
}
