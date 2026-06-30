import Link from '@tiptap/extension-link'
import Placeholder from '@tiptap/extension-placeholder'
import { EditorContent, useEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { useEffect } from 'react'

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
    <div className="rounded-xl border border-slate-200 dark:border-neutral-700">
      <div className="flex flex-wrap gap-1 border-b border-slate-200 p-2 dark:border-neutral-700">
        <button
          type="button"
          disabled={disabled}
          onClick={() => editor?.chain().focus().toggleBold().run()}
          className="rounded px-2 py-1 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          Bold
        </button>
        <button
          type="button"
          disabled={disabled}
          onClick={() => editor?.chain().focus().toggleItalic().run()}
          className="rounded px-2 py-1 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          Italic
        </button>
        <button
          type="button"
          disabled={disabled}
          onClick={() => {
            const url = window.prompt('Link URL')
            if (!url) return
            editor?.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
          }}
          className="rounded px-2 py-1 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:opacity-50 dark:text-neutral-300 dark:hover:bg-neutral-800"
        >
          Link
        </button>
      </div>
      <EditorContent
        editor={editor}
        className="prose prose-sm max-w-none px-3 py-2 dark:prose-invert min-h-[220px] focus-within:outline-none"
      />
      <div className="border-t border-slate-200 p-2 text-xs text-slate-500 dark:border-neutral-700 dark:text-neutral-400">
        Type <code className="rounded bg-slate-100 px-1 dark:bg-neutral-800">{'{{'}</code> or use merge-field buttons.
      </div>
    </div>
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
      className="rounded-full border border-slate-200 bg-white px-2 py-0.5 text-xs text-slate-700 hover:bg-slate-50 disabled:opacity-50 dark:border-neutral-600 dark:bg-neutral-900 dark:text-neutral-200 dark:hover:bg-neutral-800"
      title={token}
    >
      {label}
    </button>
  )
}
