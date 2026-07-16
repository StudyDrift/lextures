import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'
import { PLATFORM_SECRET_PLACEHOLDER } from '../../../lib/platform-settings'

const { fetchOrgAISettings, putOrgAISettings, testOrgAIConnection } = vi.hoisted(() => ({
  fetchOrgAISettings: vi.fn(),
  putOrgAISettings: vi.fn(),
  testOrgAIConnection: vi.fn(),
}))

vi.mock('../../../lib/ai-providers', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../lib/ai-providers')>()
  return {
    ...actual,
    fetchOrgAISettings,
    putOrgAISettings,
    testOrgAIConnection,
  }
})

vi.mock('../../../lib/lms-toast', () => ({
  toastSaveOk: vi.fn(),
  toastMutationError: vi.fn(),
}))

import { toastSaveOk } from '../../../lib/lms-toast'
import { AiProviderSettingsPanel } from '../ai-provider-settings-panel'

describe('AiProviderSettingsPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    fetchOrgAISettings.mockResolvedValue({
      provider: 'azure_openai',
      modelAlias: 'gpt-4o',
      fallbackProvider: null,
      byokConfigured: false,
      credentials: [
        {
          provider: 'azure_openai',
          enabled: true,
          apiKeyConfigured: true,
          apiKey: PLATFORM_SECRET_PLACEHOLDER,
          settings: { azure_base_url: 'https://example.openai.azure.com' },
        },
      ],
      providers: ['openrouter', 'anthropic', 'openai', 'azure_openai', 'bedrock', 'vertex'],
      modelAliases: ['claude-3-5-sonnet', 'gpt-4o'],
    })
  })

  it('loads org settings and shows Azure credential fields', async () => {
    render(
      <I18nProvider>
        <AiProviderSettingsPanel />
      </I18nProvider>,
    )
    await waitFor(() => expect(screen.getByText('AI provider')).toBeInTheDocument())
    expect(screen.getByDisplayValue('https://example.openai.azure.com')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Test connection/i })).toBeInTheDocument()
  })

  it('shows abstraction-disabled banner instead of silent empty', async () => {
    const err = new Error('AI_PROVIDER_ABSTRACTION_DISABLED') as Error & { code?: string }
    err.code = 'AI_PROVIDER_ABSTRACTION_DISABLED'
    fetchOrgAISettings.mockRejectedValueOnce(err)
    render(
      <I18nProvider>
        <AiProviderSettingsPanel />
      </I18nProvider>,
    )
    await waitFor(() =>
      expect(screen.getByText(/Multi-provider AI is disabled/i)).toBeInTheDocument(),
    )
  })

  it('toasts latency and provider name on successful test connection', async () => {
    testOrgAIConnection.mockResolvedValueOnce({
      provider: 'azure_openai',
      latencyMs: 42,
      responsePreview: 'Hello',
    })
    const user = userEvent.setup()
    render(
      <I18nProvider>
        <AiProviderSettingsPanel />
      </I18nProvider>,
    )
    await waitFor(() => expect(screen.getByRole('button', { name: /Test connection/i })).toBeInTheDocument())
    await user.click(screen.getByRole('button', { name: /Test connection/i }))
    await waitFor(() => {
      expect(toastSaveOk).toHaveBeenCalledWith(
        expect.stringMatching(/Azure OpenAI.*42.*Hello/i),
      )
    })
  })
})
