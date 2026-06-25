import { type FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { authorizedFetch } from '../../lib/api'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { readApiErrorMessage } from '../../lib/errors'
import { PLATFORM_SECRET_PLACEHOLDER } from '../../lib/platform-settings'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { FeatureToggleRow } from './feature-toggle-row'
import { PLATFORM_FEATURE_DEFINITIONS, type PlatformFeatureDefinition } from './platform-feature-definitions'
import type { FieldSource, PlatformSettingsPayload } from './platform-settings-types'

export type { PlatformSettingsPayload } from './platform-settings-types'

function normalizePlatformPayload(data: PlatformSettingsPayload) {
  if (typeof data.smtpPort !== 'number' || Number.isNaN(data.smtpPort)) {
    data.smtpPort = 587
  }
  data.smtpHost ??= ''
  data.smtpFrom ??= ''
  data.smtpUser ??= ''
  data.smtpPassword ??= ''
  data.sources = { ...emptyForm().sources, ...data.sources }
}

function emptyForm(): PlatformSettingsPayload {
  return {
    samlSsoEnabled: false,
    samlPublicBaseUrl: '',
    samlSpEntityId: '',
    samlSpX509Pem: '',
    samlSpPrivateKeyPem: '',
    annotationEnabled: false,
    feedbackMediaEnabled: false,
    blindGradingEnabled: true,
    moderatedGradingEnabled: false,
    originalityDetectionEnabled: false,
    originalityStubExternal: false,
    gradePostingPoliciesEnabled: true,
    gradebookCsvEnabled: false,
    resubmissionWorkflowEnabled: false,
    ltiEnabled: false,
    oneRosterEnabled: false,
    scimEnabled: false,
    studentProgressEnabled: false,
    selfReflectionEnabled: false,
    outcomesReportEnabled: false,
    atRiskAlertsEnabled: false,
    h5pEnabled: false,
    scormIngestionEnabled: false,
    oerLibraryEnabled: false,
    oerStub: false,
    itemAnalysisEnabled: false,
    xapiEmissionEnabled: false,
    equationEditorEnabled: false,
    readingLevelEnabled: false,
    graderAgentEnabled: false,
    graderAgentReviewInboxEnabled: false,
    graderAgentSuggestModeEnabled: false,
    graderAgentTextEntryGradingEnabled: true,
    graderAgentVisionGradingEnabled: false,
    graderAgentRunFiltersEnabled: false,
    graderAgentCostEstimateEnabled: false,
    codeExecutionEnabled: false,
    speechToTextEnabled: false,
    accommodationsEngineEnabled: false,
    ffAccommodationsEngine: false,
    ffBookstoreIntegration: false,
    ffCoCurricularTranscript: false,
    ffEportfolio: false,
    ffTranscripts: false,
    ffWebhooks: false,
    ffAdvisingIntegration: false,
    ffResearchConsent: false,
    ffAccessibilityIntake: false,
    ffCeuTracking: false,
    ffConsortiumSharing: false,
    ffStripeBilling: false,
    ffRevenueShare: false,
    ffTaxCollection: false,
    ffLearningPaths: false,
    ffConditionalRelease: false,
    ffPeerReview: false,
    ffCompletionCredentials: false,
    ffOnboardingFlow: false,
    ffWhatifGrades: false,
    ffGradeCurving: false,
    ffAiStudyBuddy: false,
    ffApiTokens: false,
    ffCalendarFeeds: true,
    ffAcademicCalendar: false,
    ffClassroomSignals: false,
    ffConferenceScheduling: false,
    ffBotSlack: false,
    ffBotTeams: false,
    ffBotDiscord: false,
    ffBroadcasts: false,
    ffCourseEvaluations: false,
    ffSisIntegration: false,
    ffGamification: false,
    ffDemographics: false,
    ffEnrollmentStateMachine: false,
    ffIncompleteGradeWorkflow: false,
    ffGradeSubmission: false,
    ffCatalogIntegration: false,
    ffLibrary: false,
    ffLibraryIntegration: false,
    ffReadingPreferences: false,
    ffUiMode: false,
    ffReadAloud: false,
    ffAltTextEnforcement: false,
    ffProctoringIntegration: false,
    ffCourseReviews: false,
    ffParentPortal: false,
    ffReportCards: false,
    ffPublicCatalog: false,
    ffSelfPacedMode: false,
    ffPublicApi: false,
    ffContentFilterIntegration: false,
    ffPlagiarismChecks: false,
    ffStudyReminders: false,
    translationMemoryEnabled: false,
    storageQuotasEnabled: false,
    avScanningEnabled: false,
    virtualClassroomEnabled: true,
    sessionManagementUiEnabled: false,
    mfaEnabled: false,
    mfaEnforcement: 'none',
    smtpHost: '',
    smtpPort: 587,
    smtpFrom: '',
    smtpUser: '',
    smtpPassword: '',
    sources: {
      samlSsoEnabled: 'environment',
      samlPublicBaseUrl: 'environment',
      samlSpEntityId: 'environment',
      samlSpX509Pem: 'environment',
      samlSpPrivateKeyPem: 'environment',
      annotationEnabled: 'environment',
      feedbackMediaEnabled: 'environment',
      blindGradingEnabled: 'environment',
      moderatedGradingEnabled: 'environment',
      originalityDetectionEnabled: 'environment',
      originalityStubExternal: 'environment',
      gradePostingPoliciesEnabled: 'environment',
      gradebookCsvEnabled: 'environment',
      resubmissionWorkflowEnabled: 'environment',
      ltiEnabled: 'environment',
      oneRosterEnabled: 'environment',
      scimEnabled: 'environment',
      mfaEnabled: 'environment',
      mfaEnforcement: 'environment',
      smtpHost: 'environment',
      smtpPort: 'environment',
      smtpFrom: 'environment',
      smtpUser: 'environment',
      smtpPassword: 'environment',
    },
  }
}

function sourceBadge(src: FieldSource) {
  if (src === 'database') {
    return (
      <span className="ms-2 rounded-md bg-indigo-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-indigo-800 dark:bg-indigo-950/80 dark:text-indigo-200">
        Database
      </span>
    )
  }
  if (src === 'default') {
    return (
      <span className="ms-2 rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-600 dark:bg-neutral-700 dark:text-neutral-300">
        Default
      </span>
    )
  }
  return (
    <span className="ms-2 rounded-md bg-slate-100 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-slate-600 dark:bg-neutral-700 dark:text-neutral-300">
      Environment
    </span>
  )
}

export function PlatformSettingsPanel() {
  const { refresh: refreshPlatformFeatures } = usePlatformFeatures()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [featureSaving, setFeatureSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [featureMessage, setFeatureMessage] = useState<string | null>(null)
  const [featureError, setFeatureError] = useState<string | null>(null)
  const [featureQuery, setFeatureQuery] = useState('')
  const [form, setForm] = useState<PlatformSettingsPayload>(() => emptyForm())
  const [baseline, setBaseline] = useState<PlatformSettingsPayload>(() => emptyForm())

  const visiblePlatformFeatures = useMemo(() => {
    const q = featureQuery.trim().toLowerCase()
    if (!q) return PLATFORM_FEATURE_DEFINITIONS
    return PLATFORM_FEATURE_DEFINITIONS.filter(
      (f) => f.label.toLowerCase().includes(q) || f.description.toLowerCase().includes(q),
    )
  }, [featureQuery])

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await authorizedFetch('/api/v1/settings/platform')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error(readApiErrorMessage(raw))
      }
      const data = raw as PlatformSettingsPayload
      normalizePlatformPayload(data)
      setForm(data)
      setBaseline(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load platform settings.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  function update<K extends keyof PlatformSettingsPayload>(key: K, value: PlatformSettingsPayload[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  const putPlatformSettings = useCallback(
    async (body: Record<string, unknown>) => {
      const res = await authorizedFetch('/api/v1/settings/platform', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) {
        throw new Error(readApiErrorMessage(raw))
      }
      const data = raw as PlatformSettingsPayload
      normalizePlatformPayload(data)
      setForm(data)
      setBaseline(data)
      await refreshPlatformFeatures()
      return data
    },
    [refreshPlatformFeatures],
  )

  const persistPlatformFeature = useCallback(
    async (key: PlatformFeatureDefinition['key'], value: boolean) => {
      if (featureSaving) return
      let previous = false
      setFeatureSaving(true)
      setFeatureMessage(null)
      setFeatureError(null)
      setForm((prev) => {
        previous = prev[key]
        return { ...prev, [key]: value }
      })
      try {
        const data = await putPlatformSettings({ [key]: value, updateMask: [key] })
        if (data[key] !== value) {
          throw new Error('The server did not persist this feature change. Reload the page and try again.')
        }
        setFeatureMessage('Saved.')
        toastSaveOk('Platform feature updated')
      } catch (e) {
        setForm((prev) => ({ ...prev, [key]: previous }))
        const msg = e instanceof Error ? e.message : 'Could not save feature.'
        setFeatureError(msg)
        toastMutationError(msg)
      } finally {
        setFeatureSaving(false)
      }
    },
    [featureSaving, putPlatformSettings],
  )

  const persistMfaEnforcement = useCallback(
    async (value: PlatformSettingsPayload['mfaEnforcement']) => {
      if (featureSaving) return
      let previous: PlatformSettingsPayload['mfaEnforcement'] = 'none'
      setFeatureSaving(true)
      setFeatureMessage(null)
      setFeatureError(null)
      setForm((prev) => {
        previous = prev.mfaEnforcement
        return { ...prev, mfaEnforcement: value }
      })
      try {
        await putPlatformSettings({ mfaEnforcement: value, updateMask: ['mfaEnforcement'] })
        setFeatureMessage('Saved.')
        toastSaveOk('MFA requirement updated')
      } catch (e) {
        setForm((prev) => ({ ...prev, mfaEnforcement: previous }))
        const msg = e instanceof Error ? e.message : 'Could not save MFA requirement.'
        setFeatureError(msg)
        toastMutationError(msg)
      } finally {
        setFeatureSaving(false)
      }
    },
    [featureSaving, putPlatformSettings],
  )

  async function onSubmit(e: FormEvent) {
    e.preventDefault()
    setSaving(true)
    setError(null)
    try {
      const mask: string[] = []
      const body: Record<string, unknown> = {}

      const maybe = (field: string, before: unknown, after: unknown, apply: () => void) => {
        if (before !== after) {
          mask.push(field)
          apply()
        }
      }

      maybe('samlSsoEnabled', baseline.samlSsoEnabled, form.samlSsoEnabled, () => {
        body.samlSsoEnabled = form.samlSsoEnabled
      })
      maybe('samlPublicBaseUrl', baseline.samlPublicBaseUrl, form.samlPublicBaseUrl, () => {
        body.samlPublicBaseUrl = form.samlPublicBaseUrl.trim()
      })
      maybe('samlSpEntityId', baseline.samlSpEntityId, form.samlSpEntityId, () => {
        body.samlSpEntityId = form.samlSpEntityId.trim()
      })
      maybe('samlSpX509Pem', baseline.samlSpX509Pem, form.samlSpX509Pem, () => {
        const v = form.samlSpX509Pem.trim()
        if (v && v !== PLATFORM_SECRET_PLACEHOLDER) {
          body.samlSpX509Pem = v
        }
      })
      maybe('samlSpPrivateKeyPem', baseline.samlSpPrivateKeyPem, form.samlSpPrivateKeyPem, () => {
        const v = form.samlSpPrivateKeyPem.trim()
        if (v && v !== PLATFORM_SECRET_PLACEHOLDER) {
          body.samlSpPrivateKeyPem = v
        }
      })

      maybe('smtpHost', baseline.smtpHost, form.smtpHost, () => {
        body.smtpHost = form.smtpHost.trim()
      })
      maybe('smtpPort', baseline.smtpPort, form.smtpPort, () => {
        body.smtpPort = form.smtpPort
      })
      maybe('smtpFrom', baseline.smtpFrom, form.smtpFrom, () => {
        body.smtpFrom = form.smtpFrom.trim()
      })
      maybe('smtpUser', baseline.smtpUser, form.smtpUser, () => {
        body.smtpUser = form.smtpUser.trim()
      })
      maybe('smtpPassword', baseline.smtpPassword, form.smtpPassword, () => {
        const v = form.smtpPassword.trim()
        if (v && v !== PLATFORM_SECRET_PLACEHOLDER) {
          body.smtpPassword = v
        }
      })
      if (
        baseline.sources.smtpPassword === 'database' &&
        form.smtpPassword.trim() === '' &&
        baseline.smtpPassword !== form.smtpPassword
      ) {
        mask.push('clearSmtpPassword')
        body.clearSmtpPassword = true
      }

      if (mask.length === 0) {
        toastSaveOk('No changes to save.')
        setSaving(false)
        return
      }

      body.updateMask = mask

      await putPlatformSettings(body)
      toastSaveOk('Platform settings saved.')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed.')
    } finally {
      setSaving(false)
    }
  }

  const chk =
    'rounded border border-slate-200 bg-white text-indigo-600 focus:ring-indigo-500 dark:border-neutral-600 dark:bg-neutral-800'

  if (loading) {
    return <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">Loading platform settings…</p>
  }

  return (
    <div>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Values stored here override the server process environment when set. Requires{' '}
        <code className="rounded bg-slate-100 px-1 font-mono text-xs dark:bg-neutral-800">global:app:rbac:manage</code>.
        Secrets are never returned in plain text after save.
      </p>

      {error && (
        <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
          {error}
        </p>
      )}

      <form className="mt-8 space-y-10" onSubmit={onSubmit}>
        <section>
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Outgoing email (SMTP)</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            Passwords are encrypted in the database using{' '}
            <code className="rounded bg-slate-100 px-1 font-mono dark:bg-neutral-900">PLATFORM_SECRETS_KEY</code> on the
            API (32 random bytes, base64). Process{' '}
            <code className="rounded bg-slate-100 px-1 font-mono dark:bg-neutral-900">SMTP_*</code> environment variables
            still apply when a field is not set here.
          </p>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                SMTP host {sourceBadge(form.sources.smtpHost)}
              </label>
              <input
                type="text"
                autoComplete="off"
                value={form.smtpHost}
                onChange={(e) => update('smtpHost', e.target.value)}
                placeholder="e.g. smtp.sendgrid.net"
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 font-mono text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                Port {sourceBadge(form.sources.smtpPort)}
              </label>
              <input
                type="number"
                min={1}
                max={65535}
                value={form.smtpPort}
                onChange={(e) => {
                  const n = parseInt(e.target.value, 10)
                  update('smtpPort', Number.isFinite(n) ? n : 587)
                }}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 font-mono text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                From address {sourceBadge(form.sources.smtpFrom)}
              </label>
              <input
                type="email"
                value={form.smtpFrom}
                onChange={(e) => update('smtpFrom', e.target.value)}
                placeholder="no-reply@school.edu"
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                Username (optional) {sourceBadge(form.sources.smtpUser)}
              </label>
              <input
                type="text"
                autoComplete="off"
                value={form.smtpUser}
                onChange={(e) => update('smtpUser', e.target.value)}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 font-mono text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                Password (optional) {sourceBadge(form.sources.smtpPassword)}
              </label>
              <input
                type="password"
                autoComplete="new-password"
                placeholder={PLATFORM_SECRET_PLACEHOLDER}
                value={form.smtpPassword}
                onChange={(e) => update('smtpPassword', e.target.value)}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 font-mono text-sm text-slate-900 outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
          </div>
        </section>

        <section>
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">SAML service provider</h3>
          <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
            Browser SSO endpoints use these SP settings (IdP metadata remains under Admin SAML).
          </p>
          <div className="mt-4 space-y-4">
            <label className="flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-neutral-200">
              <input
                type="checkbox"
                checked={form.samlSsoEnabled}
                onChange={(e) => update('samlSsoEnabled', e.target.checked)}
                className={chk}
              />
              Enable SAML SSO {sourceBadge(form.sources.samlSsoEnabled)}
            </label>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                Public base URL {sourceBadge(form.sources.samlPublicBaseUrl)}
              </label>
              <input
                type="url"
                value={form.samlPublicBaseUrl}
                onChange={(e) => update('samlPublicBaseUrl', e.target.value)}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                SP entity ID {sourceBadge(form.sources.samlSpEntityId)}
              </label>
              <input
                type="text"
                value={form.samlSpEntityId}
                onChange={(e) => update('samlSpEntityId', e.target.value)}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 text-sm dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                SP X.509 certificate (PEM) {sourceBadge(form.sources.samlSpX509Pem)}
              </label>
              <textarea
                rows={5}
                spellCheck={false}
                value={form.samlSpX509Pem}
                onChange={(e) => update('samlSpX509Pem', e.target.value)}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 font-mono text-xs dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-neutral-200">
                SP private key (PEM) {sourceBadge(form.sources.samlSpPrivateKeyPem)}
              </label>
              <textarea
                rows={4}
                spellCheck={false}
                placeholder={PLATFORM_SECRET_PLACEHOLDER}
                value={form.samlSpPrivateKeyPem}
                onChange={(e) => update('samlSpPrivateKeyPem', e.target.value)}
                className="mt-1.5 w-full rounded-xl border border-slate-200 bg-white px-3 py-2.5 font-mono text-xs dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-100"
              />
            </div>
          </div>
        </section>

        <section className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm shadow-slate-900/5 dark:border-neutral-800 dark:bg-neutral-950">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Platform features</h3>
          <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
            Turn platform-wide capabilities on or off. Changes save immediately. Database values override environment
            defaults.
          </p>

          <div className="mt-3">
            <input
              type="search"
              placeholder="Search features…"
              value={featureQuery}
              onChange={(e) => setFeatureQuery(e.target.value)}
              className="w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 placeholder:text-slate-400 focus:border-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-300 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus:border-indigo-500"
            />
          </div>

          <div className="mt-1 divide-y divide-slate-100 dark:divide-neutral-800">
            {visiblePlatformFeatures.length === 0 ? (
              <p className="py-6 text-center text-sm text-slate-400 dark:text-neutral-500">
                No features match &ldquo;{featureQuery}&rdquo;
              </p>
            ) : (
              visiblePlatformFeatures.map((feature) => {
                const enabled = form[feature.key]
                const source = feature.sourceKey ? form.sources[feature.sourceKey] : 'default'
                return (
                  <FeatureToggleRow
                    key={feature.key}
                    label={feature.label}
                    description={feature.description}
                    enabled={enabled}
                    disabled={featureSaving}
                    meta={sourceBadge(source)}
                    onToggle={() => void persistPlatformFeature(feature.key, !enabled)}
                  />
                )
              })
            )}
          </div>

          <div className="mt-4 border-t border-slate-100 pt-4 dark:border-neutral-800">
            <label className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
              MFA requirement {sourceBadge(form.sources.mfaEnforcement)}
            </label>
            <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
              Require two-factor authentication for some or all users after password sign-in.
            </p>
            <select
              value={form.mfaEnforcement}
              disabled={featureSaving}
              onChange={(e) =>
                void persistMfaEnforcement(e.target.value as PlatformSettingsPayload['mfaEnforcement'])
              }
              className="mt-3 w-full max-w-md rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 focus:border-indigo-400 focus:outline-none focus:ring-2 focus:ring-indigo-300 disabled:cursor-not-allowed disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100 dark:focus:border-indigo-500"
            >
              <option value="none">Optional (users choose)</option>
              <option value="staff">Required for teachers, TAs, and global admins</option>
              <option value="all">Required for everyone</option>
            </select>
            <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
              Set{' '}
              <code className="rounded bg-slate-100 px-1 font-mono dark:bg-neutral-900">PUBLIC_WEB_ORIGIN</code> on the
              API for passkey registration.
            </p>
          </div>

          {featureMessage ? (
            <p className="mt-4 text-sm text-emerald-700 dark:text-emerald-400" role="status">
              {featureMessage}
            </p>
          ) : null}
          {featureError ? (
            <p className="mt-4 text-sm text-rose-700 dark:text-rose-400" role="alert">
              {featureError}
            </p>
          ) : null}
        </section>

        <div className="flex flex-wrap gap-3">
          <button
            type="submit"
            disabled={saving}
            className="rounded-xl bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 disabled:cursor-not-allowed disabled:opacity-60 dark:bg-neutral-100 dark:text-neutral-950 dark:hover:bg-white"
          >
            {saving ? 'Saving…' : 'Save changes'}
          </button>
          <button
            type="button"
            onClick={() => void load()}
            disabled={saving || loading}
            className="rounded-xl border border-slate-200 bg-white px-4 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-200 dark:hover:bg-neutral-700"
          >
            Reload
          </button>
        </div>
      </form>
    </div>
  )
}
