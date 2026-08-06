import { useEffect, useState } from 'react'
import type { ContentToolRendererProps } from '../../host/runtime-contract'

export default function NoopProbeRenderer({
  config,
  state,
  readOnly,
  save,
  runAction,
  t,
  announce,
}: ContentToolRendererProps) {
  const prompt =
    typeof config.prompt === 'string' && config.prompt.trim()
      ? config.prompt
      : t('contentTools.authoring.previewNoPrompt')
  const remoteResponse = typeof state.response === 'string' ? state.response : ''
  const [response, setResponse] = useState(remoteResponse)
  const [result, setResult] = useState<{ correct?: boolean; reason?: string } | null>(null)
  const [checking, setChecking] = useState(false)

  useEffect(() => {
    setResponse(remoteResponse)
  }, [remoteResponse])

  async function onCheckAnswer() {
    setChecking(true)
    setResult(null)
    try {
      const raw = await runAction('grade', { response })
      const next =
        raw && typeof raw === 'object'
          ? (raw as { correct?: boolean; reason?: string })
          : null
      setResult(next)
      if (next?.correct === true) {
        announce(t('contentTools.runtime.score'))
      }
    } catch {
      setResult({ correct: false, reason: 'error' })
    } finally {
      setChecking(false)
    }
  }

  return (
    <div className="space-y-3" data-content-tool="noop_probe">
      <p className="text-sm text-fg-default">{prompt}</p>
      <label className="block space-y-1">
        <span className="text-xs font-medium text-fg-muted">
          {t('contentTools.runtime.yourAnswer')}
        </span>
        <textarea
          value={response}
          disabled={readOnly || checking}
          rows={3}
          onChange={(e) => {
            const value = e.target.value
            setResponse(value)
            void save({ response: value })
          }}
          className="w-full rounded-md border border-border-default bg-surface-raised px-2.5 py-1.5 text-sm text-fg-default dark:border-border-default dark:bg-surface-base dark:text-fg-default"
        />
      </label>
      <div className="flex flex-wrap items-center gap-2">
        <button
          type="button"
          disabled={readOnly || checking || !response.trim()}
          onClick={() => void onCheckAnswer()}
          className="rounded-md bg-slate-800 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700 disabled:opacity-50 dark:bg-neutral-200 dark:text-neutral-900 dark:hover:bg-neutral-100"
        >
          {t('contentTools.runtime.checkAnswer')}
        </button>
        {result && typeof result.correct === 'boolean' ? (
          <span
            className={
              result.correct
                ? 'text-xs font-medium text-emerald-700 dark:text-emerald-300'
                : 'text-xs font-medium text-rose-700 dark:text-rose-300'
            }
            role="status"
            data-correct={result.correct ? 'true' : 'false'}
          >
            {result.correct ? t('contentTools.runtime.score') : t('contentTools.runtime.retry')}
          </span>
        ) : null}
      </div>
    </div>
  )
}
