import { useRef, type ReactNode } from 'react'
import { Dialog } from './dialog'
import { Button } from './button'
import { Input } from './input'

export type AlertDialogProps = {
  open: boolean
  onClose: () => void
  onConfirm: () => void
  title: ReactNode
  description?: ReactNode
  confirmLabel: string
  cancelLabel: string
  variant?: 'default' | 'danger'
  busy?: boolean
  confirmDisabled?: boolean
  /** When set, confirm is disabled until the typed phrase matches (trimmed). */
  requireTypedPhrase?: string
  typedPhrase?: string
  onTypedPhraseChange?: (value: string) => void
  phraseFieldLabel?: string
  /** Accessible name for the scrim (must differ from cancelLabel when both are visible to AT). */
  dismissLabel?: string
  onExited?: () => void
}

/**
 * Modal confirmation dialog (AlertDialog APG). Uses Dialog for focus trap/inert.
 */
export function AlertDialog({
  open,
  onClose,
  onConfirm,
  title,
  description,
  confirmLabel,
  cancelLabel,
  variant = 'default',
  busy,
  confirmDisabled,
  requireTypedPhrase,
  typedPhrase = '',
  onTypedPhraseChange,
  phraseFieldLabel,
  dismissLabel,
  onExited,
}: AlertDialogProps) {
  const cancelRef = useRef<HTMLButtonElement>(null)
  const phraseOk =
    requireTypedPhrase == null || typedPhrase.trim() === requireTypedPhrase.trim()
  const disableConfirm = Boolean(busy || confirmDisabled || !phraseOk)
  // Scrim must not share the same accessible name as the Cancel button.
  const scrimLabel = dismissLabel ?? `${cancelLabel} dialog`

  return (
    <Dialog
      open={open}
      onClose={() => {
        if (!busy) onClose()
      }}
      title={title}
      description={description}
      hideClose
      closeOnBackdrop={!busy}
      closeOnEscape={!busy}
      closeLabel={cancelLabel}
      backdropLabel={scrimLabel}
      onExited={onExited}
      initialFocusRef={cancelRef}
      rootTestId="confirm-dialog-root"
      footer={
        <>
          <Button ref={cancelRef} variant="secondary" disabled={busy} onClick={onClose}>
            {cancelLabel}
          </Button>
          <Button
            variant={variant === 'danger' ? 'danger' : 'primary'}
            disabled={disableConfirm}
            loading={busy}
            onClick={() => {
              if (!disableConfirm) onConfirm()
            }}
          >
            {confirmLabel}
          </Button>
        </>
      }
    >
      {requireTypedPhrase != null ? (
        <div>
          {/* Stable id for e2e / product tests (historical ConfirmDialog). */}
          <label htmlFor="confirm-dialog-phrase" className="text-xs font-medium text-fg-default">
            {phraseFieldLabel ?? requireTypedPhrase}
          </label>
          <Input
            id="confirm-dialog-phrase"
            key={requireTypedPhrase}
            autoComplete="off"
            value={typedPhrase}
            onChange={(e) => onTypedPhraseChange?.(e.target.value)}
            disabled={busy}
            className="mt-1.5"
          />
        </div>
      ) : null}
    </Dialog>
  )
}
