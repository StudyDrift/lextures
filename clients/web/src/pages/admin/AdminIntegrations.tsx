import { useCallback, useEffect, useId, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { CheckCircle2, XCircle } from 'lucide-react'
import { fetchAdminIntegrations, type IntegrationStatus } from '../../lib/admin-console-api'

function StatusBadge({ on, label }: { on: boolean; label: string }) {
  return (
    <span className="inline-flex items-center gap-1.5 text-sm">
      {on ? (
        <CheckCircle2 className="h-4 w-4 text-green-600" aria-hidden />
      ) : (
        <XCircle className="h-4 w-4 text-slate-400" aria-hidden />
      )}
      <span>{label}</span>
      <span className="sr-only">{on ? 'enabled' : 'disabled'}</span>
    </span>
  )
}

export default function AdminIntegrations() {
  const titleId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [data, setData] = useState<IntegrationStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setData(await fetchAdminIntegrations(orgId))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load integrations.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  useEffect(() => {
    void load()
  }, [load])

  return (
    <div>
      <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-slate-100">
        Integrations
      </h1>
      <p className="mt-1 text-sm text-slate-600 dark:text-slate-400">
        Read-only status for SSO, provisioning, SIS, and webhooks.
      </p>

      {error ? (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      ) : null}

      {loading ? (
        <p className="mt-6 text-sm text-slate-500">Loading…</p>
      ) : data ? (
        <div className="mt-6 grid gap-4 md:grid-cols-2">
          <section className="rounded-xl border border-slate-200 p-4 dark:border-neutral-800">
            <h2 className="font-medium text-slate-900 dark:text-slate-100">Single sign-on</h2>
            <ul className="mt-3 space-y-2">
              <li>
                <StatusBadge on={data.sso.saml} label="SAML 2.0" />
              </li>
              <li>
                <StatusBadge on={data.sso.oidc} label="OIDC" />
              </li>
              <li>
                <StatusBadge on={data.sso.clever} label="Clever" />
              </li>
              <li>
                <StatusBadge on={data.sso.classlink} label="ClassLink" />
              </li>
            </ul>
          </section>
          <section className="rounded-xl border border-slate-200 p-4 dark:border-neutral-800">
            <h2 className="font-medium text-slate-900 dark:text-slate-100">Provisioning</h2>
            <ul className="mt-3 space-y-2">
              <li>
                <StatusBadge on={data.oneRoster.enabled} label="OneRoster" />
              </li>
              <li>
                <StatusBadge on={data.scim.enabled} label="SCIM 2.0" />
              </li>
            </ul>
          </section>
          <section className="rounded-xl border border-slate-200 p-4 dark:border-neutral-800">
            <h2 className="font-medium text-slate-900 dark:text-slate-100">SIS</h2>
            <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
              <StatusBadge on={data.sis.enabled} label="SIS integration" />
            </p>
            <p className="mt-2 text-sm tabular-nums text-slate-600 dark:text-slate-400">
              Active connections: {data.sis.activeConnections}
            </p>
          </section>
          <section className="rounded-xl border border-slate-200 p-4 dark:border-neutral-800">
            <h2 className="font-medium text-slate-900 dark:text-slate-100">Webhooks</h2>
            <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
              <StatusBadge on={data.webhooks.enabled} label="Outbound webhooks" />
            </p>
            <p className="mt-2 text-sm tabular-nums text-slate-600 dark:text-slate-400">
              Active subscriptions: {data.webhooks.subscriptions}
            </p>
          </section>
        </div>
      ) : null}
    </div>
  )
}
