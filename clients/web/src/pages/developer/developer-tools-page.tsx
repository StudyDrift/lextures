import { useEffect, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import {
  createDeveloperTool,
  listDeveloperTools,
  type DeveloperTool,
} from '../../lib/content-tool-marketplace-api'

export default function DeveloperToolsPage() {
  const { t } = useTranslation('contentTools')
  const titleId = useId()
  const { ffContentToolMarketplace } = usePlatformFeatures()
  const [tools, setTools] = useState<DeveloperTool[]>([])
  const [error, setError] = useState<string | null>(null)
  const [toolId, setToolId] = useState('acme.demo_lab')
  const [displayName, setDisplayName] = useState('Demo Lab')
  const [summary, setSummary] = useState('A sample interactive lab')
  const [busy, setBusy] = useState(false)

  async function refresh() {
    const list = await listDeveloperTools()
    setTools(list)
  }

  useEffect(() => {
    if (!ffContentToolMarketplace) return
    void refresh().catch((e: unknown) =>
      setError(e instanceof Error ? e.message : 'Failed to load'),
    )
  }, [ffContentToolMarketplace])

  if (!ffContentToolMarketplace) {
    return (
      <div className="mx-auto max-w-3xl p-6">
        <p className="text-sm">{t('contentTools.developer.disabled')}</p>
      </div>
    )
  }

  return (
    <div className="mx-auto max-w-3xl p-6" data-testid="developer-tools-page">
      <h1 id={titleId} className="text-2xl font-semibold">
        {t('contentTools.developer.title')}
      </h1>
      <p className="mt-1 text-sm text-slate-600">{t('contentTools.developer.help')}</p>
      {error ? (
        <p className="mt-4 text-sm text-rose-700" role="alert">
          {error}
        </p>
      ) : null}
      <form
        className="mt-6 space-y-3"
        onSubmit={(e) => {
          e.preventDefault()
          setBusy(true)
          setError(null)
          void createDeveloperTool({
            toolId,
            displayName,
            summary,
            visibility: 'unlisted',
          })
            .then(() => refresh())
            .catch((err: unknown) => setError(err instanceof Error ? err.message : 'Create failed'))
            .finally(() => setBusy(false))
        }}
      >
        <label className="block text-sm">
          {t('contentTools.developer.toolId')}
          <input
            className="mt-1 w-full rounded border px-3 py-2"
            data-testid="developer-tool-id"
            value={toolId}
            onChange={(e) => setToolId(e.target.value)}
          />
        </label>
        <label className="block text-sm">
          {t('contentTools.developer.displayName')}
          <input
            className="mt-1 w-full rounded border px-3 py-2"
            data-testid="developer-tool-name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </label>
        <label className="block text-sm">
          {t('contentTools.developer.summary')}
          <input
            className="mt-1 w-full rounded border px-3 py-2"
            data-testid="developer-tool-summary"
            value={summary}
            onChange={(e) => setSummary(e.target.value)}
          />
        </label>
        <button
          type="submit"
          className="rounded bg-slate-900 px-4 py-2 text-sm text-white disabled:opacity-50"
          data-testid="developer-tool-create"
          disabled={busy}
        >
          {t('contentTools.developer.create')}
        </button>
      </form>
      <ul className="mt-8 space-y-3" aria-labelledby={titleId}>
        {tools.map((tool) => (
          <li key={tool.id} className="border-b pb-3" data-testid={`developer-tool-${tool.toolId}`}>
            <div className="font-medium">{tool.displayName}</div>
            <div className="text-xs text-slate-500">
              {tool.toolId} · {tool.status} · {tool.visibility}
            </div>
            <p className="text-sm text-slate-600">{tool.summary}</p>
          </li>
        ))}
      </ul>
    </div>
  )
}
