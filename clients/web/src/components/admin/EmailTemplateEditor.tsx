import Link from '@tiptap/extension-link'
import Placeholder from '@tiptap/extension-placeholder'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { InputDialog } from '../input-dialog'

type EmailTemplateEditorProps = {
  value: string
  onChange: (html: string) => void
  onInsertReady?: (insert: (token: string) => void) => void
  disabled?: boolean
  placeholder?: string
}

export function EmailTemplateEditor({
  value,
  onChange,
  onInsertReady,
  disabled = false,
  placeholder = 'Edit email body…',
}: EmailTemplateEditorProps) {
  const { t } = useTranslation('common')
  const [linkDialogOpen, setLinkDialogOpen] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  const editor = useEditor({
    extensions: [
      StarterKit.configure({ heading: false, codeBlock: false, blockquote: false }),
      Link.configure({ openOnClick: false, autolink: true }),
      Placeholder.configure({ placeholder }),
    ],
    content: value,
    editable: !disabled,
    onUpdate: ({ editor: ed }) => {
      onChange(ed.getHTML())
    },
  })

  useEffect(() => {
    if (!editor) return
    if (editor.getHTML() !== value) {
      editor.commands.setContent(value, { emitUpdate: false })
    }
  }, [editor, value])

  useEffect(() => {
    if (!editor) return
    editor.setEditable(!disabled)
  }, [editor, disabled])

  useEffect(() => {
    if (!editor || !onInsertReady) return
    onInsertReady((token: string) => {
      editor.chain().focus().insertContent(token).run()
    })
  }, [editor, onInsertReady])

  return (
    <>
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-slate-50/40 dark:border-neutral-600 dark:bg-neutral-950/30">
      <div className="flex flex-wrap gap-0.5 border-b border-slate-200 bg-white px-2 py-1.5 dark:border-neutral-600 dark:bg-neutral-900">
        <button
          type="button"
          disabled={disabled}
          onClick={() => editor?.chain().focus().toggleBold().run()}
          className="rounded-md px-2 py-1 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-700"
        >
          Bold
        </button>
        <button
          type="button"
          disabled={disabled}
          onClick={() => editor?.chain().focus().toggleItalic().run()}
          className="rounded-md px-2 py-1 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-700"
        >
          Italic
        </button>
        <button
          type="button"
          disabled={disabled}
          onClick={() => {
            const prev = editor?.getAttributes('link').href as string | undefined
            setLinkUrl(prev ?? '')
            setLinkDialogOpen(true)
          }}
          className="rounded-md px-2 py-1 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-700"
        >
          Link
        </button>
      </div>
      <EditorContent
        editor={editor}
        className="prose prose-sm max-w-none min-h-[260px] px-3.5 py-3 focus-within:outline-none dark:prose-invert"
      />
      <div className="border-t border-slate-200 bg-white px-3 py-2 text-xs text-slate-500 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-400">
        Type <code className="rounded bg-slate-100 px-1 dark:bg-neutral-800">{'{{'}</code> or use merge-field buttons.
      </div>
    </div>
    <InputDialog
      open={linkDialogOpen}
      title={t('dialogs.linkUrl.title')}
      label={t('dialogs.linkUrl.label')}
      placeholder={t('dialogs.linkUrl.placeholder')}
      value={linkUrl}
      onValueChange={setLinkUrl}
      onConfirm={(url) => {
        const trimmed = url.trim()
        if (trimmed) {
          editor?.chain().focus().extendMarkRange('link').setLink({ href: trimmed }).run()
        }
        setLinkDialogOpen(false)
      }}
      onClose={() => setLinkDialogOpen(false)}
    />
    </>
  )
}

export function MergeFieldButton({
  label,
  token,
  onInsert,
  disabled,
}: {
  label: string
  token: string
  onInsert: (token: string) => void
  disabled?: boolean
}) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={() => onInsert(token)}
      className="inline-flex items-center rounded-lg border border-slate-200 bg-white px-2.5 py-1 text-xs font-medium text-slate-700 shadow-sm transition-colors hover:border-indigo-200 hover:bg-indigo-50 hover:text-indigo-800 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:border-indigo-500/40 dark:hover:bg-indigo-950/40 dark:hover:text-indigo-200"
      title={token}
    >
      {label}
    </button>
  )
}
