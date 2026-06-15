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
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Advising</h2>
        <p className="mt-4 text-sm text-slate-500">Loading…</p>
      </section>
    )
  }

  if (!ffAdvisingIntegration) {
    return (
      <section>
        <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Advising</h2>
        <p className="mt-4 text-sm text-slate-600 dark:text-neutral-400">
          Advising integration is not enabled. Turn on{' '}
          <span className="font-medium">Advising integration</span> under Global platform settings.
        </p>
      </section>
    )
  }

  return (
    <section>
      <h2 className="text-base font-semibold text-slate-900 dark:text-neutral-100">Advising</h2>
      <p className="mt-1 text-sm text-slate-600 dark:text-neutral-400">
        Configure the advising appointment link and degree-audit provider for student dashboards.
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
            <label htmlFor="advising-appointment-url" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Advising appointment URL
            </label>
            <input
              id="advising-appointment-url"
              type="url"
              value={appointmentUrl}
              onChange={(e) => setAppointmentUrl(e.target.value)}
              placeholder="https://navigate.example.edu/appointments"
              className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
            />
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              Shown as &quot;Schedule Advising Appointment&quot; on the student dashboard (EAB Navigate, Calendly, etc.).
            </p>
          </div>
          <div>
            <label htmlFor="advising-provider" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
              Degree audit provider
            </label>
            <select
              id="advising-provider"
              value={provider}
              onChange={(e) => setProvider(e.target.value as typeof provider)}
              className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
            >
              <option value="none">None</option>
              <option value="degreeworks">DegreeWorks</option>
              <option value="stellic">Stellic</option>
            </select>
          </div>
          {provider !== 'none' && (
            <>
              <div>
                <label htmlFor="advising-base-url" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                  Degree audit API base URL
                </label>
                <input
                  id="advising-base-url"
                  type="url"
                  value={baseUrl}
                  onChange={(e) => setBaseUrl(e.target.value)}
                  placeholder="https://degreeworks.example.edu/api"
                  className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
                />
              </div>
              <div>
                <label htmlFor="advising-creds" className="block text-sm font-medium text-slate-700 dark:text-neutral-300">
                  API credentials reference
                </label>
                <input
                  id="advising-creds"
                  type="text"
                  value={credentialsRef}
                  onChange={(e) => setCredentialsRef(e.target.value)}
                  placeholder="cloud-provider credential id"
                  className="mt-1 block w-full rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-900 focus:outline-none focus:ring-2 focus:ring-indigo-500 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-50"
                />
              </div>
              <label className="flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-300">
                <input
                  type="checkbox"
                  checked={atRiskBanner}
                  onChange={(e) => setAtRiskBanner(e.target.checked)}
                  className="rounded border-slate-300"
                />
                Show at-risk banner on student dashboard when flagged by degree audit
              </label>
            </>
          )}
          <button
            type="submit"
            disabled={saving}
            className="rounded-md bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-500 disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save configuration'}
          </button>
          {config && (
            <p className="text-xs text-slate-500 dark:text-neutral-400">
              Current provider: {config.degreeAuditProvider || 'none'}
            </p>
          )}
        </form>
      )}
    </section>
  )
}
