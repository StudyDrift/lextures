import { Copy, Eye, Settings2, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { ContentToolInstance, ContentToolsCatalogTool } from '../../../lib/courses-api'

export type ToolBlockCardProps = {
  instanceId: string
  toolId: string
  instance?: ContentToolInstance
  catalogTool?: ContentToolsCatalogTool
  onConfigure?: () => void
  onPreview?: () => void
  onDuplicate?: () => void
  onDelete?: () => void
}

function configSummary(config: Record<string, unknown> | undefined): string {
  if (!config) return ''
  const prompt = config.prompt
  if (typeof prompt === 'string' && prompt.trim()) {
    return prompt.trim().length > 80 ? `${prompt.trim().slice(0, 77)}…` : prompt.trim()
  }
  const keys = Object.keys(config)
  if (keys.length === 0) return ''
  return keys.slice(0, 3).join(', ')
}

export function ToolBlockCard({
  instanceId,
  toolId,
  instance,
  catalogTool,
  onConfigure,
  onPreview,
  onDuplicate,
  onDelete,
}: ToolBlockCardProps) {
  const { t } = useTranslation('contentTools')
  const title = t(`contentTools.tools.${toolId}.name`, {
    defaultValue: catalogTool?.name || instance?.title || toolId || t('contentTools.authoring.unavailableTool'),
  })
  const summary = configSummary(instance?.config)
  const unavailable = !toolId

  return (
    <div
      role="group"
      aria-label={t('contentTools.authoring.toolBlockAria', { name: title })}
      data-content-tool-instance={instanceId}
      className="rounded-md border border-slate-200 bg-slate-50/80 dark:border-neutral-700 dark:bg-neutral-900/50"
    >
      <div className="flex items-start justify-between gap-2 px-3 py-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">
            {unavailable ? t('contentTools.authoring.unavailableTool') : title}
          </p>
          {summary ? (
            <p className="mt-0.5 line-clamp-2 text-xs text-slate-500 dark:text-neutral-400">{summary}</p>
          ) : (
            <p className="mt-0.5 text-xs text-slate-500 dark:text-neutral-400">
              {t('contentTools.authoring.noConfigSummary')}
            </p>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-0.5">
          <button
            type="button"
            onClick={onConfigure}
            className="rounded p-1.5 text-slate-600 hover:bg-slate-200 dark:text-neutral-300 dark:hover:bg-neutral-700"
            aria-label={t('contentTools.authoring.configure')}
            title={t('contentTools.authoring.configure')}
          >
            <Settings2 className="h-3.5 w-3.5" aria-hidden />
          </button>
          <button
            type="button"
            onClick={onPreview}
            className="rounded p-1.5 text-slate-600 hover:bg-slate-200 dark:text-neutral-300 dark:hover:bg-neutral-700"
            aria-label={t('contentTools.authoring.preview')}
            title={t('contentTools.authoring.preview')}
          >
            <Eye className="h-3.5 w-3.5" aria-hidden />
          </button>
          <button
            type="button"
            onClick={onDuplicate}
            className="rounded p-1.5 text-slate-600 hover:bg-slate-200 dark:text-neutral-300 dark:hover:bg-neutral-700"
            aria-label={t('contentTools.authoring.duplicate')}
            title={t('contentTools.authoring.duplicate')}
          >
            <Copy className="h-3.5 w-3.5" aria-hidden />
          </button>
          <button
            type="button"
            onClick={onDelete}
            className="rounded p-1.5 text-rose-600 hover:bg-rose-50 dark:text-rose-400 dark:hover:bg-rose-950/40"
            aria-label={t('contentTools.authoring.delete')}
            title={t('contentTools.authoring.delete')}
          >
            <Trash2 className="h-3.5 w-3.5" aria-hidden />
          </button>
        </div>
      </div>
    </div>
  )
}
