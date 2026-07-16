import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'
import { PLATFORM_SECRET_PLACEHOLDER } from './platform-settings'

/** Supported AI provider ids (matches server `aiprovider.ListProviders`). */
export type AIProviderId =
  | 'openrouter'
  | 'anthropic'
  | 'openai'
  | 'azure_openai'
  | 'bedrock'
  | 'vertex'

export type AuthMode = 'api_key' | 'access_key' | 'iam_role' | 'service_account' | 'adc'

export const AI_PROVIDER_IDS: AIProviderId[] = [
  'openrouter',
  'anthropic',
  'openai',
  'azure_openai',
  'bedrock',
  'vertex',
]

export const AI_PROVIDER_LABELS: Record<AIProviderId, string> = {
  openrouter: 'OpenRouter',
  anthropic: 'Anthropic',
  openai: 'OpenAI',
  azure_openai: 'Azure OpenAI',
  bedrock: 'AWS Bedrock',
  vertex: 'Google Vertex AI',
}

export type ProviderSettingField = {
  key: string
  labelKey: string
  placeholder?: string
  required?: boolean
  type?: 'text' | 'password' | 'select' | 'textarea' | 'file'
  /** When set, field is shown only for these auth modes. */
  whenAuthModes?: AuthMode[]
  options?: { value: string; labelKey: string }[]
  helpKey?: string
}

/** Extra non-secret settings fields per provider (AP.5 / AP.8 field matrix). */
export const PROVIDER_SETTING_FIELDS: Record<AIProviderId, ProviderSettingField[]> = {
  openrouter: [],
  anthropic: [{ key: 'base_url', labelKey: 'settings.ai.fields.baseUrl', placeholder: 'https://api.anthropic.com' }],
  openai: [{ key: 'base_url', labelKey: 'settings.ai.fields.baseUrl', placeholder: 'https://api.openai.com/v1' }],
  azure_openai: [
    {
      key: 'azure_base_url',
      labelKey: 'settings.ai.fields.azureBaseUrl',
      placeholder: 'https://contoso.openai.azure.com',
      required: true,
    },
    {
      key: 'azure_api_version',
      labelKey: 'settings.ai.fields.azureApiVersion',
      placeholder: '2024-10-21',
    },
    {
      key: 'default_deployment',
      labelKey: 'settings.ai.fields.defaultDeployment',
      placeholder: 'gpt-4o',
    },
    {
      key: 'deployments',
      labelKey: 'settings.ai.fields.deployments',
      type: 'textarea',
      placeholder: '{"gpt-4o":"gpt4o-prod","text-fast":"gpt4o-mini"}',
      helpKey: 'settings.ai.fields.deploymentsHelp',
    },
  ],
  bedrock: [
    {
      key: 'auth_mode',
      labelKey: 'settings.ai.fields.authMode',
      type: 'select',
      options: [
        { value: 'api_key', labelKey: 'settings.ai.authMode.apiKey' },
        { value: 'access_key', labelKey: 'settings.ai.authMode.accessKey' },
        { value: 'iam_role', labelKey: 'settings.ai.authMode.iamRole' },
      ],
    },
    {
      key: 'aws_region',
      labelKey: 'settings.ai.fields.awsRegion',
      placeholder: 'us-east-1',
      required: true,
    },
    {
      key: 'bedrock_base_url',
      labelKey: 'settings.ai.fields.bedrockBaseUrl',
      placeholder: 'https://bedrock-runtime.us-east-1.amazonaws.com',
      whenAuthModes: ['api_key'],
    },
  ],
  vertex: [
    {
      key: 'auth_mode',
      labelKey: 'settings.ai.fields.authMode',
      type: 'select',
      options: [
        { value: 'api_key', labelKey: 'settings.ai.authMode.apiKey' },
        { value: 'service_account', labelKey: 'settings.ai.authMode.serviceAccount' },
        { value: 'adc', labelKey: 'settings.ai.authMode.adc' },
      ],
    },
    { key: 'gcp_project', labelKey: 'settings.ai.fields.gcpProject', placeholder: 'my-gcp-project', required: true },
    { key: 'gcp_location', labelKey: 'settings.ai.fields.gcpLocation', placeholder: 'us-central1', required: true },
  ],
}

