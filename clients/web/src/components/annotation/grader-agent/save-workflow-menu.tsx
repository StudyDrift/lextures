import { useEffect, useId, useRef, useState } from 'react'
import { ChevronDown, Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SaveTemplateDialog } from './save-template-dialog'

type SaveWorkflowMenuProps = {
  saving: boolean
  defaultTemplateName?: string
  acceptVisible?: boolean
  acceptDisabled?: boolean
  acceptTooltip?: string | null
  onSave: () => void | Promise<void>
  onSaveAsTemplate: (name: string) => void | Promise<void>
  onAccept?: () => void | Promise<void>
}

export function SaveWorkflowMenu({
  saving,
  defaultTemplateName,
  acceptVisible = false,
  acceptDisabled = false,
  acceptTooltip = null,
  onSave,
  onSaveAsTemplate,
  onAccept,
}: SaveWorkflowMenuProps) {
  const { t } = useTranslation('common')
  const [open, setOpen] = useState(false)
  const [templateDialogOpen, setTemplateDialogOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  const menuId = useId()

  useEffect(() => {
    if (!open) return
    function onDoc(e: MouseEvent) {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false)
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false)
    }
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
    }
  }, [open])

  return (
    <>
      <div ref={rootRef} className="relative shrink-0">
        <button
          type="button"
          disabled={saving}
          aria-haspopup="menu"
          aria-expanded={open}
          aria-controls={open ? menuId : undefined}
          onClick={() => setOpen((o) => !o)}
          className="inline-flex items-center gap-2 rounded-xl border border-border-strong bg-surface-raised px-3 py-2 text-sm font-semibold text-fg-default shadow-sm transition-[background-color,color,border-color] hover:bg-surface-base disabled:cursor-not-allowed disabled:opacity-60 dark:border-border-default dark:bg-surface-raised dark:text-fg-default dark:hover:bg-surface-overlay"
        >
          {saving ? (
            <>
              <Loader2 className="h-4 w-4 motion-safe:animate-spin" aria-hidden />
              <span>{t('gradingAgent.save.saving')}</span>
            </>
          ) : (
            <>
              {t('gradingAgent.save')}
              <ChevronDown className={`h-4 w-4 shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} aria-hidden />
            </>
          )}
        </button>
        {open && !saving ? (
          <div
            id={menuId}
            role="menu"
            aria-label={t('gradingAgent.save.menuLabel')}
            className="absolute end-0 z-50 mt-1 min-w-[12rem] overflow-hidden rounded-xl border border-border-default bg-surface-raised py-1 shadow-lg shadow-slate-900/10 dark:border-border-default dark:bg-surface-raised"
          >
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false)
                void onSave()
              }}
              className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-fg-default transition-[background-color,color,border-color] hover:bg-surface-base dark:text-fg-default dark:hover:bg-surface-overlay"
            >
              {t('gradingAgent.save.option')}
            </button>
            <button
              type="button"
              role="menuitem"
              onClick={() => {
                setOpen(false)
                setTemplateDialogOpen(true)
              }}
              className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-fg-default transition-[background-color,color,border-color] hover:bg-surface-base dark:text-fg-default dark:hover:bg-surface-overlay"
            >
              {t('gradingAgent.save.asTemplate')}
            </button>
            {acceptVisible && onAccept ? (
              <>
                <div className="my-1 border-t border-border-subtle" role="separator" aria-hidden />
                <button
                  type="button"
                  role="menuitem"
                  disabled={acceptDisabled}
                  title={acceptDisabled && acceptTooltip ? acceptTooltip : undefined}
                  onClick={() => {
                    if (acceptDisabled) return
                    setOpen(false)
                    void onAccept()
                  }}
                  className="flex w-full items-center gap-2 px-2.5 py-2 text-start text-sm font-medium text-accent-fg transition-[background-color,color,border-color] hover:bg-indigo-50 disabled:cursor-not-allowed disabled:opacity-50 dark:text-indigo-300 dark:hover:bg-indigo-950/40"
                >
                  {t('gradingAgent.accept')}
                </button>
              </>
            ) : null}
          </div>
        ) : null}
      </div>
      {templateDialogOpen ? (
        <SaveTemplateDialog
          defaultName={defaultTemplateName ?? ''}
          saving={saving}
          onClose={() => setTemplateDialogOpen(false)}
          onSave={async (name) => {
            await onSaveAsTemplate(name)
            setTemplateDialogOpen(false)
          }}
        />
      ) : null}
    </>
  )
}
