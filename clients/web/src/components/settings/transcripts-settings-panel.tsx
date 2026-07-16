import { useCallback, useEffect, useId, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createAdminTranscriptRecipient,
  fetchAdminTranscriptRecipients,
  fetchAdminTranscriptRequests,
  fetchAdminTranscriptsConfig,
  saveAdminTranscriptsConfig,
  updateAdminTranscriptRecipient,
  type TranscriptRecipient,
  type TranscriptRequest,
  type TranscriptsConfig,
} from '../../lib/transcripts-api'

const SECRET_PLACEHOLDER = '••••••••••••'

export function TranscriptsSettingsPanel() {
  const { ffTranscripts, loading: featuresLoading } = usePlatformFeatures()
  const urlId = useId()
  const secretId = useId()
  const pickupId = useId()
  const [config, setConfig] = useState<TranscriptsConfig | null>(null)
  const [webhookUrl, setWebhookUrl] = useState('')
  const [webhookSecret, setWebhookSecret] = useState('')
  const [pickupInstructions, setPickupInstructions] = useState('')
  const [officialEnabled, setOfficialEnabled] = useState(false)
  const [ordersUiEnabled, setOrdersUiEnabled] = useState(false)
  const [autoApprovalEnabled, setAutoApprovalEnabled] = useState(false)
  const [registrarConsoleEnabled, setRegistrarConsoleEnabled] = useState(false)
  const [consentRequired, setConsentRequired] = useState(true)
  const [recipients, setRecipients] = useState<TranscriptRecipient[]>([])
  const [newRecipientName, setNewRecipientName] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)
  const [failures, setFailures] = useState<TranscriptRequest[]>([])
  const officialId = useId()
  const ordersUiId = useId()
  const autoApprovalId = useId()
  const registrarConsoleId = useId()
  const consentRequiredId = useId()

  const load = useCallback(async () => {
    if (!ffTranscripts) {
      setConfig(null)
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const [cfg, failed, directory] = await Promise.all([
        fetchAdminTranscriptsConfig(),
        fetchAdminTranscriptRequests(),
        fetchAdminTranscriptRecipients().catch(() => [] as TranscriptRecipient[]),
      ])
      setConfig(cfg)
      setWebhookUrl(cfg.webhookUrl)
      setWebhookSecret(cfg.hasWebhookSecret ? SECRET_PLACEHOLDER : '')
      setPickupInstructions(cfg.pickupInstructions ?? '')
      setOfficialEnabled(cfg.officialEnabled === true)
      setOrdersUiEnabled(cfg.ordersUiEnabled === true)
      setAutoApprovalEnabled(cfg.autoApprovalEnabled === true)
      setRegistrarConsoleEnabled(cfg.registrarConsoleEnabled === true)
      setConsentRequired(cfg.consentRequired !== false)
      setFailures(failed)
      setRecipients(directory)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load transcripts settings.')
    } finally {
      setLoading(false)
    }
  }, [ffTranscripts])

  useEffect(() => {
    if (featuresLoading) return
    void load()
  }, [load, featuresLoading])

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    setSaved(false)
    try {
      const payload: {
        webhookUrl: string
        webhookSecret?: string
        pickupInstructions?: string
        officialEnabled?: boolean
        ordersUiEnabled?: boolean
        autoApprovalEnabled?: boolean
        registrarConsoleEnabled?: boolean
        consentRequired?: boolean
      } = {
        webhookUrl: webhookUrl.trim(),
        pickupInstructions: pickupInstructions.trim(),
        officialEnabled,
        ordersUiEnabled,
        autoApprovalEnabled,
        registrarConsoleEnabled,
        consentRequired,
      }
      if (webhookSecret.trim() && webhookSecret !== SECRET_PLACEHOLDER) {
        payload.webhookSecret = webhookSecret.trim()
      }
      const cfg = await saveAdminTranscriptsConfig(payload)
      setConfig(cfg)
      setWebhookSecret(cfg.hasWebhookSecret ? SECRET_PLACEHOLDER : '')
      setPickupInstructions(cfg.pickupInstructions ?? '')
      setOfficialEnabled(cfg.officialEnabled === true)
      setOrdersUiEnabled(cfg.ordersUiEnabled === true)
      setAutoApprovalEnabled(cfg.autoApprovalEnabled === true)
      setRegistrarConsoleEnabled(cfg.registrarConsoleEnabled === true)
      setConsentRequired(cfg.consentRequired !== false)
      setSaved(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Could not save settings.')
    } finally {
      setSaving(false)
    }
  }

  if (featuresLoading) {
    return (
      <section>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Transcripts</h2>
        <p className="mt-4 text-sm text-slate-500">Loading…</p>
      </section>
    )
  }

  if (!ffTranscripts) {
    return (
      <section>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Transcripts</h2>
        <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">
          Transcripts is not enabled for this platform. Turn on{' '}
          <span className="font-medium">Transcripts</span> under Global platform settings.
        </p>
      </section>
    )
  }

  return (
    <section>
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Transcripts</h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Register your institution&apos;s webhook URL. When a student requests an official transcript,
        Lextures sends a POST request to this endpoint with the student&apos;s information.
      </p>

      {error && (
        <p role="alert" className="mt-4 text-sm text-red-600 dark:text-red-400">
          {error}
        </p>
      )}
      {saved && (
        <p role="status" className="mt-4 text-sm text-emerald-600 dark:text-emerald-400">
          Settings saved.
        </p>
      )}

      {loading ? (
        <p className="mt-4 text-sm text-slate-500">Loading configuration…</p>
      ) : (
        <form onSubmit={handleSave} className="mt-6 space-y-4">
          <div>
            <label htmlFor={urlId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Webhook URL
            </label>
            <input
              id={urlId}
              type="url"
              required
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              placeholder="https://sis.example.edu/api/transcript-requests"
              className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
            />
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              POST-only endpoint. Receives JSON with requestId, requestedAt, delivery preferences, and student profile fields.
            </p>
          </div>
          <div>
            <label htmlFor={pickupId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Pickup instructions
            </label>
            <textarea
              id={pickupId}
              rows={4}
              value={pickupInstructions}
              onChange={(e) => setPickupInstructions(e.target.value)}
              placeholder={'Registrar office, Room 101\nMonday–Friday, 9:00 AM–4:00 PM\nBring a photo ID'}
              className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
            />
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              Shown to students who choose in-person pickup. Leave blank to hide the pickup option.
            </p>
          </div>
          <div className="flex items-start gap-3 rounded-md border border-slate-200 px-3 py-3 dark:border-neutral-700">
            <input
              id={officialId}
              type="checkbox"
              checked={officialEnabled}
              onChange={(e) => setOfficialEnabled(e.target.checked)}
              className="mt-1 h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
            />
            <div>
              <label htmlFor={officialId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Enable official transcript generation
              </label>
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                When on, Lextures can issue sealed official transcripts (PDF + PESC XML) from the gradebook.
                Unofficial previews remain available whenever Transcripts is enabled.
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3 rounded-md border border-slate-200 px-3 py-3 dark:border-neutral-700">
            <input
              id={ordersUiId}
              type="checkbox"
              checked={ordersUiEnabled}
              onChange={(e) => setOrdersUiEnabled(e.target.checked)}
              className="mt-1 h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
            />
            <div>
              <label htmlFor={ordersUiId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Multi-recipient order builder
              </label>
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                When on, students use the recipient directory and multi-destination order flow. The legacy
                single-destination request modal remains when this is off.
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3 rounded-md border border-slate-200 px-3 py-3 dark:border-neutral-700">
            <input
              id={autoApprovalId}
              type="checkbox"
              checked={autoApprovalEnabled}
              onChange={(e) => setAutoApprovalEnabled(e.target.checked)}
              className="mt-1 h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
            />
            <div>
              <label htmlFor={autoApprovalId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Auto-approve orders without holds
              </label>
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                When on, submitted orders with no active holds skip the registrar review queue and go straight
                to processing.
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3 rounded-md border border-slate-200 px-3 py-3 dark:border-neutral-700">
            <input
              id={registrarConsoleId}
              type="checkbox"
              checked={registrarConsoleEnabled}
              onChange={(e) => setRegistrarConsoleEnabled(e.target.checked)}
              className="mt-1 h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
            />
            <div>
              <label htmlFor={registrarConsoleId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Registrar fulfillment console
              </label>
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                When on, registrars can open the fulfillment queue at Admin → Transcripts to approve, reject,
                and manage holds.
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3 rounded-md border border-slate-200 px-3 py-3 dark:border-neutral-700">
            <input
              id={consentRequiredId}
              type="checkbox"
              checked={consentRequired}
              onChange={(e) => setConsentRequired(e.target.checked)}
              className="mt-1 h-4 w-4 rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
            />
            <div>
              <label htmlFor={consentRequiredId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                Require FERPA e-signature for third-party releases
              </label>
              <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                When on (recommended), third-party transcript orders stay in pending consent until the student
                or a linked guardian signs a scoped release authorization. Self-delivery is logged without a
                third-party signature.
              </p>
            </div>
          </div>
          <div>
            <label htmlFor={secretId} className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Webhook secret (optional)
            </label>
            <input
              id={secretId}
              type="password"
              autoComplete="off"
              value={webhookSecret}
              onChange={(e) => setWebhookSecret(e.target.value)}
              placeholder={config?.hasWebhookSecret ? SECRET_PLACEHOLDER : 'HMAC signing secret'}
              className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
            />
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              When set, outbound delivery webhooks and inbound SIS hold upserts use an{' '}
              <code className="font-mono">X-Lextures-Signature</code> header (HMAC-SHA256 of the body).
            </p>
          </div>
          <button
            type="submit"
            disabled={saving}
            className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save configuration'}
          </button>
        </form>
      )}

      {!loading && (
        <div className="mt-8">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Recipient directory</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            Curate institutions and employers students can send transcripts to. Global seeded rows appear
            alongside your organization&apos;s entries.
          </p>
          <form
            className="mt-3 flex flex-wrap gap-2"
            onSubmit={(e) => {
              e.preventDefault()
              const name = newRecipientName.trim()
              if (!name) return
              void createAdminTranscriptRecipient({
                type: 'institution',
                name,
                capabilities: ['electronic_pdf', 'secure_link_email', 'postal_mail'],
                verified: true,
              })
                .then((rec) => {
                  setRecipients((prev) => [rec, ...prev])
                  setNewRecipientName('')
                })
                .catch((err: unknown) => {
                  setError(err instanceof Error ? err.message : 'Could not add recipient.')
                })
            }}
          >
            <input
              value={newRecipientName}
              onChange={(e) => setNewRecipientName(e.target.value)}
              placeholder="Institution or employer name"
              className="min-w-[16rem] flex-1 rounded-md border border-slate-300 px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
            />
            <button
              type="submit"
              className="rounded-md border border-slate-300 px-3 py-2 text-sm font-medium dark:border-neutral-700"
            >
              Add recipient
            </button>
          </form>
          {recipients.length === 0 ? (
            <p className="mt-3 text-sm text-slate-500">No recipients yet.</p>
          ) : (
            <ul className="mt-3 divide-y divide-slate-200 rounded-md border border-slate-200 dark:divide-neutral-800 dark:border-neutral-700">
              {recipients.slice(0, 40).map((rec) => (
                <li key={rec.id} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                  <div>
                    <p className="font-medium text-slate-900 dark:text-neutral-50">{rec.name}</p>
                    <p className="text-xs text-slate-500">
                      {rec.type}
                      {rec.verified ? ' · verified' : ''}
                      {rec.active ? '' : ' · inactive'}
                    </p>
                  </div>
                  <button
                    type="button"
                    className="text-xs font-medium text-indigo-600 hover:underline dark:text-indigo-400"
                    onClick={() => {
                      void updateAdminTranscriptRecipient(rec.id, { active: !rec.active, verified: rec.verified })
                        .then((updated) => {
                          setRecipients((prev) => prev.map((r) => (r.id === updated.id ? updated : r)))
                        })
                        .catch((err: unknown) => {
                          setError(err instanceof Error ? err.message : 'Could not update recipient.')
                        })
                    }}
                  >
                    {rec.active ? 'Deactivate' : 'Activate'}
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {!loading && failures.length > 0 && (
        <div className="mt-8">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            Delivery failures
          </h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            These requests failed to deliver to your webhook endpoint.
          </p>
          <div className="mt-3 overflow-x-auto rounded-md border border-slate-200 dark:border-neutral-700">
            <table className="min-w-full divide-y divide-slate-200 text-sm dark:divide-neutral-700">
              <thead className="bg-slate-50 dark:bg-neutral-800">
                <tr>
                  <th className="px-3 py-2 text-start text-xs font-medium text-slate-500 dark:text-neutral-400">
                    Requested
                  </th>
                  <th className="px-3 py-2 text-start text-xs font-medium text-slate-500 dark:text-neutral-400">
                    Error
                  </th>
                  <th className="px-3 py-2 text-start text-xs font-medium text-slate-500 dark:text-neutral-400">
                    HTTP status
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100 bg-white dark:divide-neutral-800 dark:bg-neutral-900">
                {failures.map((f) => (
                  <tr key={f.id}>
                    <td className="whitespace-nowrap px-3 py-2 text-slate-700 dark:text-neutral-300">
                      {new Date(f.requestedAt).toLocaleString()}
                    </td>
                    <td className="px-3 py-2 text-red-600 dark:text-red-400">
                      {f.errorMessage ?? '—'}
                    </td>
                    <td className="whitespace-nowrap px-3 py-2 text-slate-700 dark:text-neutral-300">
                      {f.webhookResponseCode ?? '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </section>
  )
}
