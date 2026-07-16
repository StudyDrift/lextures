import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'
import { PLATFORM_SECRET_PLACEHOLDER } from '../../../lib/platform-settings'

const {
  fetchPlatformAIProviders,
  putPlatformAIProvider,
  deletePlatformAIProvider,
  putPlatformAIProviderPolicy,
} = vi.hoisted(() => ({
  fetchPlatformAIProviders: vi.fn(),
  putPlatformAIProvider: vi.fn(),
  deletePlatformAIProvider: vi.fn(),
  putPlatformAIProviderPolicy: vi.fn(),
}))

vi.mock('../../../lib/ai-providers', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../lib/ai-providers')>()
  return {
    ...actual,
    fetchPlatformAIProviders,
    putPlatformAIProvider,
    deletePlatformAIProvider,
    putPlatformAIProviderPolicy,
  }
})

vi.mock('../../../lib/lms-toast', () => ({
  toastSaveOk: vi.fn(),
  toastMutationError: vi.fn(),
}))

import { AiProvidersPanel } from '../ai-providers-panel'

const credentials = [
  {
    provider: 'openrouter',
    enabled: true,
    apiKeyConfigured: false,
    apiKey: '',
    settings: {},
  },
  {
    provider: 'anthropic',
    enabled: true,
    apiKeyConfigured: true,
    apiKey: PLATFORM_SECRET_PLACEHOLDER,
    settings: {},
  },
  {
    provider: 'azure_openai',
    enabled: true,
    apiKeyConfigured: false,
    apiKey: '',
    settings: {},
  },
]

function renderPanel(activeProvider = 'anthropic') {
  return render(
    <I18nProvider>
      <AiProvidersPanel activeProvider={activeProvider} />
    </I18nProvider>,
  )
}

describe('AiProvidersPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    fetchPlatformAIProviders.mockResolvedValue({
      credentials,
      providers: credentials.map((c) => c.provider),
      tenantByokAllowed: true,
      tenantAllowedProviders: [],
    })
  })

  it('renders provider cards with configured / not configured / default badges', async () => {
    renderPanel()
    await waitFor(() => expect(screen.getByText('OpenRouter')).toBeInTheDocument())
    expect(screen.getByText('Anthropic')).toBeInTheDocument()
    expect(screen.getByText('Azure OpenAI')).toBeInTheDocument()
    expect(screen.getAllByText('Configured').length).toBeGreaterThanOrEqual(1)
    expect(screen.getAllByText('Not configured').length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('Default')).toBeInTheDocument()
  })

  it('shows empty-state copy when no providers are configured', async () => {
    fetchPlatformAIProviders.mockResolvedValueOnce({
      credentials: credentials.map((c) => ({ ...c, apiKeyConfigured: false, apiKey: '' })),
      providers: credentials.map((c) => c.provider),
      tenantByokAllowed: true,
      tenantAllowedProviders: [],
    })
    renderPanel('')
    await waitFor(() =>
      expect(screen.getByText(/No AI providers are configured yet/i)).toBeInTheDocument(),
    )
    expect(screen.getByRole('link', { name: /Configure AI providers/i })).toHaveAttribute(
      'href',
      expect.stringContaining('ai-providers'),
    )
  })

  it('shows abstraction-disabled banner on 404', async () => {
    const err = new Error('AI_PROVIDER_ABSTRACTION_DISABLED') as Error & { code?: string }
    err.code = 'AI_PROVIDER_ABSTRACTION_DISABLED'
    fetchPlatformAIProviders.mockRejectedValueOnce(err)
    renderPanel()
    await waitFor(() =>
      expect(screen.getByText(/Multi-provider AI is disabled/i)).toBeInTheDocument(),
    )
  })

  it('saves a provider credential with Azure fields', async () => {
    putPlatformAIProvider.mockResolvedValueOnce({
      provider: 'azure_openai',
      enabled: true,
      apiKeyConfigured: true,
      apiKey: PLATFORM_SECRET_PLACEHOLDER,
      settings: {
        azure_base_url: 'https://example.openai.azure.com',
        azure_api_version: '2024-10-21',
      },
    })
    const user = userEvent.setup()
    renderPanel()
    await waitFor(() => expect(screen.getByText('Azure OpenAI')).toBeInTheDocument())

    const endpoint = screen.getByPlaceholderText('https://contoso.openai.azure.com')
    await user.type(endpoint, 'https://example.openai.azure.com')
    const version = screen.getByPlaceholderText('2024-10-21')
    await user.type(version, '2024-10-21')

    const azureCard = screen.getByText('Azure OpenAI').closest('li')
    expect(azureCard).toBeTruthy()
    const saveButtons = azureCard!.querySelectorAll('button')
    const saveBtn = Array.from(saveButtons).find((b) => b.textContent === 'Save')
    expect(saveBtn).toBeTruthy()
    await user.click(saveBtn!)

    await waitFor(() => {
      expect(putPlatformAIProvider).toHaveBeenCalledWith(
        'azure_openai',
        expect.objectContaining({
          enabled: true,
          settings: expect.objectContaining({
            azure_base_url: 'https://example.openai.azure.com',
            azure_api_version: '2024-10-21',
          }),
        }),
      )
    })
  })
})
