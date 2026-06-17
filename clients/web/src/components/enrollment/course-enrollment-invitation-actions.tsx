import { useCallback, useState } from 'react'
import {
  approveEnrollmentInvitation,
  declineEnrollmentInvitation,
} from '../../lib/enrollment-invitation-api'
import { toast, toastMutationError } from '../../lib/lms-toast'

type CourseEnrollmentInvitationActionsProps = {
  courseCode: string
  enrollmentId: string
  compact?: boolean
  onResolved?: (approved: boolean) => void
}

export function CourseEnrollmentInvitationActions({
  courseCode,
  enrollmentId,
  compact = false,
  onResolved,
}: CourseEnrollmentInvitationActionsProps) {
  const [busy, setBusy] = useState<'approve' | 'decline' | null>(null)

  const handleApprove = useCallback(async () => {
    if (busy) return
    setBusy('approve')
    try {
      await approveEnrollmentInvitation(courseCode, enrollmentId)
      toast.success('Enrollment approved. You can now access this course.')
      onResolved?.(true)
    } catch {
      toastMutationError('Could not approve the invitation. Try again.')
    } finally {
      setBusy(null)
    }
  }, [busy, courseCode, enrollmentId, onResolved])

  const handleDecline = useCallback(async () => {
    if (busy) return
    setBusy('decline')
    try {
      await declineEnrollmentInvitation(courseCode, enrollmentId)
      toast.success('Invitation declined.')
      onResolved?.(false)
    } catch {
      toastMutationError('Could not decline the invitation. Try again.')
    } finally {
      setBusy(null)
    }
  }, [busy, courseCode, enrollmentId, onResolved])

  const btnClass = compact
    ? 'rounded-lg px-3 py-1.5 text-xs font-semibold'
    : 'rounded-xl px-4 py-2 text-sm font-semibold'

  return (
    <div className={`flex flex-wrap items-center gap-2 ${compact ? '' : 'mt-3'}`}>
      <button
        type="button"
        disabled={busy !== null}
        onClick={() => void handleApprove()}
        className={`${btnClass} bg-emerald-600 text-white shadow-sm hover:bg-emerald-500 disabled:opacity-60`}
      >
        {busy === 'approve' ? 'Approving…' : 'Approve'}
      </button>
      <button
        type="button"
        disabled={busy !== null}
        onClick={() => void handleDecline()}
        className={`${btnClass} bg-red-600 text-white shadow-sm hover:bg-red-500 disabled:opacity-60`}
      >
        {busy === 'decline' ? 'Declining…' : 'Decline'}
      </button>
    </div>
  )
}