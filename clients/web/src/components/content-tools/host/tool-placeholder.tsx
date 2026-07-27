export type ToolPlaceholderReason =
  | 'unavailable'
  | 'disabled'
  | 'archived'
  | 'openInBrowser'
  | 'readOnlyArchived'
  | 'readOnlyPastDue'
  | 'readOnlyPreview'
  | 'loading'
  | 'error'
  | 'maintenance'
  | 'recovery'
  | 'updateRequired'

export type ToolPlaceholderProps = {
  reason: ToolPlaceholderReason
  message: string
  onRetry?: () => void
  retryLabel?: string
}

export function ToolPlaceholder({ reason, message, onRetry, retryLabel }: ToolPlaceholderProps) {
  return (
    <div
      role="status"
      data-content-tool-placeholder={reason}
      className="rounded-md border border-slate-200 bg-slate-50/80 px-3 py-3 text-sm text-slate-600 dark:border-neutral-700 dark:bg-neutral-900/50 dark:text-neutral-300"
    >
      <p>{message}</p>
      {onRetry && retryLabel ? (
        <button
          type="button"
          onClick={onRetry}
          className="mt-2 rounded-md bg-slate-800 px-3 py-1.5 text-xs font-medium text-white hover:bg-slate-700 dark:bg-neutral-200 dark:text-neutral-900"
        >
          {retryLabel}
        </button>
      ) : null}
    </div>
  )
}
