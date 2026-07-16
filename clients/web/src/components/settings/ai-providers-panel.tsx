import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import {
  anyProviderConfigured,
  buildCredentialSecretUpdate,
  deletePlatformAIProvider,
  draftFromCredential,
  emptyCredentialDraft,
  fetchPlatformAIProviders,
  putPlatformAIProvider,
  putPlatformAIProviderPolicy,
  settingsObjectFromDraft,
  type AIProviderCredential,
  type ProviderCredentialDraft,
} from '../../lib/ai-providers'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'
import { useConfirm } from '../use-confirm'
import { ProviderCredentialForm } from './provider-credential-form'

type Props = {
  /** Provider currently chosen as platform active (first configured). */
  activeProvider?: string
  onCredentialsChanged?: () => void
}

export function AiProvidersPanel({ activeProvider, onCredentialsChanged }: Props) {
  const { t } = useTranslation('common')
  const { confirm, ConfirmDialogHost } = useConfirm()
  const [credentials, setCredentials] = useState<AIProviderCredential[]>([])
  const [drafts, setDrafts] = useState<Record<string, ProviderCredentialDraft>>({})
  const [tenantByokAllowed, setTenantByokAllowed] = useState(true)
  const [tenantAllowedProviders, setTenantAllowedProviders] = useState<string[]>([])
  const [allProviders, setAllProviders] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [disabledByFlag, setDisabledByFlag] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState<string | null>(null)
  const [clearing, setClearing] = useState<string | null>(null)
  const [policySaving, setPolicySaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    setDisabledByFlag(false)
    try {
      const data = await fetchPlatformAIProviders()
      setCredentials(data.credentials ?? [])
      setAllProviders(data.providers ?? [])
      setTenantByokAllowed(data.tenantByokAllowed !== false)
      setTenantAllowedProviders(data.tenantAllowedProviders ?? [])
      const next: Record<string, ProviderCredentialDraft> = {}
      for (const c of data.credentials ?? []) {
        next[c.provider] = draftFromCredential(c)
      }
      setDrafts(next)
    } catch (e) {
      if (e instanceof Error && (e as Error & { code?: string }).code === 'AI_PROVIDER_ABSTRACTION_DISABLED') {
        setDisabledByFlag(true)
        return
      }
      setError(e instanceof Error ? e.message : t('settings.ai.errors.loadProviders'))
    } finally {
      setLoading(false)
    }
  }, [t])

  useEffect(() => {
    void load()
  }, [load])

  const noneConfigured = useMemo(() => !anyProviderConfigured(credentials), [credentials])

  function updateDraft(provider: string, patch: Partial<ProviderCredentialDraft>) {
    setDrafts((prev) => {
      const base = prev[provider] ?? emptyCredentialDraft()
      return { ...prev, [provider]: { ...base, ...patch } }
    })
  }

  async function handleSave(provider: string) {
    const draft = drafts[provider]
    const cred = credentials.find((c) => c.provider === provider)
    if (!draft) return
    setSaving(provider)
    try {
      const secret = buildCredentialSecretUpdate(draft, cred ?? { provider, enabled: true, apiKeyConfigured: false })
      const saved = await putPlatformAIProvider(provider, {
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
      })
      setCredentials((prev) => {
        const others = prev.filter((c) => c.provider !== provider)
        return [...others, saved].sort((a, b) => a.provider.localeCompare(b.provider))
      })
      setDrafts((prev) => ({ ...prev, [provider]: draftFromCredential(saved) }))
      toastSaveOk(t('settings.ai.toasts.providerSaved', { provider }))
      onCredentialsChanged?.()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('settings.ai.errors.saveProvider'))
    } finally {
      setSaving(null)
    }
  }

  async function handleClear(provider: string) {
    if (
      !(await confirm({
        title: t('settings.ai.confirm.clearKey'),
        variant: 'danger',
        confirmLabel: t('settings.ai.actions.clearKey'),
      }))
    ) {
      return
    }
    setClearing(provider)
    try {
      await deletePlatformAIProvider(provider)
      setCredentials((prev) =>
        prev.map((c) =>
          c.provider === provider
            ? { ...c, apiKeyConfigured: false, apiKey: '', settings: c.settings ?? {} }
            : c,
        ),
      )
      setDrafts((prev) => ({
        ...prev,
        [provider]: {
          ...emptyCredentialDraft(),
          enabled: prev[provider]?.enabled ?? true,
          settings: prev[provider]?.settings ?? {},
        },
      }))
      toastSaveOk(t('settings.ai.toasts.providerCleared', { provider }))
      onCredentialsChanged?.()
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('settings.ai.errors.clearProvider'))
    } finally {
      setClearing(null)
    }
  }

  async function handleSavePolicy() {
    setPolicySaving(true)
    try {
      const next = await putPlatformAIProviderPolicy({
        tenantByokAllowed,
        tenantAllowedProviders: tenantAllowedProviders.length > 0 ? tenantAllowedProviders : [],
      })
      setTenantByokAllowed(next.tenantByokAllowed)
      setTenantAllowedProviders(next.tenantAllowedProviders ?? [])
      toastSaveOk(t('settings.ai.toasts.policySaved'))
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : t('settings.ai.errors.savePolicy'))
    } finally {
      setPolicySaving(false)
    }
  }

  function toggleAllowedProvider(provider: string) {
    setTenantAllowedProviders((prev) =>
      prev.includes(provider) ? prev.filter((p) => p !== provider) : [...prev, provider],
    )
  }

  if (disabledByFlag) {
    return (
      <section className="mt-6" aria-labelledby="ai-providers-heading">
        <h3 id="ai-providers-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
          {t('settings.ai.providers.title')}
        </h3>
        <p
          className="mt-3 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100"
          role="status"
        >
          {t('settings.ai.banner.abstractionDisabled')}
        </p>
      </section>
    )
  }

  return (
    <section className="mt-6" aria-labelledby="ai-providers-heading">
      <h3 id="ai-providers-heading" className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
        {t('settings.ai.providers.title')}
      </h3>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">{t('settings.ai.providers.description')}</p>

      {loading && <p className="mt-4 text-sm text-slate-500 dark:text-neutral-400">{t('common.loading')}</p>}
      {error && (
        <p className="mt-4 text-sm text-rose-700 dark:text-rose-300" role="alert">
          {error}
        </p>
      )}

      {!loading && !error && noneConfigured && (
        <div
          className="mt-4 rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/50 dark:bg-amber-950/40 dark:text-amber-100"
          role="status"
        >
          <p>{t('settings.ai.empty.noProviders')}</p>
          <p className="mt-2">
            <a
              href="https://github.com/lextures/lextures/blob/main/docs/ai-providers-byok.md"
              className="font-medium text-indigo-700 underline hover:text-indigo-600 dark:text-indigo-300"
              target="_blank"
              rel="noreferrer"
            >
              {t('settings.ai.empty.docsLink')}
            </a>
          </p>
        </div>
      )}

      {!loading && !error && (
        <>
          <ul className="mt-4 divide-y divide-slate-200 rounded-xl border border-slate-200 dark:divide-neutral-700 dark:border-neutral-700">
            {credentials.map((cred) => {
              const draft = drafts[cred.provider]
              if (!draft) return null
              return (
                <ProviderCredentialForm
                  key={cred.provider}
                  provider={cred.provider}
                  credential={cred}
                  draft={draft}
                  active={activeProvider === cred.provider}
                  saving={saving === cred.provider}
                  clearing={clearing === cred.provider}
                  onChange={(patch) => updateDraft(cred.provider, patch)}
                  onSave={() => void handleSave(cred.provider)}
                  onClear={() => void handleClear(cred.provider)}
                />
              )
            })}
          </ul>

          <div className="mt-6 rounded-xl border border-slate-200 bg-white p-4 dark:border-neutral-700 dark:bg-neutral-900">
            <h4 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">
              {t('settings.ai.policy.title')}
            </h4>
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">{t('settings.ai.policy.description')}</p>
            <label className="mt-3 flex items-center gap-2 text-sm text-slate-700 dark:text-neutral-200">
              <input
                type="checkbox"
                checked={tenantByokAllowed}
                disabled={policySaving}
                onChange={(e) => setTenantByokAllowed(e.target.checked)}
                className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
              />
              {t('settings.ai.policy.allowByok')}
            </label>
            {tenantByokAllowed && allProviders.length > 0 && (
              <fieldset className="mt-3">
                <legend className="text-xs font-medium text-slate-600 dark:text-neutral-300">
                  {t('settings.ai.policy.allowedProviders')}
                </legend>
                <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
                  {t('settings.ai.policy.allowedProvidersHelp')}
                </p>
                <div className="mt-2 flex flex-wrap gap-3">
                  {allProviders.map((p) => (
                    <label key={p} className="flex items-center gap-1.5 text-sm text-slate-700 dark:text-neutral-200">
                      <input
                        type="checkbox"
                        checked={
                          tenantAllowedProviders.length === 0 || tenantAllowedProviders.includes(p)
                        }
                        disabled={policySaving}
                        onChange={() => {
                          // Empty list means "all allowed". First toggle from all → explicit list without this one.
                          if (tenantAllowedProviders.length === 0) {
                            setTenantAllowedProviders(allProviders.filter((x) => x !== p))
                            return
                          }
                          toggleAllowedProvider(p)
                        }}
                        className="rounded border-slate-300 text-indigo-600 focus:ring-indigo-500"
                      />
                      {p}
                    </label>
                  ))}
                </div>
              </fieldset>
            )}
            <button
              type="button"
              disabled={policySaving}
              onClick={() => void handleSavePolicy()}
              className="mt-4 rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 disabled:opacity-50 dark:border-neutral-600 dark:text-neutral-200"
            >
              {policySaving ? t('settings.ai.actions.saving') : t('settings.ai.policy.save')}
            </button>
          </div>
        </>
      )}
      {ConfirmDialogHost}
    </section>
  )
}
