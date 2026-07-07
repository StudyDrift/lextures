import { useCallback, useEffect, useId, useState, type FormEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import { useConfirm } from '../../components/use-confirm'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createWebhookSubscription,
  deleteWebhookSubscription,
  eventTypeLabel,
  fetchWebhookDeliveries,
  fetchWebhookEventTypes,
  fetchWebhookSubscriptions,
  testWebhookSubscription,
  updateWebhookSubscription,
  type WebhookDelivery,
  type WebhookEventGroup,
  type WebhookSubscription,
} from '../../lib/webhooks-api'
import { toastMutationError } from '../../lib/lms-toast'

export default function WebhooksAdminPage() {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const titleId = useId()
  const labelId = useId()
  const urlId = useId()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId') ?? ''
  const { ffWebhooks, loading: featuresLoading } = usePlatformFeatures()
  const [subscriptions, setSubscriptions] = useState<WebhookSubscription[]>([])
  const [groups, setGroups] = useState<WebhookEventGroup[]>([])
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [deliveries, setDeliveries] = useState<WebhookDelivery[]>([])
  const [label, setLabel] = useState('Registrar grade sync')
  const [endpointUrl, setEndpointUrl] = useState('https://hooks.example.edu/lextures')
  const [selectedEvents, setSelectedEvents] = useState<string[]>(['grade.posted'])
  const [testEventType, setTestEventType] = useState('grade.posted')
  const [signingKeyReveal, setSigningKeyReveal] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [busyId, setBusyId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [message, setMessage] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!orgId) return
    setLoading(true)
    setError(null)
    try {
      const [subs, catalog] = await Promise.all([
        fetchWebhookSubscriptions(orgId),
        fetchWebhookEventTypes(orgId),
      ])
      setSubscriptions(subs)
      setGroups(catalog.groups)
      if (catalog.eventTypes.length > 0) {
        setTestEventType(catalog.eventTypes[0] ?? 'grade.posted')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load webhooks.')
    } finally {
      setLoading(false)
    }
  }, [orgId])

  const loadDeliveries = useCallback(
    async (subscriptionId: string) => {
      if (!orgId) return
      try {
        const rows = await fetchWebhookDeliveries(orgId, subscriptionId)
        setDeliveries(rows)
      } catch (e) {
        setError(e instanceof Error ? e.message : 'Failed to load delivery log.')
      }
    },
    [orgId],
  )

  useEffect(() => {
    if (featuresLoading || !ffWebhooks || !orgId) return
    void load()
  }, [featuresLoading, ffWebhooks, load, orgId])

  useEffect(() => {
    if (selectedId) void loadDeliveries(selectedId)
    else setDeliveries([])
  }, [selectedId, loadDeliveries])

  function toggleEvent(eventType: string) {
    setSelectedEvents((prev) =>
      prev.includes(eventType) ? prev.filter((e) => e !== eventType) : [...prev, eventType],
    )
  }

  async function handleCreate(e: FormEvent) {
    e.preventDefault()
    if (!orgId || selectedEvents.length === 0) return
    setBusyId('create')
    setError(null)
    setMessage(null)
    setSigningKeyReveal(null)
    try {
      const created = await createWebhookSubscription(orgId, {
        label: label.trim(),
        endpointUrl: endpointUrl.trim(),
        eventTypes: selectedEvents,
      })
      setSigningKeyReveal(created.signingKey)
      setMessage('Webhook subscription created. Copy the signing key now — it will not be shown again.')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create subscription.')
    } finally {
      setBusyId(null)
    }
  }

  async function handleTest(sub: WebhookSubscription) {
    if (!orgId) return
    setBusyId(`test-${sub.id}`)
    setError(null)
    setMessage(null)
    try {
      const delivery = await testWebhookSubscription(orgId, sub.id, testEventType)
      setSelectedId(sub.id)
      setMessage(
        delivery.status === 'delivered'
          ? 'Test event delivered successfully.'
          : `Test delivery status: ${delivery.status}`,
      )
      await loadDeliveries(sub.id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Test delivery failed.')
    } finally {
      setBusyId(null)
    }
  }

  async function handleReactivate(sub: WebhookSubscription) {
    if (!orgId) return
    setBusyId(`reactivate-${sub.id}`)
    try {
      await updateWebhookSubscription(orgId, sub.id, { reactivate: true })
      setMessage('Subscription reactivated.')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to reactivate subscription.')
    } finally {
      setBusyId(null)
    }
  }

  async function handleDelete(sub: WebhookSubscription) {
    if (!orgId) return
    if (
      !(await confirm({
        title: t('admin.deleteWebhook.title', { label: sub.label }),
        variant: 'danger',
      }))
    ) {
      return
    }
    setBusyId(`delete-${sub.id}`)
    try {
      await deleteWebhookSubscription(orgId, sub.id)
      if (selectedId === sub.id) setSelectedId(null)
      await load()
    } catch (err) {
      toastMutationError(err instanceof Error ? err.message : 'Failed to delete subscription.')
    } finally {
      setBusyId(null)
    }
  }

  if (featuresLoading) {
    return <p className="text-sm text-slate-600 dark:text-neutral-400">Loading platform features…</p>
  }

  if (!ffWebhooks) {
    return (
      <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100">
        Outbound webhooks are disabled. Enable <strong>Outbound webhooks</strong> in Settings → Global
        platform.
      </div>
    )
  }

  if (!orgId) {
    return (
      <p className="text-sm text-slate-600 dark:text-neutral-400">
        Open this page from Admin with an organization selected (add <code>?orgId=…</code> to the URL).
      </p>
    )
  }

  return (
    <div className="space-y-8">
      <header>
        <h1 id={titleId} className="text-xl font-semibold text-slate-900 dark:text-neutral-100">
          Webhooks
        </h1>
        <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
          Register HTTPS endpoints to receive signed LMS event notifications.
        </p>
      </header>

      {error ? (
        <div role="alert" className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-200">
          {error}
        </div>
      ) : null}
      {message ? (
        <div role="status" className="rounded-md border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-900 dark:border-emerald-900/50 dark:bg-emerald-950/30 dark:text-emerald-100">
          {message}
        </div>
      ) : null}
      {signingKeyReveal ? (
        <div className="rounded-md border border-sky-200 bg-sky-50 p-3 text-sm dark:border-sky-900/50 dark:bg-sky-950/30">
          <p className="font-medium text-sky-900 dark:text-sky-100">Signing key (shown once)</p>
          <code className="mt-2 block break-all rounded bg-white/80 px-2 py-1 font-mono text-xs dark:bg-neutral-900">
            {signingKeyReveal}
          </code>
        </div>
      ) : null}

      <section aria-labelledby={titleId} className="rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Create subscription</h2>
        <form className="mt-4 space-y-4" onSubmit={(e) => void handleCreate(e)}>
          <div>
            <label htmlFor={labelId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Label
            </label>
            <input
              id={labelId}
              required
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              className="mt-1 w-full max-w-lg rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
            />
          </div>
          <div>
            <label htmlFor={urlId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Endpoint URL (HTTPS only)
            </label>
            <input
              id={urlId}
              required
              type="url"
              value={endpointUrl}
              onChange={(e) => setEndpointUrl(e.target.value)}
              className="mt-1 w-full max-w-2xl rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-950"
            />
          </div>
          <fieldset>
            <legend className="text-sm font-medium text-slate-700 dark:text-neutral-300">Event types</legend>
            <div className="mt-2 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {groups.map((group) => (
                <div key={group.domain}>
                  <p className="text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-500">
                    {group.domain}
                  </p>
                  <ul className="mt-2 space-y-2">
                    {group.types.map((eventType) => (
                      <li key={eventType}>
                        <label className="flex cursor-pointer items-center gap-2 text-sm">
                          <input
                            type="checkbox"
                            checked={selectedEvents.includes(eventType)}
                            onChange={() => toggleEvent(eventType)}
                          />
                          {eventTypeLabel(eventType)}
                        </label>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </fieldset>
          <button
            type="submit"
            disabled={busyId === 'create' || selectedEvents.length === 0}
            className="rounded-md bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
          >
            {busyId === 'create' ? 'Creating…' : 'Create subscription'}
          </button>
        </form>
      </section>

      <section className="rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Subscriptions</h2>
        {loading ? <p className="mt-3 text-sm text-slate-600">Loading…</p> : null}
        <div className="mt-3 overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead>
              <tr className="border-b border-slate-200 dark:border-neutral-800">
                <th scope="col" className="px-2 py-2 font-medium">Label</th>
                <th scope="col" className="px-2 py-2 font-medium">Status</th>
                <th scope="col" className="px-2 py-2 font-medium">Events</th>
                <th scope="col" className="px-2 py-2 font-medium">Endpoint</th>
                <th scope="col" className="px-2 py-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {subscriptions.map((sub) => (
                <tr key={sub.id} className="border-b border-slate-100 dark:border-neutral-800/80">
                  <td className="px-2 py-2">{sub.label}</td>
                  <td className="px-2 py-2">
                    <span
                      className={
                        sub.status === 'active'
                          ? 'rounded-full bg-emerald-100 px-2 py-0.5 text-xs text-emerald-800 dark:bg-emerald-950 dark:text-emerald-200'
                          : 'rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-900 dark:bg-amber-950 dark:text-amber-200'
                      }
                    >
                      {sub.status}
                    </span>
                  </td>
                  <td className="px-2 py-2">{sub.eventTypes.map(eventTypeLabel).join(', ')}</td>
                  <td className="max-w-xs truncate px-2 py-2 font-mono text-xs">{sub.endpointUrl}</td>
                  <td className="px-2 py-2">
                    <div className="flex flex-wrap gap-2">
                      <button
                        type="button"
                        className="text-sky-700 underline dark:text-sky-300"
                        onClick={() => setSelectedId(sub.id)}
                      >
                        Log
                      </button>
                      <button
                        type="button"
                        className="text-sky-700 underline dark:text-sky-300"
                        disabled={busyId === `test-${sub.id}`}
                        onClick={() => void handleTest(sub)}
                      >
                        Test
                      </button>
                      {sub.status === 'paused' ? (
                        <button
                          type="button"
                          className="text-sky-700 underline dark:text-sky-300"
                          disabled={busyId === `reactivate-${sub.id}`}
                          onClick={() => void handleReactivate(sub)}
                        >
                          Reactivate
                        </button>
                      ) : null}
                      <button
                        type="button"
                        className="text-red-700 underline dark:text-red-300"
                        disabled={busyId === `delete-${sub.id}`}
                        onClick={() => void handleDelete(sub)}
                      >
                        Delete
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
              {!loading && subscriptions.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-2 py-4 text-slate-500">
                    No webhook subscriptions yet.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>

      {selectedId ? (
        <section className="rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-800 dark:bg-neutral-900">
          <div className="flex flex-wrap items-end gap-3">
            <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Delivery log</h2>
            <label className="text-sm">
              Test event type{' '}
              <select
                value={testEventType}
                onChange={(e) => setTestEventType(e.target.value)}
                className="ml-1 rounded border border-slate-300 px-2 py-1 dark:border-neutral-700 dark:bg-neutral-950"
              >
                {groups.flatMap((g) => g.types).map((t) => (
                  <option key={t} value={t}>
                    {eventTypeLabel(t)}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="mt-3 overflow-x-auto">
            <table className="min-w-full text-left text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-neutral-800">
                  <th scope="col" className="px-2 py-2 font-medium">Time</th>
                  <th scope="col" className="px-2 py-2 font-medium">Event</th>
                  <th scope="col" className="px-2 py-2 font-medium">Status</th>
                  <th scope="col" className="px-2 py-2 font-medium">HTTP</th>
                  <th scope="col" className="px-2 py-2 font-medium">Retries</th>
                  <th scope="col" className="px-2 py-2 font-medium">Response</th>
                </tr>
              </thead>
              <tbody>
                {deliveries.map((row) => (
                  <tr key={row.id} className="border-b border-slate-100 dark:border-neutral-800/80">
                    <td className="whitespace-nowrap px-2 py-2">{new Date(row.createdAt).toLocaleString()}</td>
                    <td className="px-2 py-2">{eventTypeLabel(row.eventType)}{row.test ? ' (test)' : ''}</td>
                    <td className="px-2 py-2">{row.status}</td>
                    <td className="px-2 py-2">{row.lastHttpStatus ?? '—'}</td>
                    <td className="px-2 py-2">{row.attemptCount}</td>
                    <td className="max-w-md truncate px-2 py-2 font-mono text-xs" title={row.lastResponse ?? ''}>
                      {row.lastResponse ?? '—'}
                    </td>
                  </tr>
                ))}
                {deliveries.length === 0 ? (
                  <tr>
                    <td colSpan={6} className="px-2 py-4 text-slate-500">
                      No deliveries recorded yet.
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </section>
      ) : null}
      {ConfirmDialogHost}
    </div>
  )
}
