import { BookLoader } from '../quiz/book-loader'
import { atRiskI18n } from '../../lib/at-risk-i18n'

type AtRiskReportRunningOverlayProps = {
  open: boolean
}

/** Full-screen overlay with the book loader while an at-risk report is running. */
export function AtRiskReportRunningOverlay({ open }: AtRiskReportRunningOverlayProps) {
  if (!open) return null

  return (
    <div
      className="fixed inset-0 z-[500] flex items-center justify-center bg-black/40 p-4"
      role="status"
      aria-live="polite"
      aria-busy="true"
      aria-label={atRiskI18n.reportRunning}
    >
      <div className="flex flex-col items-center gap-5 rounded-2xl border border-slate-200 bg-white px-10 py-12 shadow-2xl dark:border-neutral-700 dark:bg-neutral-900">
        <div className="inline-flex origin-center scale-[0.55] sm:scale-[0.65]">
          <BookLoader />
        </div>
        <p className="text-center text-sm font-semibold text-slate-800 dark:text-neutral-100">
          {atRiskI18n.reportRunning}
        </p>
      </div>
    </div>
  )
}