export type AIProviderCredential = {
  provider: string
  enabled: boolean
  apiKeyConfigured: boolean
  apiKey?: string
  authMode?: string
  secretsConfigured?: Record<string, boolean>
  awsAccessKeyIdConfigured?: boolean
  awsSecretAccessKeyConfigured?: boolean
  serviceAccountJsonConfigured?: boolean
  settings?: Record<string, unknown>
  updatedAt?: string
  updatedBy?: string
}

export type PlatformAIProvidersResponse = {
  credentials: AIProviderCredential[]
  providers: string[]
  tenantByokAllowed: boolean
  tenantAllowedProviders: string[]
}

export type OrgAIProviderCredentialInput = {
  provider: string
  enabled?: boolean
  apiKey?: string
  clearApiKey?: boolean
  awsAccessKeyId?: string
  clearAwsAccessKeyId?: boolean
  awsSecretAccessKey?: string
  clearAwsSecretAccessKey?: boolean
  serviceAccountJson?: string
  clearServiceAccountJson?: boolean
  settings?: Record<string, unknown>
}

export type OrgAISettings = {
  orgId?: string
  provider?: string
  modelAlias?: string
  fallbackProvider?: string | null
  byokConfigured?: boolean
  settings?: Record<string, unknown>
  credentials?: AIProviderCredential[]
  providers?: string[]
  modelAliases?: string[]
  modelAliasCatalog?: { alias: string; label?: string }[]
  registryVersion?: string
}

export type OrgAISettingsPutBody = {
  provider: string
  modelAlias: string
  fallbackProvider?: string | null
  byokApiKey?: string
  clearByokApiKey?: boolean
  settings?: Record<string, unknown>
  credentials?: OrgAIProviderCredentialInput[]
}

export type ProviderCredentialDraft = {
  enabled: boolean
  apiKey: string
  apiKeyBaseline: string
  awsAccessKeyId: string
  awsAccessKeyIdBaseline: string
  awsSecretAccessKey: string
  awsSecretAccessKeyBaseline: string
  serviceAccountJson: string
  serviceAccountJsonBaseline: string
  settings: Record<string, string>
}

export function isKnownAIProvider(id: string): id is AIProviderId {
  return (AI_PROVIDER_IDS as string[]).includes(id)
}

export function providerLabel(id: string): string {
  if (isKnownAIProvider(id)) return AI_PROVIDER_LABELS[id]
  return id
}

export function authModeFromDraft(provider: string, settings: Record<string, string>): AuthMode {
  const mode = (settings.auth_mode ?? '').trim() as AuthMode
  if (mode) return mode
  if (provider === 'bedrock' || provider === 'vertex' || provider === 'azure_openai') return 'api_key'
  return 'api_key'
}

export function settingsFromCredential(cred: AIProviderCredential | undefined): Record<string, string> {
  const out: Record<string, string> = {}
  if (!cred?.settings) return out
  for (const [k, v] of Object.entries(cred.settings)) {
    if (v == null) continue
    if (k === 'deployments' && typeof v === 'object') {
      out[k] = JSON.stringify(v)
      continue
    }
    out[k] = typeof v === 'string' ? v : String(v)
  }
  if (cred.authMode && !out.auth_mode) {
    out.auth_mode = cred.authMode
  }
  return out
}

function secretPlaceholder(configured: boolean): string {
  return configured ? PLATFORM_SECRET_PLACEHOLDER : ''
}

export function draftFromCredential(cred: AIProviderCredential): ProviderCredentialDraft {
  const apiConfigured = cred.apiKeyConfigured === true
  const awsIdConfigured = cred.awsAccessKeyIdConfigured === true
  const awsSecretConfigured = cred.awsSecretAccessKeyConfigured === true
  const saConfigured = cred.serviceAccountJsonConfigured === true
  return {
    enabled: cred.enabled !== false,
    apiKey: secretPlaceholder(apiConfigured),
    apiKeyBaseline: secretPlaceholder(apiConfigured),
    awsAccessKeyId: secretPlaceholder(awsIdConfigured),
    awsAccessKeyIdBaseline: secretPlaceholder(awsIdConfigured),
    awsSecretAccessKey: secretPlaceholder(awsSecretConfigured),
    awsSecretAccessKeyBaseline: secretPlaceholder(awsSecretConfigured),
    serviceAccountJson: secretPlaceholder(saConfigured),
    serviceAccountJsonBaseline: secretPlaceholder(saConfigured),
    settings: settingsFromCredential(cred),
  }
}

/**
 * Builds apiKey / clearApiKey for PUT when the secret field changed.
 * Unchanged placeholder or empty (when never configured) → omit.
 */
