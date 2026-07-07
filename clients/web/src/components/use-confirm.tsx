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
  const [typedPhrase, setTypedPhrase] = useState('')
  const [busy, setBusy] = useState(false)
  const triggerRef = useRef<HTMLElement | null>(null)

  const confirm = useCallback((options: ConfirmOptions): Promise<boolean> => {
    triggerRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    return new Promise((resolve) => {
      setTypedPhrase('')
      setBusy(false)
      setPending({ ...options, resolve })
    })
  }, [])

  const close = useCallback(
    (result: boolean) => {
      if (busy) return
      pending?.resolve(result)
      setPending(null)
      setTypedPhrase('')
      const el = triggerRef.current
      triggerRef.current = null
      queueMicrotask(() => el?.focus())
    },
    [busy, pending],
  )

  const ConfirmDialogHost = pending ? (
    <ConfirmDialog
      open
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
    />
  ) : null

  return { confirm, ConfirmDialogHost, setConfirmBusy: setBusy }
}
