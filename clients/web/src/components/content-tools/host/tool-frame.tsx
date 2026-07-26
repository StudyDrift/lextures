import type { ReactNode } from 'react'
import type { ToolSyncStatus } from './use-tool-state'

export type ToolFrameProps = {
  label: string
  status: string
  syncStatus: ToolSyncStatus
  savedLabel: string
  savingLabel: string
  unsyncedLabel: string
  children: ReactNode
  onBlurCapture?: () => void
  busy?: boolean
}

function statusChipClass(status: string): string {
  switch (status) {
    case 'completed':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-950/50 dark:text-emerald-200'
    case 'submitted':
      return 'bg-sky-100 text-sky-800 dark:bg-sky-950/50 dark:text-sky-200'
    case 'in_progress':
      return 'bg-amber-100 text-amber-900 dark:bg-amber-950/40 dark:text-amber-200'
    default:
      return 'bg-slate-100 text-slate-700 dark:bg-neutral-800 dark:text-neutral-300'
  }
}

export function ToolFrame({
  label,
  status,
  syncStatus,
  savedLabel,
  savingLabel,
  unsyncedLabel,
  children,
  onBlurCapture,
  busy = false,
}: ToolFrameProps) {
  const syncLabel =
    syncStatus === 'saving'
      ? savingLabel
      : syncStatus === 'unsynced'
        ? unsyncedLabel
        : syncStatus === 'saved'
          ? savedLabel
          : null

  return (
    <div
      role="group"
      aria-label={label}
      aria-busy={busy || syncStatus === 'saving' ? true : undefined}
      data-content-tool-frame=""
      onBlurCapture={onBlurCapture}
      className="not-prose my-4 rounded-md border border-slate-200 bg-slate-50/80 dark:border-neutral-700 dark:bg-neutral-900/50"
    >
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-slate-200/80 px-3 py-2 dark:border-neutral-700">
        <p className="truncate text-sm font-semibold text-slate-900 dark:text-neutral-100">{label}</p>
        <div className="flex flex-wrap items-center gap-1.5">
          <span
            className={`rounded px-1.5 py-0.5 text-[11px] font-medium uppercase tracking-wide ${statusChipClass(status)}`}
          >
            {status.replace(/_/g, ' ')}
          </span>
          {syncLabel ? (
            <span
              data-sync-status={syncStatus}
              className={
                syncStatus === 'unsynced'
                  ? 'rounded px-1.5 py-0.5 text-[11px] font-medium text-amber-800 dark:text-amber-200'
                  : 'rounded px-1.5 py-0.5 text-[11px] font-medium text-slate-500 dark:text-neutral-400'
              }
            >
              {syncLabel}
            </span>
          ) : null}
        </div>
      </div>
      <div className="px-3 py-3">{children}</div>
    </div>
  )
}
