import { ShieldAlert } from 'lucide-react'

export type AcademicIntegrityNoticeProps = {
  className?: string
}

/** Plan 14.8 — student-visible notice when originality checks are enabled on an assignment. */
export function AcademicIntegrityNotice({ className = '' }: AcademicIntegrityNoticeProps) {
  return (
    <div
      role="note"
      aria-label="Academic integrity notice"
      className={`flex gap-3 rounded-xl border border-amber-200 bg-amber-50/90 px-4 py-3 text-sm text-amber-950 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-50 ${className}`}
    >
      <ShieldAlert className="mt-0.5 h-5 w-5 shrink-0 opacity-80" aria-hidden />
      <p>
        Submissions to this assignment are checked for plagiarism and AI-generated content. Reports are
        advisory and reviewed by your instructor. Use proper citations and submit your own work.
      </p>
    </div>
  )
}
