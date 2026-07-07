import { useCallback, useRef, useState, type ReactNode } from 'react'
import { InputDialog } from './input-dialog'

export type PromptOptions = {
  title: string
  description?: ReactNode
  label?: string
  defaultValue?: string
  placeholder?: string
  confirmLabel?: string
  cancelLabel?: string
}

type PendingPrompt = PromptOptions & {
  resolve: (value: string | null) => void
}

/** Promise-based wrapper around {@link InputDialog} for mechanical prompt() migration. */
export function usePrompt() {
  const [pending, setPending] = useState<PendingPrompt | null>(null)
  const [value, setValue] = useState('')
  const [busy, setBusy] = useState(false)
  const triggerRef = useRef<HTMLElement | null>(null)

  const prompt = useCallback((options: PromptOptions): Promise<string | null> => {
    triggerRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null
    return new Promise((resolve) => {
      setValue(options.defaultValue ?? '')
      setBusy(false)
      setPending({ ...options, resolve })
    })
  }, [])

  const close = useCallback(
    (result: string | null) => {
      if (busy) return
      pending?.resolve(result)
      setPending(null)
      const el = triggerRef.current
      triggerRef.current = null
      queueMicrotask(() => el?.focus())
    },
    [busy, pending],
  )

  const InputDialogHost = pending ? (
    <InputDialog
      open
      title={pending.title}
      description={pending.description}
      label={pending.label}
      value={value}
      onValueChange={setValue}
      placeholder={pending.placeholder}
      confirmLabel={pending.confirmLabel}
      cancelLabel={pending.cancelLabel}
      busy={busy}
      onConfirm={(next) => close(next)}
      onClose={() => close(null)}
    />
  ) : null

  return { prompt, InputDialogHost, setPromptBusy: setBusy }
}
