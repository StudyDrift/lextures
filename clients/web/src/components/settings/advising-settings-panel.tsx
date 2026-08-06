import { useCallback, useEffect, useState } from 'react'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  fetchAdminAdvisingConfig,
  saveAdminAdvisingConfig,
  type AdvisingConfig,
} from '../../lib/advising-api'

export function AdvisingSettingsPanel() {
  const { ffAdvisingIntegration, loading: featuresLoading } = usePlatformFeatures()
  const [config, setConfig] = useState<AdvisingConfig | null>(null)
  const [appointmentUrl, setAppointmentUrl] = useState('')
  const [provider, setProvider] = useState<'none' | 'degreeworks' | 'stellic'>('none')
  const [baseUrl, setBaseUrl] = useState('')
  const [credentialsRef, setCredentialsRef] = useState('')
  const [atRiskBanner, setAtRiskBanner] = useState(false)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const load = useCallback(async () => {
    if (!ffAdvisingIntegration) {
      setConfig(null)
      setLoading(false)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const cfg = await fetchAdminAdvisingConfig()
      setConfig(cfg)
      setAppointmentUrl(cfg.appointmentUrl ?? '')
      setProvider(cfg.degreeAuditProvider ?? 'none')
      setBaseUrl(cfg.degreeAuditBaseUrl ?? '')
      setCredentialsRef(cfg.apiCredentialsRef ?? '')
      setAtRiskBanner(cfg.atRiskBannerEnabled ?? false)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load advising settings.')
    } finally {
      setLoading(false)
    }
  }, [ffAdvisingIntegration])

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
      const cfg = await saveAdminAdvisingConfig({
        appointmentUrl: appointmentUrl.trim(),
        degreeAuditProvider: provider,
        degreeAuditBaseUrl: baseUrl.trim(),
        apiCredentialsRef: credentialsRef.trim(),
        atRiskBannerEnabled: atRiskBanner,
      })
      setConfig(cfg)
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
        <h2 className="text-base font-semibold text-fg-default">Advising</h2>
        <p className="mt-4 text-sm text-fg-muted">Loading…</p>
      </section>
    )
  }

  if (!ffAdvisingIntegration) {
    return (
      <section>
        <h2 className="text-base font-semibold text-fg-default">Advising</h2>
        <p className="mt-4 text-sm text-fg-muted">
          Advising integration is not enabled. Turn on{' '}
          <span className="font-medium">Advising integration</span> under Global platform settings.
        </p>
      </section>
    )
  }

  return (
    <section>
      <h2 className="text-base font-semibold text-fg-default">Advising</h2>
      <p className="mt-1 text-sm text-fg-muted">
        Configure the advising appointment link and degree-audit provider for student dashboards.
      </p>

      {error && (
        <p role="alert" className="mt-4 text-sm text-danger-fg">
          {error}
        </p>
      )}
      {saved && (
        <p role="status" className="mt-4 text-sm text-emerald-600 dark:text-emerald-400">
          Settings saved.
        </p>
      )}

      {loading ? (
        <p className="mt-4 text-sm text-fg-muted">Loading configuration…</p>
      ) : (
        <form onSubmit={handleSave} className="mt-6 space-y-4">
          <div>
            <label htmlFor="advising-appointment-url" className="block text-sm font-medium text-fg-muted">
              Advising appointment URL
            </label>
            <input
              id="advising-appointment-url"
              type="url"
              value={appointmentUrl}
              onChange={(e) => setAppointmentUrl(e.target.value)}
              placeholder="https://navigate.example.edu/appointments"
              className="mt-1 block w-full rounded-md border border-border-strong bg-surface-raised px-3 py-2 text-sm text-fg-default focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-border-default dark:bg-surface-raised"
            />
            <p className="mt-1 text-xs text-fg-muted">
              Shown as &quot;Schedule Advising Appointment&quot; on the student dashboard (EAB Navigate, Calendly, etc.).
            </p>
          </div>
          <div>
            <label htmlFor="advising-provider" className="block text-sm font-medium text-fg-muted">
              Degree audit provider
            </label>
            <select
              id="advising-provider"
              value={provider}
              onChange={(e) => setProvider(e.target.value as typeof provider)}
              className="mt-1 block w-full rounded-md border border-border-strong bg-surface-raised px-3 py-2 text-sm text-fg-default focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-border-default dark:bg-surface-raised"
            >
              <option value="none">None</option>
              <option value="degreeworks">DegreeWorks</option>
              <option value="stellic">Stellic</option>
            </select>
          </div>
          {provider !== 'none' && (
            <>
              <div>
                <label htmlFor="advising-base-url" className="block text-sm font-medium text-fg-muted">
                  Degree audit API base URL
                </label>
                <input
                  id="advising-base-url"
                  type="url"
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  placeholder="https://degreeworks.example.edu/api"
                  className="mt-1 block w-full rounded-md border border-border-strong bg-surface-raised px-3 py-2 text-sm text-fg-default focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-border-default dark:bg-surface-raised"
                />
              </div>
              <div>
                <label htmlFor="advising-creds" className="block text-sm font-medium text-fg-muted">
                  API credentials reference
                </label>
                <input
                  id="advising-creds"
                  type="text"
                  value={credentialsRef}
                  onChange={(e) => setCredentialsRef(e.target.value)}
                  placeholder="cloud-provider credential id"
                  className="mt-1 block w-full rounded-md border border-border-strong bg-surface-raised px-3 py-2 text-sm text-fg-default focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-border-default dark:bg-surface-raised"
                />
              </div>
              <label className="flex items-center gap-2 text-sm text-fg-muted">
                <input
                  type="checkbox"
                  checked={atRiskBanner}
                  onChange={(e) => setAtRiskBanner(e.target.checked)}
                  className="rounded border-border-strong"
                />
                Show at-risk banner on student dashboard when flagged by degree audit
              </label>
            </>
          )}
          <button
            type="submit"
            disabled={saving}
            className="rounded-md bg-accent-solid px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save configuration'}
          </button>
          {config && (
            <p className="text-xs text-fg-muted">
              Current provider: {config.degreeAuditProvider || 'none'}
            </p>
          )}
        </form>
      )}
    </section>
  )
}
