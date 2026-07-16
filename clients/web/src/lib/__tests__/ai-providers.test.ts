import { describe, expect, it } from 'vitest'
import { PLATFORM_SECRET_PLACEHOLDER } from '../platform-settings'
import {
  anyProviderConfigured,
  authModeFromDraft,
  buildCredentialSecretUpdate,
  buildSecretUpdate,
  draftFromCredential,
  emptyCredentialDraft,
  PROVIDER_SETTING_FIELDS,
  settingsObjectFromDraft,
  showsApiKeyField,
  showsAwsAccessKeyFields,
  showsServiceAccountField,
  type AIProviderCredential,
} from '../ai-providers'

describe('ai-providers helpers', () => {
  it('exposes Azure/Bedrock/Vertex field matrix keys including auth modes', () => {
    expect(PROVIDER_SETTING_FIELDS.azure_openai.map((f) => f.key)).toEqual([
      'azure_base_url',
      'azure_api_version',
      'default_deployment',
      'deployments',
    ])
    expect(PROVIDER_SETTING_FIELDS.bedrock.map((f) => f.key)).toContain('auth_mode')
    expect(PROVIDER_SETTING_FIELDS.bedrock.map((f) => f.key)).toContain('aws_region')
    expect(PROVIDER_SETTING_FIELDS.vertex.map((f) => f.key)).toContain('auth_mode')
    expect(PROVIDER_SETTING_FIELDS.openrouter).toEqual([])
  })

  it('builds drafts with placeholder when configured', () => {
    const cred: AIProviderCredential = {
      provider: 'anthropic',
      enabled: true,
      apiKeyConfigured: true,
      settings: { base_url: 'https://api.anthropic.com' },
    }
    const draft = draftFromCredential(cred)
    expect(draft.apiKey).toBe(PLATFORM_SECRET_PLACEHOLDER)
    expect(draft.apiKeyBaseline).toBe(PLATFORM_SECRET_PLACEHOLDER)
    expect(draft.settings.base_url).toBe('https://api.anthropic.com')
  })

  it('omits secret update when placeholder unchanged', () => {
    expect(
      buildSecretUpdate(
        { apiKey: PLATFORM_SECRET_PLACEHOLDER, apiKeyBaseline: PLATFORM_SECRET_PLACEHOLDER },
        true,
      ),
    ).toEqual({})
  })

  it('sets apiKey when a new secret is entered', () => {
    expect(
      buildSecretUpdate(
        { apiKey: 'sk-new', apiKeyBaseline: PLATFORM_SECRET_PLACEHOLDER },
        true,
      ),
    ).toEqual({ apiKey: 'sk-new' })
  })

  it('clears apiKey when placeholder field is emptied', () => {
    expect(
      buildSecretUpdate({ apiKey: '', apiKeyBaseline: PLATFORM_SECRET_PLACEHOLDER }, true),
    ).toEqual({ clearApiKey: true })
  })

  it('builds multi-secret credential update for Bedrock access keys', () => {
    const draft = emptyCredentialDraft()
    draft.awsAccessKeyId = 'AKIATEST'
    draft.awsSecretAccessKey = 'secret'
    const update = buildCredentialSecretUpdate(draft, {
      provider: 'bedrock',
      enabled: true,
      apiKeyConfigured: false,
    })
    expect(update.awsAccessKeyId).toBe('AKIATEST')
    expect(update.awsSecretAccessKey).toBe('secret')
  })

  it('parses deployments JSON into settings object', () => {
    expect(
      settingsObjectFromDraft('azure_openai', {
        azure_base_url: ' https://example.openai.azure.com ',
        azure_api_version: '',
        default_deployment: 'gpt-4o',
        deployments: '{"gpt-4o":"gpt4o-prod"}',
      }),
    ).toEqual({
      azure_base_url: 'https://example.openai.azure.com',
      default_deployment: 'gpt-4o',
      deployments: { 'gpt-4o': 'gpt4o-prod' },
    })
  })

  it('gates secret fields by auth mode', () => {
    expect(showsApiKeyField('bedrock', { auth_mode: 'iam_role' })).toBe(false)
    expect(showsAwsAccessKeyFields('bedrock', { auth_mode: 'access_key' })).toBe(true)
    expect(showsServiceAccountField('vertex', { auth_mode: 'service_account' })).toBe(true)
    expect(authModeFromDraft('vertex', {})).toBe('api_key')
  })

  it('detects any configured provider including iam_role', () => {
    expect(anyProviderConfigured([])).toBe(false)
    expect(
      anyProviderConfigured([
        { provider: 'openrouter', enabled: true, apiKeyConfigured: false },
        { provider: 'anthropic', enabled: true, apiKeyConfigured: true },
      ]),
    ).toBe(true)
    expect(
      anyProviderConfigured([{ provider: 'bedrock', enabled: true, apiKeyConfigured: false, authMode: 'iam_role' }]),
    ).toBe(true)
  })
})
