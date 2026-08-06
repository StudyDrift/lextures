import { useCallback, useEffect, useState } from 'react'
import { Bot, Copy } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

type MCPConfigResponse = {
  apiBaseUrl: string
  cursorConfig: Record<string, unknown>
  claudeDesktopConfig: Record<string, unknown>
  instructions: string[]
}

export function IntegrationsMcpPanel() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [config, setConfig] = useState<MCPConfigResponse | null>(null)
  const [tokenDraft, setTokenDraft] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await authorizedFetch('/api/v1/me/integrations/mcp')
      const raw: unknown = await res.json().catch(() => ({}))
      if (!res.ok) throw new Error(readApiErrorMessage(raw))
      setConfig(raw as MCPConfigResponse)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not load MCP setup.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  function configWithToken(base: Record<string, unknown>, token: string): string {
    const draft = structuredClone(base) as {
      mcpServers?: Record<string, { env?: Record<string, string> }>
    }
    const env = draft.mcpServers?.lextures?.env
    if (env && token.trim()) {
      env.LEXTURES_API_TOKEN = token.trim()
    }
    return JSON.stringify(draft, null, 2)
  }

  async function copyJson(text: string) {
    try {
      await navigator.clipboard.writeText(text)
      toastSaveOk('MCP config copied.')
    } catch {
      toastMutationError('Could not copy to clipboard.')
    }
  }

  return (
    <section className="mt-10 border-t border-border-default pt-8 dark:border-border-default">
      <div>
        <h3 className="flex items-center gap-2 text-sm font-semibold text-fg-default">
          <Bot className="h-4 w-4" aria-hidden />
          MCP for AI agents
        </h3>
        <p className="mt-1 text-sm text-fg-muted">
          Connect Cursor, Claude Desktop, or other MCP clients to Lextures using an access key with the{' '}
          <code className="font-mono text-xs">mcp:connect</code> scope.
        </p>
      </div>

      {error && (
        <p className="mt-4 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-800 dark:border-rose-900/50 dark:bg-rose-950/40 dark:text-rose-200">
          {error}
        </p>
      )}

      {loading && <p className="mt-4 text-sm text-fg-muted">Loading MCP setup…</p>}

      {!loading && config && (
        <div className="mt-4 space-y-6">
          <ol className="list-decimal space-y-2 ps-5 text-sm text-fg-muted">
            {config.instructions.map((step) => (
              <li key={step}>{step}</li>
            ))}
          </ol>

          <div>
            <label className="block text-sm font-medium text-fg-default">
              Paste access key (optional — included when copying config)
            </label>
            <input
              type="password"
              autoComplete="off"
              value={tokenDraft}
              onChange={(e) => setTokenDraft(e.target.value)}
              placeholder="ltk_…"
              className="mt-1 w-full rounded-xl border border-border-default bg-surface-raised px-3 py-2 font-mono text-sm outline-none ring-indigo-500/20 focus:border-indigo-400 focus:ring-2 dark:border-border-default dark:bg-surface-overlay dark:text-fg-default"
            />
          </div>

          <div>
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-sm font-medium text-fg-default">Cursor / Claude Desktop config</p>
              <button
                type="button"
                onClick={() => void copyJson(configWithToken(config.cursorConfig, tokenDraft))}
                className="inline-flex items-center gap-1 rounded-lg border border-border-default px-3 py-1.5 text-sm font-medium text-fg-muted hover:bg-surface-base dark:border-border-default dark:text-fg-default dark:hover:bg-surface-overlay"
              >
                <Copy className="h-4 w-4" aria-hidden />
                Copy JSON
              </button>
            </div>
            <pre className="mt-2 max-h-80 overflow-auto rounded-xl border border-border-default bg-surface-base p-3 text-xs text-fg-default dark:border-border-default dark:bg-surface-raised dark:text-fg-default">
              {configWithToken(config.cursorConfig, tokenDraft)}
            </pre>
          </div>

          <div>
            <p className="text-sm font-medium text-fg-default">API base URL</p>
            <p className="mt-1 font-mono text-xs text-fg-muted">{config.apiBaseUrl}</p>
          </div>

          <p className="text-xs text-fg-muted">
            The MCP server runs from <code className="font-mono">clients/mcp/dist/index.js</code> in this repository
            and calls the Lextures API with your key. Build it with{' '}
            <code className="font-mono">cd clients/mcp && npm install && npm run build</code> before connecting.
          </p>
        </div>
      )}
    </section>
  )
}
