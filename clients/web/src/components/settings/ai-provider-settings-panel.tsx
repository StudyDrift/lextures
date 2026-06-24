import { useCallback, useEffect, useState } from 'react'
import { authorizedFetch } from '../../lib/api'
import { readApiErrorMessage } from '../../lib/errors'
import { PLATFORM_SECRET_PLACEHOLDER } from '../../lib/platform-settings'
import { toastMutationError, toastSaveOk } from '../../lib/lms-toast'

type AIProviderSettings = {
  provider?: string
  modelAlias?: string
  fallbackProvider?: string | null
  byokConfigured?: boolean
  settings?: Record<string, unknown>
  providers?: string[]
  modelAliases?: string[]
}

const PROVIDER_LABELS: Record<string, string> = {
  openrouter: 'OpenRouter',
  anthropic: 'Anthropic',
  openai: 'OpenAI',
  azure_openai: 'Azure OpenAI',
  bedrock: 'AWS Bedrock',
  vertex: 'Google Vertex AI',
}

export function AiProviderSettingsPanel() {
  const [data, setData] = useState<AIProviderSettings | null>(null)
  const [provider, setProvider] = useState('openrouter')
  const [modelAlias, setModelAlias] = useState('claude-3-5-sonnet')
  const [fallbackProvider, setFallbackProvider] = useState('')
  const [byokKey, setByokKey] = useState('')
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)

  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const res = await authorizedFetch('/api/v1/admin/ai-settings')
        if (res.status === 404 || res.status === 403) {
          return
        }
        if (!res.ok) {
          throw new Error(await readApiErrorMessage(res))
        }
        const json = (await res.json()) as AIProviderSettings
        if (!cancelled) {
          setData(json)
          setProvider(json.provider ?? 'openrouter')
          setModelAlias(json.modelAlias ?? 'claude-3-5-sonnet')
          setFallbackProvider(json.fallbackProvider ?? '')
          setByokKey(json.byokConfigured ? PLATFORM_SECRET_PLACEHOLDER : '')
        }
      } catch {
        /* not org admin or feature disabled */
      } finally {
        if (!cancelled) setLoading(false)
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const save = useCallback(async () => {
    setSaving(true)
    try {
      const payload: Record<string, unknown> = {
        provider,
        modelAlias,
        fallbackProvider: fallbackProvider.trim() || null,
      }
      if (byokKey && byokKey !== PLATFORM_SECRET_PLACEHOLDER) {
        payload.byokApiKey = byokKey
      }
      const res = await authorizedFetch('/api/v1/admin/ai-settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })
      if (!res.ok) {
        throw new Error(await readApiErrorMessage(res))
      }
      const json = (await res.json()) as AIProviderSettings
      setData((prev) => ({ ...prev, ...json }))
      if (json.byokConfigured) {
        setByokKey(PLATFORM_SECRET_PLACEHOLDER)
      }
      toastSaveOk('AI provider settings saved.')
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Could not save.')
    } finally {
      setSaving(false)
    }
  }, [byokKey, fallbackProvider, modelAlias, provider])

  const testConnection = useCallback(async () => {
    setTesting(true)
    try {
      const res = await authorizedFetch('/api/v1/admin/ai-settings/test', { method: 'POST' })
      if (!res.ok) {
        throw new Error(await readApiErrorMessage(res))
      }
      const json = (await res.json()) as {
        provider?: string
        latencyMs?: number
        responsePreview?: string
      }
      toastSaveOk(
        `Connected via ${json.provider ?? provider} (${json.latencyMs ?? '?'} ms): ${json.responsePreview ?? 'OK'}`,
      )
    } catch (e) {
      toastMutationError(e instanceof Error ? e.message : 'Test connection failed.')
    } finally {
      setTesting(false)
    }
  }, [provider])

  if (loading || !data) {
    return null
  }

  const providers = data.providers ?? Object.keys(PROVIDER_LABELS)
  const aliases = data.modelAliases ?? ['claude-3-5-sonnet', 'gpt-4o', 'gemini-1.5-pro']

  return (
    <section className="mt-8 rounded-xl border border-slate-200 bg-white p-5 dark:border-neutral-600 dark:bg-neutral-900">
      <h3 className="text-sm font-semibold text-slate-900 dark:text-neutral-100">AI provider</h3>
      <p className="mt-1 text-sm text-slate-500 dark:text-neutral-400">
        Choose which AI backend serves your organization. BYOK keys are stored encrypted and never returned after save.
      </p>
      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <label className="block text-sm">
          <span className="font-medium text-slate-700 dark:text-neutral-300">Provider</span>
          <select
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            value={provider}
            onChange={(e) => setProvider(e.target.value)}
          >
            {providers.map((p) => (
              <option key={p} value={p}>
                {PROVIDER_LABELS[p] ?? p}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="font-medium text-slate-700 dark:text-neutral-300">Model alias</span>
          <select
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            value={modelAlias}
            onChange={(e) => setModelAlias(e.target.value)}
          >
            {aliases.map((a) => (
              <option key={a} value={a}>
                {a}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm sm:col-span-2">
          <span className="font-medium text-slate-700 dark:text-neutral-300">Fallback provider</span>
          <select
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            value={fallbackProvider}
            onChange={(e) => setFallbackProvider(e.target.value)}
          >
            <option value="">None</option>
            {providers.map((p) => (
              <option key={p} value={p}>
                {PROVIDER_LABELS[p] ?? p}
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm sm:col-span-2">
          <span className="font-medium text-slate-700 dark:text-neutral-300">BYOK API key</span>
          <input
            type="password"
            className="mt-1 w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm dark:border-neutral-600 dark:bg-neutral-800"
            value={byokKey}
            placeholder={PLATFORM_SECRET_PLACEHOLDER}
            onChange={(e) => setByokKey(e.target.value)}
            autoComplete="off"
          />
          {data.byokConfigured ? (
            <span className="mt-1 block text-xs text-emerald-600 dark:text-emerald-400">Configured</span>
          ) : null}
        </label>
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        <button
          type="button"
          className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-50 dark:bg-neutral-100 dark:text-neutral-900"
          disabled={saving}
          onClick={() => void save()}
        >
          {saving ? 'Saving…' : 'Save'}
        </button>
        <button
          type="button"
          className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 disabled:opacity-50 dark:border-neutral-600 dark:text-neutral-200"
          disabled={testing}
          onClick={() => void testConnection()}
        >
          {testing ? 'Testing…' : 'Test connection'}
        </button>
      </div>
    </section>
  )
}