export function buildSecretUpdate(
  draft: Pick<ProviderCredentialDraft, 'apiKey' | 'apiKeyBaseline'>,
  configured: boolean,
): { apiKey?: string; clearApiKey?: boolean } {
  const trimmed = draft.apiKey.trim()
  const baseline = draft.apiKeyBaseline.trim()
  if (trimmed === baseline) return {}
  if (trimmed && trimmed !== PLATFORM_SECRET_PLACEHOLDER) {
    return { apiKey: trimmed }
  }
  if (baseline === PLATFORM_SECRET_PLACEHOLDER && trimmed === '' && configured) {
    return { clearApiKey: true }
  }
  return {}
}

function buildNamedSecretUpdate(
  value: string,
  baseline: string,
  configured: boolean,
): { set?: string; clear?: boolean } {
  const trimmed = value.trim()
  const base = baseline.trim()
  if (trimmed === base) return {}
  if (trimmed && trimmed !== PLATFORM_SECRET_PLACEHOLDER) {
    return { set: trimmed }
  }
  if (base === PLATFORM_SECRET_PLACEHOLDER && trimmed === '' && configured) {
    return { clear: true }
  }
  return {}
}

/** Builds all secret fields for a credential PUT (AP.8 multi-secret). */
export function buildCredentialSecretUpdate(
  draft: ProviderCredentialDraft,
  cred: AIProviderCredential | undefined,
): OrgAIProviderCredentialInput {
  const out: OrgAIProviderCredentialInput = { provider: cred?.provider ?? '' }
  const api = buildSecretUpdate(draft, cred?.apiKeyConfigured === true)
  if (api.apiKey) out.apiKey = api.apiKey
  if (api.clearApiKey) out.clearApiKey = true

  const awsId = buildNamedSecretUpdate(
    draft.awsAccessKeyId,
    draft.awsAccessKeyIdBaseline,
    cred?.awsAccessKeyIdConfigured === true,
  )
  if (awsId.set) out.awsAccessKeyId = awsId.set
  if (awsId.clear) out.clearAwsAccessKeyId = true

  const awsSecret = buildNamedSecretUpdate(
    draft.awsSecretAccessKey,
    draft.awsSecretAccessKeyBaseline,
    cred?.awsSecretAccessKeyConfigured === true,
  )
  if (awsSecret.set) out.awsSecretAccessKey = awsSecret.set
  if (awsSecret.clear) out.clearAwsSecretAccessKey = true

  const sa = buildNamedSecretUpdate(
    draft.serviceAccountJson,
    draft.serviceAccountJsonBaseline,
    cred?.serviceAccountJsonConfigured === true,
  )
  if (sa.set) out.serviceAccountJson = sa.set
  if (sa.clear) out.clearServiceAccountJson = true

  return out
}

export function settingsObjectFromDraft(
  provider: string,
  settings: Record<string, string>,
): Record<string, unknown> {
  const fields = isKnownAIProvider(provider) ? PROVIDER_SETTING_FIELDS[provider] : []
  const out: Record<string, unknown> = {}
  const authMode = authModeFromDraft(provider, settings)
  for (const field of fields) {
    if (field.whenAuthModes && !field.whenAuthModes.includes(authMode)) continue
    const v = (settings[field.key] ?? '').trim()
    if (!v) continue
    if (field.key === 'deployments') {
      try {
        const parsed = JSON.parse(v) as unknown
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          out.deployments = parsed
          continue
        }
      } catch {
        // fall through — store as string so server validation can reject
      }
    }
    out[field.key] = v
  }
  return out
}

export function visibleSettingFields(provider: string, settings: Record<string, string>): ProviderSettingField[] {
  if (!isKnownAIProvider(provider)) return []
  const authMode = authModeFromDraft(provider, settings)
  return PROVIDER_SETTING_FIELDS[provider].filter(
    (f) => !f.whenAuthModes || f.whenAuthModes.includes(authMode),
  )
}

export function showsApiKeyField(provider: string, settings: Record<string, string>): boolean {
  const mode = authModeFromDraft(provider, settings)
  if (provider === 'bedrock') return mode === 'api_key'
  if (provider === 'vertex') return mode === 'api_key'
  return true
}

export function showsAwsAccessKeyFields(provider: string, settings: Record<string, string>): boolean {
  return provider === 'bedrock' && authModeFromDraft(provider, settings) === 'access_key'
}

