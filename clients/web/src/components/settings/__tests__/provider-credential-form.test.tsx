import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'
import { PLATFORM_SECRET_PLACEHOLDER } from '../../../lib/platform-settings'
import { draftFromCredential } from '../../../lib/ai-providers'
import { ProviderCredentialForm } from '../provider-credential-form'

describe('ProviderCredentialForm', () => {
  it('renders status badges and Azure extra fields', () => {
    const cred = {
      provider: 'azure_openai',
      enabled: true,
      apiKeyConfigured: true,
      apiKey: PLATFORM_SECRET_PLACEHOLDER,
      settings: { azure_base_url: 'https://example.openai.azure.com' },
    }
    render(
      <I18nProvider>
        <ul>
          <ProviderCredentialForm
            provider="azure_openai"
            credential={cred}
            draft={draftFromCredential(cred)}
            active
            onChange={vi.fn()}
            onSave={vi.fn()}
            onClear={vi.fn()}
          />
        </ul>
      </I18nProvider>,
    )
    expect(screen.getByText('Azure OpenAI')).toBeInTheDocument()
    expect(screen.getByText('Configured')).toBeInTheDocument()
    expect(screen.getByText('Enabled')).toBeInTheDocument()
    expect(screen.getByText('Default')).toBeInTheDocument()
    expect(screen.getByText('Azure endpoint URL *')).toBeInTheDocument()
    expect(screen.getByDisplayValue('https://example.openai.azure.com')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Clear key/i })).toBeInTheDocument()
    expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'true')
  })

  it('hides clear button when not configured', () => {
    const cred = {
      provider: 'openrouter',
      enabled: true,
      apiKeyConfigured: false,
      settings: {},
    }
    render(
      <I18nProvider>
        <ul>
          <ProviderCredentialForm
            provider="openrouter"
            credential={cred}
            draft={draftFromCredential(cred)}
            onChange={vi.fn()}
            onSave={vi.fn()}
            onClear={vi.fn()}
          />
        </ul>
      </I18nProvider>,
    )
    expect(screen.queryByRole('button', { name: /Clear key/i })).not.toBeInTheDocument()
    expect(screen.getByText('Not configured')).toBeInTheDocument()
  })
})
