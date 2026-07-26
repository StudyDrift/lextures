import { render, screen, waitFor } from '@testing-library/react'
import { lazy, type ComponentType } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { serializeFence } from '../../../../lib/content-tools/lex-tool-fence'
import type { ContentToolRendererProps } from '../runtime-contract'

const instanceId = 'a1b2c3d4-e5f6-7890-abcd-ef1234567890'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: { defaultValue?: string }) => opts?.defaultValue ?? key,
  }),
}))

vi.mock('../../../../context/use-permissions', () => ({
  usePermissions: () => ({
    allows: () => false,
    loading: false,
    error: null,
    refresh: async () => undefined,
  }),
}))

vi.mock('../registry', () => {
  const MockRenderer = (props: ContentToolRendererProps) => (
    <div data-testid="mock-renderer">
      <span data-testid="tool-id">{props.toolId}</span>
      <span data-testid="instance-id">{props.instanceId}</span>
      <span data-testid="prompt">{String(props.config.prompt ?? '')}</span>
    </div>
  )
  return {
    resolveRenderer: (toolId: string) =>
      toolId === 'noop_probe'
        ? lazy(async () => ({
            default: MockRenderer as ComponentType<ContentToolRendererProps>,
          }))
        : null,
    isRendererRegistered: (toolId: string) => toolId === 'noop_probe',
  }
})

vi.mock('../../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../../lib/courses-api')>(
    '../../../../lib/courses-api',
  )
  return {
    ...actual,
    fetchContentToolsInstances: vi.fn(),
    getContentToolState: vi.fn(),
    putContentToolState: vi.fn(),
    submitContentToolState: vi.fn(),
    runContentToolAction: vi.fn(),
  }
})

import { fetchContentToolsInstances } from '../../../../lib/courses-api'
import { ContentToolHost } from '../content-tool-host'
import { ContentToolsPageProvider } from '../content-tools-page-context'
import { LiveRegionProvider } from '../tool-live-region'

describe('ContentToolHost from lex-tool fence', () => {
  beforeEach(() => {
    vi.mocked(fetchContentToolsInstances).mockReset()
    vi.mocked(fetchContentToolsInstances).mockResolvedValue([
      {
        id: instanceId,
        toolId: 'noop_probe',
        toolVersion: '1.0.0',
        hostKind: 'content_page',
        config: { prompt: 'Probe prompt' },
        status: 'active',
        updatedAt: '2026-01-01T00:00:00Z',
        state: {
          instanceId,
          revision: 0,
          status: 'not_started',
          state: {},
          stateJson: {},
          score: null,
        },
      },
    ])
  })

  it('renders the registered renderer for a valid fence payload', async () => {
    const fenceText = serializeFence({ instanceId, toolId: 'noop_probe' })
    render(
      <LiveRegionProvider>
        <ContentToolsPageProvider courseCode="DEMO" itemId="item-1" hostKind="content_page">
          <ContentToolHost fenceText={fenceText} />
        </ContentToolsPageProvider>
      </LiveRegionProvider>,
    )

    await waitFor(() => {
      expect(screen.getByTestId('mock-renderer')).toBeInTheDocument()
    })
    expect(screen.getByTestId('tool-id')).toHaveTextContent('noop_probe')
    expect(screen.getByTestId('instance-id')).toHaveTextContent(instanceId)
    expect(screen.getByTestId('prompt')).toHaveTextContent('Probe prompt')
    expect(fetchContentToolsInstances).toHaveBeenCalledWith('DEMO', {
      itemId: 'item-1',
      hostKind: 'content_page',
      withState: true,
    })
  })

  it('shows unavailable placeholder for invalid fence JSON', async () => {
    render(
      <LiveRegionProvider>
        <ContentToolsPageProvider courseCode="DEMO">
          <ContentToolHost fenceText="{not-json" />
        </ContentToolsPageProvider>
      </LiveRegionProvider>,
    )
    expect(await screen.findByText('contentTools.runtime.unavailable')).toBeInTheDocument()
  })
})