export function showsServiceAccountField(provider: string, settings: Record<string, string>): boolean {
  return provider === 'vertex' && authModeFromDraft(provider, settings) === 'service_account'
}

export async function fetchPlatformAIProviders(): Promise<PlatformAIProvidersResponse> {
  const res = await authorizedFetch('/api/v1/settings/ai/providers')
  if (res.status === 404) {
    const err = new Error('AI_PROVIDER_ABSTRACTION_DISABLED') as Error & { code?: string }
    err.code = 'AI_PROVIDER_ABSTRACTION_DISABLED'
    throw err
  }
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  return (await res.json()) as PlatformAIProvidersResponse
}

export async function putPlatformAIProvider(
  provider: string,
  body: {
    enabled?: boolean
    apiKey?: string
    clearApiKey?: boolean
    awsAccessKeyId?: string
    clearAwsAccessKeyId?: boolean
    awsSecretAccessKey?: string
    clearAwsSecretAccessKey?: boolean
    serviceAccountJson?: string
    clearServiceAccountJson?: boolean
    settings?: Record<string, unknown>
  },
): Promise<AIProviderCredential> {
  const res = await authorizedFetch(`/api/v1/settings/ai/providers/${encodeURIComponent(provider)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  return (await res.json()) as AIProviderCredential
}

export async function deletePlatformAIProvider(provider: string): Promise<void> {
  const res = await authorizedFetch(`/api/v1/settings/ai/providers/${encodeURIComponent(provider)}`, {
    method: 'DELETE',
  })
  if (!res.ok && res.status !== 204) {
    throw new Error(await readApiErrorMessage(res))
  }
}

export async function putPlatformAIProviderPolicy(body: {
  tenantByokAllowed?: boolean
  tenantAllowedProviders?: string[]
}): Promise<{ tenantByokAllowed: boolean; tenantAllowedProviders: string[] }> {
  const res = await authorizedFetch('/api/v1/settings/ai/providers', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  return (await res.json()) as { tenantByokAllowed: boolean; tenantAllowedProviders: string[] }
}

export async function fetchOrgAISettings(): Promise<OrgAISettings> {
  const res = await authorizedFetch('/api/v1/admin/ai-settings')
  if (res.status === 404) {
    const err = new Error('AI_PROVIDER_ABSTRACTION_DISABLED') as Error & { code?: string }
    err.code = 'AI_PROVIDER_ABSTRACTION_DISABLED'
    throw err
  }
  if (res.status === 403) {
    const err = new Error('FORBIDDEN') as Error & { code?: string }
    err.code = 'FORBIDDEN'
    throw err
  }
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  return (await res.json()) as OrgAISettings
}

export async function putOrgAISettings(body: OrgAISettingsPutBody): Promise<OrgAISettings> {
  const res = await authorizedFetch('/api/v1/admin/ai-settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  return (await res.json()) as OrgAISettings
}

export async function testOrgAIConnection(): Promise<{
  ok?: boolean
  provider?: string
  authMode?: string
  modelAlias?: string
  modelId?: string
  latencyMs?: number
  totalLatencyMs?: number
  promptTokens?: number
  completionTokens?: number
  responsePreview?: string
}> {
  const res = await authorizedFetch('/api/v1/admin/ai-settings/test', { method: 'POST' })
  if (!res.ok) {
    throw new Error(await readApiErrorMessage(res))
  }
  return (await res.json()) as {
    ok?: boolean
    provider?: string
    authMode?: string
    modelAlias?: string
    modelId?: string
    latencyMs?: number
    totalLatencyMs?: number
    promptTokens?: number
    completionTokens?: number
    responsePreview?: string
  }
}

export function anyProviderConfigured(credentials: AIProviderCredential[]): boolean {
  return credentials.some(
    (c) =>
      c.apiKeyConfigured === true ||
      c.awsAccessKeyIdConfigured === true ||
      c.serviceAccountJsonConfigured === true ||
      c.authMode === 'iam_role' ||
      c.authMode === 'adc',
  )
}

export function emptyCredentialDraft(): ProviderCredentialDraft {
  return {
    enabled: true,
    apiKey: '',
    apiKeyBaseline: '',
    awsAccessKeyId: '',
    awsAccessKeyIdBaseline: '',
    awsSecretAccessKey: '',
    awsSecretAccessKeyBaseline: '',
    serviceAccountJson: '',
    serviceAccountJsonBaseline: '',
    settings: {},
  }
}
