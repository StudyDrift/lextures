import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  AI_PROVIDER_LABELS,
  buildSecretUpdate,
  buildCredentialSecretUpdate,
  draftFromCredential,
  emptyCredentialDraft,
  fetchOrgAISettings,
  isKnownAIProvider,
  providerLabel,
  putOrgAISettings,
  settingsObjectFromDraft,
  testOrgAIConnection,
  type AIProviderCredential,
  type OrgAISettings,
  type ProviderCredentialDraft,
} from '../../lib/ai-providers'
import { PLATFORM_SECRET_PLACEHOLDER } from '../../lib/platform-settings'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { ProviderCredentialForm } from './provider-credential-form'

export function AiProviderSettingsPanel() {
  const { t } = useTranslation('common')
  const [data, setData] = useState<OrgAISettings | null>(null)
  const [provider, setProvider] = useState('openrouter')
  const [modelAlias, setModelAlias] = useState('claude-3-5-sonnet')
  const [fallbackProvider, setFallbackProvider] = useState('')
  const [byokKey, setByokKey] = useState('')
  const [byokBaseline, setByokBaseline] = useState('')
  const [credentials, setCredentials] = useState<AIProviderCredential[]>([])
  const [drafts, setDrafts] = useState<Record<string, ProviderCredentialDraft>>({})
  const [loading, setLoading] = useState(true)
  const [disabledByFlag, setDisabledByFlag] = useState(false)
  const [forbidden, setForbidden] = useState(false)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [credSaving, setCredSaving] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setDisabledByFlag(false)
    setForbidden(false)
    try {
      const json = await fetchOrgAISettings()
      setData(json)
      setProvider(json.provider ?? 'openrouter')
      setModelAlias(json.modelAlias ?? 'claude-3-5-sonnet')
      setFallbackProvider(json.fallbackProvider ?? '')
      const placeholder = json.byokConfigured ? PLATFORM_SECRET_PLACEHOLDER : ''
      setByokKey(placeholder)
      setByokBaseline(placeholder)
      const creds = json.credentials ?? []
      setCredentials(creds)
      const next: Record<string, ProviderCredentialDraft> = {}
      for (const c of creds) {
        next[c.provider] = draftFromCredential(c)
      }
      // Ensure selected provider has a draft even if no credential row yet.
      const providers = json.providers ?? Object.keys(AI_PROVIDER_LABELS)
      for (const p of providers) {
        if (!next[p]) {
          next[p] = emptyCredentialDraft()
        }
      }
      setDrafts(next)
    } catch (e) {
      const code = e instanceof Error ? (e as Error & { code?: string }).code : undefined
      if (code === 'AI_PROVIDER_ABSTRACTION_DISABLED') {
        setDisabledByFlag(true)
        return
      }
      if (code === 'FORBIDDEN') {
        setForbidden(true)
        return
      }
      /* not org admin or feature disabled */
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const providers = useMemo(() => {
    if (data?.providers?.length) return data.providers
    return Object.keys(AI_PROVIDER_LABELS)
  }, [data?.providers])

  const aliases = data?.modelAliases ?? ['claude-3-5-sonnet', 'gpt-4o', 'gemini-1.5-pro']

  function updateDraft(providerId: string, patch: Partial<ProviderCredentialDraft>) {
    setDrafts((prev) => {
      const base = prev[providerId] ?? emptyCredentialDraft()
      return { ...prev, [providerId]: { ...base, ...patch } }
    })
  }

  const save = useCallback(async () => {
    setSaving(true)
    try {
      const payload: Parameters<typeof putOrgAISettings>[0] = {
        provider,
        modelAlias,
        fallbackProvider: fallbackProvider.trim() || null,
      }
      const byokUpdate = buildSecretUpdate(
        { apiKey: byokKey, apiKeyBaseline: byokBaseline },
        data?.byokConfigured === true,
      )
      if (byokUpdate.apiKey) payload.byokApiKey = byokUpdate.apiKey
      if (byokUpdate.clearApiKey) payload.clearByokApiKey = true

      // Include selected provider credential (settings + key) in the multi-cred array.
      const draft = drafts[provider]
      if (draft) {
        const cred = credentials.find((c) => c.provider === provider)
        const secret = buildCredentialSecretUpdate(
          draft,
          cred ?? { provider, enabled: true, apiKeyConfigured: false },
        )
        payload.credentials = [
          {
            provider,
            enabled: draft.enabled,
            settings: settingsObjectFromDraft(provider, draft.settings),
            apiKey: secret.apiKey,
            clearApiKey: secret.clearApiKey,
            awsAccessKeyId: secret.awsAccessKeyId,
            clearAwsAccessKeyId: secret.clearAwsAccessKeyId,
            awsSecretAccessKey: secret.awsSecretAccessKey,
            clearAwsSecretAccessKey: secret.clearAwsSecretAccessKey,
            serviceAccountJson: secret.serviceAccountJson,
            clearServiceAccountJson: secret.clearServiceAccountJson,
          },
        ]
        // Prefer per-provider credential key over legacy single BYOK when present.
        if (secret.apiKey) {
          delete payload.byokApiKey
          delete payload.clearByokApiKey
        }
      }

      const json = await putOrgAISettings(payload)
      setData((prev) => ({ ...prev, ...json }))
      if (json.byokConfigured) {
        setByokKey(PLATFORM_SECRET_PLACEHOLDER)
        setByokBaseline(PLATFORM_SECRET_PLACEHOLDER)
      } else {
        setByokKey('')
        setByokBaseline('')
      }
      if (json.credentials) {
        setCredentials(json.credentials)
        const next: Record<string, ProviderCredentialDraft> = { ...drafts }
        for (const c of json.credentials) {
          next[c.provider] = draftFromCredential(c)
        }
        setDrafts(next)
      }
      toastSaveOk(t('settings.ai.toasts.orgSaved'))
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('settings.ai.errors.saveOrg'))
    } finally {
      setSaving(false)
    }
  }, [byokBaseline, byokKey, credentials, data?.byokConfigured, drafts, fallbackProvider, modelAlias, provider, t])

  const saveCredential = useCallback(
    async (providerId: string) => {
      const draft = drafts[providerId]
      if (!draft) return
      setCredSaving(providerId)
      try {
        const cred = credentials.find((c) => c.provider === providerId)
        const secret = buildCredentialSecretUpdate(
          draft,
          cred ?? { provider: providerId, enabled: true, apiKeyConfigured: false },
        )
        const json = await putOrgAISettings({
          provider,
          modelAlias,
          fallbackProvider: fallbackProvider.trim() || null,
          credentials: [
            {
              provider: providerId,
              enabled: draft.enabled,
              settings: settingsObjectFromDraft(providerId, draft.settings),
              apiKey: secret.apiKey,
              clearApiKey: secret.clearApiKey,
              awsAccessKeyId: secret.awsAccessKeyId,
              clearAwsAccessKeyId: secret.clearAwsAccessKeyId,
              awsSecretAccessKey: secret.awsSecretAccessKey,
              clearAwsSecretAccessKey: secret.clearAwsSecretAccessKey,
              serviceAccountJson: secret.serviceAccountJson,
              clearServiceAccountJson: secret.clearServiceAccountJson,
            },
          ],
        })
        setData((prev) => ({ ...prev, ...json }))
        if (json.credentials) {
          setCredentials(json.credentials)
          const next = { ...drafts }
          for (const c of json.credentials) {
            next[c.provider] = draftFromCredential(c)
          }
          setDrafts(next)
        }
        toastSaveOk(t('settings.ai.toasts.providerSaved', { provider: providerLabel(providerId) }))
      } catch (e) {
        toastMutationError(e instanceof Error ? e.message : t('settings.ai.errors.saveProvider'))
      } finally {
        setCredSaving(null)
      }
    },
    [credentials, drafts, fallbackProvider, modelAlias, provider, t],
  )

  const testConnection = useCallback(async () => {
    setTesting(true)
    try {
      const json = await testOrgAIConnection()
      const name = json.provider ?? provider
      const label = isKnownAIProvider(name) ? AI_PROVIDER_LABELS[name] : name
      toastSaveOk(
        t('settings.ai.toasts.testOk', {
          provider: label,
          authMode: json.authMode ?? 'api_key',
          latency: json.latencyMs ?? json.totalLatencyMs ?? '?',
          preview: json.responsePreview ?? 'OK',
        }),
      )
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('settings.ai.errors.testFailed'))
    } finally {
      setTesting(false)
    }
  }, [provider, t])

  if (loading) {
    return (
      <section className="mt-8">
        <p className="text-sm text-slate-500 dark:text-neutral-400">{t('common.loading')}</p>
      </section>
    )
  }

  if (forbidden) {
    return null
  }

  if (disabledByFlag) {
    return (
      <section className="mt-8 rounded-xl border border-amber-200 bg-amber-50 p-5 dark:border-amber-900/50 dark:bg-amber-950/40">
        <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-100">
          {t('settings.ai.org.title')}
        </h3>
        <p className="mt-2 text-sm text-amber-900 dark:text-amber-100" role="status">
          {t('settings.ai.banner.abstractionDisabled')}
        </p>
      </section>
    )
  }

  if (!data) {
    return null
  }

  const selectedDraft = drafts[provider]
  const selectedCred =
    credentials.find((c) => c.provider === provider) ??
    ({
      provider,
      enabled: true,
      apiKeyConfigured: false,
      settings: {},
    } satisfies AIProviderCredential)

  return (
    <section className="mt-8 rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-600 dark:bg-neutral-900">
      <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">{t('settings.ai.org.title')}</h3>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{t('settings.ai.org.description')}</p>

      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <label className="block text-sm">
          <span className="font-medium text-slate-700 dark:text-neutral-300">{t('settings.ai.org.provider')}</span>
          <select
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            value={provider}
            onChange={(e) => setProvider(e.target.value)}
          >
            {providers.map((p) => (
              <option key={p} value={p}>
                {providerLabel(p)}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="font-medium text-slate-700 dark:text-neutral-300">{t('settings.ai.org.modelAlias')}</span>
          <select
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            value={modelAlias}
            onChange={(e) => setModelAlias(e.target.value)}
          >
            {aliases.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm sm:col-span-2">
          <span className="font-medium text-slate-700 dark:text-neutral-300">
            {t('settings.ai.org.fallbackProvider')}
          </span>
          <select
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            value={fallbackProvider}
            onChange={(e) => setFallbackProvider(e.target.value)}
          >
            <option value="">{t('settings.ai.org.fallbackNone')}</option>
            {providers.map((p) => (
              <option key={p} value={p}>
                {providerLabel(p)}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm sm:col-span-2">
          <span className="font-medium text-slate-700 dark:text-neutral-300">{t('settings.ai.org.legacyByok')}</span>
          <input
            type="password"
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            value={byokKey}
            placeholder={PLATFORM_SECRET_PLACEHOLDER}
            onChange={(e) => setByokKey(e.target.value)}
            autoComplete="off"
          />
          <span className="mt-1 block text-xs text-slate-500 dark:text-neutral-400">
            {t('settings.ai.org.legacyByokHelp')}
          </span>
          {data.byokConfigured ? (
            <span className="mt-1 block text-xs text-emerald-600 dark:text-emerald-400">
              {t('settings.ai.badge.configured')}
            </span>
          ) : null}
        </label>
      </div>

      {selectedDraft ? (
        <div className="mt-6">
          <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {t('settings.ai.org.providerCredentials', { provider: providerLabel(provider) })}
          </h4>
          <ul className="mt-2 divide-y divide-slate-200 rounded-xl border border-slate-200 dark:divide-neutral-700 dark:border-neutral-700">
            <ProviderCredentialForm
              provider={provider}
              credential={selectedCred}
              draft={selectedDraft}
              active
              saving={credSaving === provider}
              onChange={(patch) => updateDraft(provider, patch)}
              onSave={() => void saveCredential(provider)}
            />
          </ul>
        </div>
      ) : null}

      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
          disabled={saving}
          onClick={() => void save()}
        >
          {saving ? t('settings.ai.actions.saving') : t('settings.ai.actions.save')}
        </button>
        <button
          type="button"
          className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 disabled:opacity-50 dark:border-neutral-600 dark:text-neutral-200"
          disabled={testing}
          onClick={() => void testConnection()}
        >
          {testing ? t('settings.ai.actions.testing') : t('settings.ai.actions.testConnection')}
        </button>
      </div>
    </section>
  )
}
