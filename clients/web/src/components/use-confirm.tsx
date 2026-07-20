import { useCallback, useRef, useState, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { ConfirmDialog } from './confirm-dialog'

export type ConfirmOptions = {
  title: string
  description?: ReactNode
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'default' | 'danger'
  requireTypedPhrase?: string
}

type PendingConfirm = ConfirmOptions & {
  resolve: (confirmed: boolean) => void
}

/** Promise-based wrapper around {@link ConfirmDialog} for mechanical confirm() migration. */
export function useConfirm() {
  const { t } = useTranslation('common')
  const [pending, setPending] = useState<PendingConfirm | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [typedPhrase, setTypedPhrase] = useState('')
  const [busy, setBusy] = useState(false)
  const triggerRef = useRef<HTMLElement | null>(null)
  const resolvedRef = useRef(false)
  const dialogOpenRef = useRef(false)
  dialogOpenRef.current = dialogOpen

  const confirm = useCallback((options: ConfirmOptions): Promise<boolean> => {
    triggerRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    resolvedRef.current = false
    return new Promise((resolve) => {
      setTypedPhrase('')
      setBusy(false)
      setPending({ ...options, resolve })
      setDialogOpen(true)
    })
  }, [])

  const returnFocus = useCallback(() => {
    const el = triggerRef.current
    triggerRef.current = null
    queueMicrotask(() => el?.focus())
  }, [])

  const close = useCallback(
    (result: boolean) => {
      if (busy) return
      if (!resolvedRef.current) {
        resolvedRef.current = true
        pending?.resolve(result)
      }
      // Focus returns on exit-start (not exit-end) so a11y is not delayed by motion.
      returnFocus()
      setDialogOpen(false)
      setTypedPhrase('')
    },
    [busy, pending, returnFocus],
  )

  const handleExited = useCallback(() => {
    // Ignore exit completion if a new confirm already re-opened (AC-6).
    if (dialogOpenRef.current) return
    setPending(null)
    setBusy(false)
  }, [])

  const ConfirmDialogHost = pending ? (
    <ConfirmDialog
      open={dialogOpen}
      title={pending.title}
      description={pending.description}
      confirmLabel={pending.confirmLabel ?? t('dialogs.confirm')}
      cancelLabel={pending.cancelLabel ?? t('dialogs.cancel')}
      variant={pending.variant}
      requireTypedPhrase={pending.requireTypedPhrase}
      typedPhrase={typedPhrase}
      onTypedPhraseChange={setTypedPhrase}
      busy={busy}
      onConfirm={() => close(true)}
      onClose={() => close(false)}
      onExited={handleExited}
    />
  ) : null

  return { confirm, ConfirmDialogHost, setConfirmBusy: setBusy }
}
