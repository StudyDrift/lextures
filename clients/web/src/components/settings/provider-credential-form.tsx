import { useTranslation } from 'react-i18next'
import { PLATFORM_SECRET_PLACEHOLDER } from '../../lib/platform-settings'
import {
  isKnownAIProvider,
  providerLabel,
  showsApiKeyField,
  showsAwsAccessKeyFields,
  showsServiceAccountField,
  visibleSettingFields,
  type AIProviderCredential,
  type ProviderCredentialDraft,
} from '../../lib/ai-providers'

type Props = {
  provider: string
  credential?: AIProviderCredential
  draft: ProviderCredentialDraft
  active?: boolean
  disabled?: boolean
  showEnabledToggle?: boolean
  onChange: (patch: Partial<ProviderCredentialDraft>) => void
  onSave: () => void
  onClear?: () => void
  saving?: boolean
  clearing?: boolean
}

const inputClass =
  'mt-1 w-full rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm text-slate-900 outline-none focus:border-indigo-400 focus:ring-2 focus:ring-indigo-500/20 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-100'

export function ProviderCredentialForm({
  provider,
  credential,
  draft,
  active = false,
  disabled = false,
  showEnabledToggle = true,
  onChange,
  onSave,
  onClear,
  saving = false,
  clearing = false,
}: Props) {
  const { t } = useTranslation('common')
  const label = providerLabel(provider)
  const configured =
    credential?.apiKeyConfigured === true ||
    credential?.awsAccessKeyIdConfigured === true ||
    credential?.serviceAccountJsonConfigured === true ||
    credential?.authMode === 'iam_role' ||
    credential?.authMode === 'adc'
  const fields = visibleSettingFields(provider, draft.settings)
  const busy = disabled || saving || clearing
  const showApiKey = showsApiKeyField(provider, draft.settings)
  const showAwsKeys = showsAwsAccessKeyFields(provider, draft.settings)
  const showSA = showsServiceAccountField(provider, draft.settings)

  function updateSetting(key: string, value: string) {
    onChange({ settings: { ...draft.settings, [key]: value } })
  }

  function onServiceAccountFile(file: File | null) {
    if (!file) return
    const reader = new FileReader()
    reader.onload = () => {
      const text = typeof reader.result === 'string' ? reader.result : ''
      onChange({ serviceAccountJson: text })
    }
    reader.readAsText(file)
  }

  return (
    <li className="px-4 py-4">
      <div className="flex flex-wrap items-start gap-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <p className="text-sm font-medium text-slate-900 dark:text-neutral-100">{label}</p>
            {configured ? (
              <span className="rounded-full bg-emerald-50 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300">
                {t('settings.ai.badge.configured')}
              </span>
            ) : (
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-600 dark:bg-neutral-800 dark:text-neutral-300">
                {t('settings.ai.badge.notConfigured')}
              </span>
            )}
            {draft.enabled ? (
              <span className="rounded-full bg-indigo-50 px-2 py-0.5 text-xs font-medium text-indigo-700 dark:bg-indigo-950/40 dark:text-indigo-300">
                {t('settings.ai.badge.enabled')}
              </span>
            ) : (
              <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-500 dark:bg-neutral-800 dark:text-neutral-400">
                {t('settings.ai.badge.disabled')}
              </span>
            )}
            {active ? (
              <span className="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-800 dark:bg-amber-950/40 dark:text-amber-200">
                {t('settings.ai.badge.default')}
              </span>
            ) : null}
          </div>
          {isKnownAIProvider(provider) && (provider === 'bedrock' || provider === 'vertex') ? (
            <p className="mt-1 text-xs text-slate-500 dark:text-neutral-400">
              {t('settings.ai.help.cloudAuth', {
                docs:
                  provider === 'bedrock'
                    ? 'docs/runbooks/bedrock-iam-setup.md'
                    : 'docs/runbooks/vertex-adc-setup.md',
              })}
            </p>
          ) : null}
        </div>
        {showEnabledToggle ? (
          <button
            type="button"
            role="switch"
            aria-checked={draft.enabled}
            aria-label={
              draft.enabled
                ? t('settings.ai.actions.disableProvider', { provider: label })
                : t('settings.ai.actions.enableProvider', { provider: label })
            }
            disabled={busy}
            onClick={() => onChange({ enabled: !draft.enabled })}
            className={`relative mt-0.5 inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600 disabled:cursor-not-allowed disabled:opacity-60 ${
              draft.enabled ? 'bg-indigo-600' : 'bg-slate-200 dark:bg-neutral-600'
            }`}
          >
            <span
              className={`pointer-events-none block h-5 w-5 rounded-full bg-white shadow ring-0 transition-transform ${
                draft.enabled ? 'translate-x-5' : 'translate-x-0'
              }`}
            />
          </button>
        ) : null}
      </div>

      <div className="mt-4 grid gap-3 sm:grid-cols-2">
        {fields.map((field) => {
          if (field.type === 'select' && field.options) {
            return (
              <label key={field.key} className="block text-xs sm:col-span-2">
                <span className="font-medium text-slate-600 dark:text-neutral-300">{t(field.labelKey)}</span>
                <select
                  value={draft.settings[field.key] ?? field.options[0]?.value ?? ''}
                  disabled={busy}
                  onChange={(e) => updateSetting(field.key, e.target.value)}
                  className={inputClass}
                >
                  {field.options.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {t(opt.labelKey)}
                    </option>
                  ))}
                </select>
              </label>
            )
          }
          if (field.type === 'textarea') {
            return (
              <label key={field.key} className="block text-xs sm:col-span-2">
                <span className="font-medium text-slate-600 dark:text-neutral-300">
                  {t(field.labelKey)}
                  {field.required ? ' *' : ''}
                </span>
                <textarea
                  rows={3}
                  value={draft.settings[field.key] ?? ''}
                  disabled={busy}
                  placeholder={field.placeholder}
                  onChange={(e) => updateSetting(field.key, e.target.value)}
                  className={`${inputClass} font-mono`}
                />
                {field.helpKey ? (
                  <span className="mt-1 block text-xs text-slate-500 dark:text-neutral-400">{t(field.helpKey)}</span>
                ) : null}
              </label>
            )
          }
          return (
            <label key={field.key} className="block text-xs">
              <span className="font-medium text-slate-600 dark:text-neutral-300">
                {t(field.labelKey)}
                {field.required ? ' *' : ''}
              </span>
              <input
                type={field.type === 'password' ? 'password' : 'text'}
                autoComplete="off"
                value={draft.settings[field.key] ?? ''}
                disabled={busy}
                placeholder={field.placeholder}
                required={field.required}
                onChange={(e) => updateSetting(field.key, e.target.value)}
                className={inputClass}
              />
            </label>
          )
        })}

        {showApiKey ? (
          <label className="block text-xs sm:col-span-2">
            <span className="font-medium text-slate-600 dark:text-neutral-300">{t('settings.ai.fields.apiKey')}</span>
            <input
              type="password"
              autoComplete="off"
              value={draft.apiKey}
              disabled={busy}
              placeholder={PLATFORM_SECRET_PLACEHOLDER}
              onChange={(e) => onChange({ apiKey: e.target.value })}
              className={`${inputClass} font-mono`}
            />
            <span className="mt-1 block text-xs text-slate-500 dark:text-neutral-400">
              {t('settings.ai.fields.apiKeyHelp')}
            </span>
          </label>
        ) : null}

        {showAwsKeys ? (
          <>
            <label className="block text-xs sm:col-span-2">
              <span className="font-medium text-slate-600 dark:text-neutral-300">
                {t('settings.ai.fields.awsAccessKeyId')}
              </span>
              <input
                type="password"
                autoComplete="off"
                value={draft.awsAccessKeyId}
                disabled={busy}
                placeholder={PLATFORM_SECRET_PLACEHOLDER}
                onChange={(e) => onChange({ awsAccessKeyId: e.target.value })}
                className={`${inputClass} font-mono`}
              />
            </label>
            <label className="block text-xs sm:col-span-2">
              <span className="font-medium text-slate-600 dark:text-neutral-300">
                {t('settings.ai.fields.awsSecretAccessKey')}
              </span>
              <input
                type="password"
                autoComplete="off"
                value={draft.awsSecretAccessKey}
                disabled={busy}
                placeholder={PLATFORM_SECRET_PLACEHOLDER}
                onChange={(e) => onChange({ awsSecretAccessKey: e.target.value })}
                className={`${inputClass} font-mono`}
              />
            </label>
          </>
        ) : null}

        {showSA ? (
          <label className="block text-xs sm:col-span-2">
            <span className="font-medium text-slate-600 dark:text-neutral-300">
              {t('settings.ai.fields.serviceAccountJson')}
            </span>
            <textarea
              rows={4}
              value={draft.serviceAccountJson}
              disabled={busy}
              placeholder={PLATFORM_SECRET_PLACEHOLDER}
              onChange={(e) => onChange({ serviceAccountJson: e.target.value })}
              className={`${inputClass} font-mono`}
            />
            <input
              type="file"
              accept="application/json,.json"
              disabled={busy}
              className="mt-2 block w-full text-xs text-slate-600 dark:text-neutral-300"
              onChange={(e) => onServiceAccountFile(e.target.files?.[0] ?? null)}
            />
            <span className="mt-1 block text-xs text-slate-500 dark:text-neutral-400">
              {t('settings.ai.fields.serviceAccountJsonHelp')}
            </span>
          </label>
        ) : null}
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          disabled={busy}
          onClick={onSave}
          className="rounded-lg bg-slate-900 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
        >
          {saving ? t('settings.ai.actions.saving') : t('settings.ai.actions.save')}
        </button>
        {onClear && configured ? (
          <button
            type="button"
            disabled={busy}
            onClick={onClear}
            className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 disabled:opacity-50 dark:border-neutral-600 dark:text-neutral-200"
          >
            {clearing ? t('settings.ai.actions.clearing') : t('settings.ai.actions.clearKey')}
          </button>
        ) : null}
      </div>
    </li>
  )
}
