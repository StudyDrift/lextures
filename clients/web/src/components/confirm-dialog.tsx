import { type ReactNode } from 'react'
import { AlertDialog } from './ui/alert-dialog'

export type ConfirmDialogProps = {
  open: boolean
  title: string
  description?: ReactNode
  confirmLabel?: string
  cancelLabel?: string
  /** When `'danger'`, confirm button uses destructive styling. */
  variant?: 'default' | 'danger'
  /** If set, user must type this exact string (after trim) to enable Confirm. */
  requireTypedPhrase?: string
  /** Current value of the confirmation phrase field (controlled). */
  typedPhrase?: string
  onTypedPhraseChange?: (value: string) => void
  confirmDisabled?: boolean
  busy?: boolean
  onConfirm: () => void
  onClose: () => void
  /** Fired when exit animation finishes and the dialog unmounts. */
  onExited?: () => void
}

/**
 * Product confirm dialog — thin wrapper over UX.2 {@link AlertDialog}.
 * Keeps the historical prop surface for existing `useConfirm` call sites.
 */
export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'default',
  requireTypedPhrase,
  typedPhrase = '',
  onTypedPhraseChange,
  confirmDisabled,
  busy,
  onConfirm,
  onClose,
  onExited,
}: ConfirmDialogProps) {
  return (
    <AlertDialog
      open={open}
      title={title}
      description={description}
      confirmLabel={confirmLabel}
      cancelLabel={cancelLabel}
      variant={variant}
      requireTypedPhrase={requireTypedPhrase}
      typedPhrase={typedPhrase}
      onTypedPhraseChange={onTypedPhraseChange}
      confirmDisabled={confirmDisabled}
      busy={busy}
      onConfirm={onConfirm}
      onClose={onClose}
      onExited={onExited}
    />
  )
}
