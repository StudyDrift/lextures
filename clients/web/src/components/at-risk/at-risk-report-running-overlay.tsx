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
      <div className="flex flex-col items-center gap-5 rounded-2xl border border-border-default bg-surface-raised px-10 py-12 shadow-2xl dark:border-border-default dark:bg-surface-raised">
        <div className="inline-flex origin-center scale-[0.55] sm:scale-[0.65]">
          <BookLoader />
        </div>
        <p className="text-center text-sm font-semibold text-fg-default">
          {atRiskI18n.reportRunning}
        </p>
      </div>
    </div>
  )
}
