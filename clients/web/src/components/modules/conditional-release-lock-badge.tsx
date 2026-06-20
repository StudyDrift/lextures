import { Lock } from 'lucide-react'
import type { LockReason } from '../../lib/conditional-release-api'

export function ConditionalReleaseLockBadge({ reason }: { reason?: LockReason | null }) {
  if (!reason) {
    return (
      <span
        className="inline-flex shrink-0 items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-800 dark:bg-neutral-800 dark:text-neutral-100"
        aria-label="Locked"
      >
        <Lock className="h-3 w-3" strokeWidth={2} aria-hidden />
        Locked
      </span>
    )
  }
  return (
    <span
      className="inline-flex shrink-0 items-center gap-1 rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-semibold text-slate-800 dark:bg-neutral-800 dark:text-neutral-100"
      aria-label={`Locked — ${reason.message}`}
      title={reason.message}
    >
      <Lock className="h-3 w-3" strokeWidth={2} aria-hidden />
      {reason.message}
    </span>
  )
}
