import { useTranslation } from 'react-i18next'
import type { NodeDryRunDetail } from './use-grader-agent-workflow'

type AiNodeCompiledPromptProps = {
  detail: NodeDryRunDetail
}

function CompiledBlock({ label, value }: { label: string; value: string }) {
  if (!value.trim()) return null
  return (
    <div>
      <p className="mb-1.5 text-xs font-semibold uppercase tracking-wide text-fg-muted">
        {label}
      </p>
      <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-lg border border-border-default bg-surface-base px-3 py-2 font-mono text-xs leading-relaxed text-fg-default dark:border-border-default dark:bg-surface-base dark:text-fg-default">
        {value}
      </pre>
    </div>
  )
}

export function AiNodeCompiledPrompt({ detail }: AiNodeCompiledPromptProps) {
  const { t } = useTranslation('common')
  const hasContent =
    Boolean(detail.compiledPrompt?.trim()) ||
    Boolean(detail.compiledSystemPrompt?.trim()) ||
    Boolean(detail.compiledInput?.trim()) ||
    Boolean(detail.compiledOutput?.trim())

  if (!hasContent) return null

  return (
    <div className="space-y-3 border-t border-border-default pt-3 dark:border-border-default">
      <p className="text-sm font-medium text-fg-default">
        {t('gradingAgent.canvas.inspector.aiCompiled.title')}
      </p>
      <p className="text-xs text-fg-muted">
        {t('gradingAgent.canvas.inspector.aiCompiled.help')}
      </p>
      <CompiledBlock
        label={t('gradingAgent.canvas.inspector.aiCompiled.systemPrompt')}
        value={detail.compiledSystemPrompt ?? ''}
      />
      <CompiledBlock label={t('gradingAgent.canvas.inspector.aiCompiled.prompt')} value={detail.compiledPrompt ?? ''} />
      <CompiledBlock label={t('gradingAgent.canvas.inspector.aiCompiled.input')} value={detail.compiledInput ?? ''} />
      <CompiledBlock label={t('gradingAgent.canvas.inspector.aiCompiled.output')} value={detail.compiledOutput ?? ''} />
    </div>
  )
}