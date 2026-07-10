import {
  Archive,
  CheckCircle2,
  CircleDot,
  Loader2,
  MinusCircle,
  Sparkles,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { FeedbackStatus } from '../../lib/feedback-admin-api'

const STATUS_STYLES: Record<
  FeedbackStatus,
  { className: string; Icon: typeof CircleDot }
> = {
  new: {
    className:
      'border-indigo-200 bg-indigo-50 text-indigo-800 dark:border-indigo-800/60 dark:bg-indigo-950/50 dark:text-indigo-200',
    Icon: Sparkles,
  },
  triaged: {
    className:
      'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-800/60 dark:bg-amber-950/40 dark:text-amber-200',
    Icon: CircleDot,
  },
  in_progress: {
    className:
      'border-blue-200 bg-blue-50 text-blue-900 dark:border-blue-800/60 dark:bg-blue-950/40 dark:text-blue-200',
    Icon: Loader2,
  },
  resolved: {
    className:
      'border-emerald-200 bg-emerald-50 text-emerald-900 dark:border-emerald-800/60 dark:bg-emerald-950/40 dark:text-emerald-200',
    Icon: CheckCircle2,
  },
  wont_fix: {
    className:
      'border-slate-300 bg-slate-100 text-slate-700 dark:border-neutral-600 dark:bg-neutral-800 dark:text-neutral-300',
    Icon: MinusCircle,
  },
  archived: {
    className:
      'border-slate-200 bg-slate-50 text-slate-600 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-400',
    Icon: Archive,
  },
}

export function FeedbackAdminStatusBadge({ status }: { status: FeedbackStatus }) {
  const { t } = useTranslation('common')
  const style = STATUS_STYLES[status]
  const Icon = style.Icon
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${style.className}`}
    >
      <Icon className="h-3 w-3 shrink-0" aria-hidden />
      {t(`settings.feedback.status.${status}`)}
    </span>
  )
}
