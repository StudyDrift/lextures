import { useRef, useState, type RefObject } from 'react'
import { MessageSquarePlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { FeedbackDialog } from './feedback-dialog'

function FeedbackWidgetTrigger({
  open,
  label,
  onOpen,
  triggerRef,
}: {
  open: boolean
  label: string
  onOpen: () => void
  triggerRef: RefObject<HTMLButtonElement | null>
}) {
  return (
    <button
      ref={triggerRef}
      type="button"
      aria-label={label}
      aria-expanded={open}
      aria-haspopup="dialog"
      onClick={onOpen}
      data-testid="feedback-widget-trigger"
      className="inline-flex h-9 min-w-9 shrink-0 items-center justify-center gap-1 rounded-xl bg-indigo-600 px-2 text-xs font-semibold text-white shadow-sm transition-[background-color,color,border-color] hover:bg-indigo-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/40 focus-visible:ring-offset-2 md:px-2.5 dark:bg-indigo-500 dark:hover:bg-indigo-400"
    >
      <MessageSquarePlus className="h-4 w-4 shrink-0" aria-hidden />
      <span className="hidden max-w-[8rem] truncate md:inline">{label}</span>
    </button>
  )
}

export function FeedbackWidgetMenu() {
  const { ffFeedback } = usePlatformFeatures()
  const { t } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const triggerRef = useRef<HTMLButtonElement>(null)
  const label = t('feedback.button')

  if (!ffFeedback) return null

  function handleClose() {
    setOpen(false)
    window.setTimeout(() => triggerRef.current?.focus(), 0)
  }

  return (
    <>
      <FeedbackWidgetTrigger
        open={open}
        label={label}
        triggerRef={triggerRef}
        onOpen={() => setOpen(true)}
      />
      <FeedbackDialog open={open} onClose={handleClose} />
    </>
  )
}
