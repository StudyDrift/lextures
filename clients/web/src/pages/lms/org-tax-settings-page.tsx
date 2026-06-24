import { useCallback, useEffect, useId, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { fetchOrgTaxSettings, fetchTaxReport, saveOrgTaxSettings, type OrgTaxSettings } from '../../lib/tax-api'
import { authorizedFetch } from '../../lib/api'
import { formatMoney } from '../../lib/billing-api'
import { LmsPage } from './lms-page'

type MeOrg = { orgId?: string }

export default function OrgTaxSettingsPage() {
  const titleId = useId()
  const { ffTaxCollection, loading: featuresLoading } = usePlatformFeatures()
  const [orgId, setOrgId] = useState<string | null>(null)
  const [settings, setSettings] = useState<OrgTaxSettings | null>(null)
  const [period, setPeriod] = useState(() => {
    const now = new Date()
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  })
  const [reportRows, setReportRows] = useState<Awaited<ReturnType<typeof fetchTaxReport>>['rows']>([])
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const meRes = await authorizedFetch('/api/v1/me')
      if (!meRes.ok) throw new Error('Could not load profile.')
      const me = (await meRes.json()) as MeOrg
      const oid = me.orgId
      if (!oid) throw new Error('No organization found.')
      setOrgId(oid)
      const [s, report] = await Promise.all([
        fetchOrgTaxSettings(oid),
        fetchTaxReport(oid, period),
      ])
      setSettings(s)
      setReportRows(report.rows)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load tax settings.')
    } finally {
      setLoading(false)
    }
  }, [period])

  useEffect(() => {
    if (!featuresLoading && ffTaxCollection) void load()
  }, [featuresLoading, ffTaxCollection, load])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    if (!orgId || !settings) return
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const updated = await saveOrgTaxSettings(orgId, settings)
      setSettings(updated)
      setSaved(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save settings.')
    } finally {
      setSaving(false)
    }
  }

  if (featuresLoading) return <p>Loading…</p>
  if (!ffTaxCollection) {
    return (
      <LmsPage title="Tax settings">
        <p role="alert">Tax collection is not enabled for this institution.</p>
      </LmsPage>
    )
  }

  return (
    <LmsPage title="Tax settings">
      <div className="mx-auto max-w-3xl space-y-6">
        <header>
          <h1 id={titleId} className="text-2xl font-semibold text-slate-900 dark:text-neutral-100">
            Tax settings
          </h1>
          <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">
            Configure Stripe Tax jurisdictions, seller details, and run period reports.
          </p>
        </header>

        {error ? (
          <p role="alert" className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-800 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-200">
            {error}
          </p>
        ) : null}
        {saved ? (
          <p role="status" className="text-sm text-emerald-700 dark:text-emerald-300">
            Settings saved.
          </p>
        ) : null}

        {loading || !settings ? (
          <p className="text-sm text-slate-600 dark:text-neutral-400">Loading…</p>
        ) : (
          <form onSubmit={(e) => void handleSave(e)} className="space-y-6">
            <section className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
              <label className="flex items-center gap-2 text-sm font-medium">
                <input
                  type="checkbox"
                  checked={settings.enabled}
                  onChange={(e) => setSettings({ ...settings, enabled: e.target.checked })}
                />
                Enable tax collection for this organization
              </label>

              <div className="mt-4 grid gap-4 sm:grid-cols-2">
                <div>
                  <label className="block text-sm font-medium">Price display</label>
                  <select
                    value={settings.priceDisplay}
                    onChange={(e) =>
                      setSettings({
                        ...settings,
                        priceDisplay: e.target.value as 'inclusive' | 'exclusive',
                      })
                    }
                    className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                  >
                    <option value="exclusive">Tax exclusive (US norm)</option>
                    <option value="inclusive">Tax inclusive (EU norm)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium">Default tax category</label>
                  <input
                    type="text"
                    value={settings.defaultTaxCategory}
                    onChange={(e) => setSettings({ ...settings, defaultTaxCategory: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </div>
              </div>

              <div className="mt-4">
                <label className="block text-sm font-medium">Registered jurisdictions (comma-separated)</label>
                <input
                  type="text"
                  value={settings.registeredJurisdictions.join(', ')}
                  onChange={(e) =>
                    setSettings({
                      ...settings,
                      registeredJurisdictions: e.target.value
                        .split(',')
                        .map((s) => s.trim().toUpperCase())
                        .filter(Boolean),
                    })
                  }
                  placeholder="GB, DE, US-CA"
                  className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                />
              </div>

              <div className="mt-4 grid gap-4">
                <div>
                  <label className="block text-sm font-medium">Seller name</label>
                  <input
                    type="text"
                    value={settings.sellerName}
                    onChange={(e) => setSettings({ ...settings, sellerName: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium">Seller address</label>
                  <textarea
                    value={settings.sellerAddress}
                    onChange={(e) => setSettings({ ...settings, sellerAddress: e.target.value })}
                    rows={3}
                    className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium">Seller tax ID</label>
                  <input
                    type="text"
                    value={settings.sellerTaxId}
                    onChange={(e) => setSettings({ ...settings, sellerTaxId: e.target.value })}
                    className="mt-1 w-full rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
                  />
                </div>
              </div>

              <button
                type="submit"
                disabled={saving}
                className="mt-4 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-700 disabled:opacity-60"
              >
                {saving ? 'Saving…' : 'Save settings'}
              </button>
            </section>
          </form>
        )}

        <section className="rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex flex-wrap items-end gap-3">
            <div>
              <label className="block text-sm font-medium">Report period</label>
              <input
                type="month"
                value={period}
                onChange={(e) => setPeriod(e.target.value)}
                className="mt-1 rounded-lg border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
              />
            </div>
            <button
              type="button"
              onClick={() => void load()}
              className="rounded-lg border border-slate-300 px-4 py-2 text-sm hover:bg-slate-50 dark:border-neutral-700 dark:hover:bg-neutral-800"
            >
              Refresh report
            </button>
          </div>

          {reportRows.length === 0 ? (
            <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">No tax collected in this period.</p>
          ) : (
            <div className="mt-4 overflow-x-auto">
              <table className="min-w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-slate-200 text-slate-500 dark:border-neutral-700">
                    <th className="py-2 pr-4 font-medium">Jurisdiction</th>
                    <th className="py-2 pr-4 font-medium">Type</th>
                    <th className="py-2 pr-4 font-medium">Transactions</th>
                    <th className="py-2 pr-4 font-medium">Tax collected</th>
                    <th className="py-2 font-medium">Subtotal</th>
                  </tr>
                </thead>
                <tbody>
                  {reportRows.map((row) => (
                    <tr key={`${row.jurisdiction}-${row.taxType}`} className="border-b border-slate-100 dark:border-neutral-800">
                      <td className="py-3 pr-4">{row.jurisdiction}</td>
                      <td className="py-3 pr-4 uppercase">{row.taxType}</td>
                      <td className="py-3 pr-4">{row.transactionCount}</td>
                      <td className="py-3 pr-4">{formatMoney(row.taxCollectedCents, 'usd')}</td>
                      <td className="py-3">{formatMoney(row.subtotalCents, 'usd')}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
      </div>
    </LmsPage>
  )
}