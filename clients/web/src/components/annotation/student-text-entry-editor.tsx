import type { Editor } from '@tiptap/core'
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { InputDialog } from '../input-dialog'
import { BlockFloatingToolbar } from '../editor/block-editor/block-floating-toolbar'
import { MarkdownBodyEditor } from '../editor/block-editor/markdown-body-editor'
import { MarkdownFormatToolbar } from '../editor/block-editor/markdown-format-toolbar'
import type { MarkdownEditKind } from '../editor/block-editor/markdown-insert'
import { MarkdownArticleView } from '../syllabus/syllabus-markdown-view'
import { submissionTextHasMarkdown } from './student-text-entry-markdown'

export type StudentTextEntryEditorProps = {
  value: string
  onChange: (markdown: string) => void
  disabled?: boolean
  labelledBy?: string
}

export function SubmissionBodyText({
  markdown,
  courseCode,
}: {
  markdown: string
  courseCode?: string
}) {
  if (!submissionTextHasMarkdown(markdown)) {
    return <p className="whitespace-pre-wrap">{markdown}</p>
  }
  return <MarkdownArticleView markdown={markdown} courseCode={courseCode} />
}

function applyFormat(editor: Editor, kind: MarkdownEditKind, onLink: () => void) {
  const chain = editor.chain().focus()
  switch (kind) {
    case 'bold':
      chain.toggleBold().run()
      return
    case 'italic':
      chain.toggleItalic().run()
      return
    case 'inlineCode':
      chain.toggleCode().run()
      return
    case 'codeBlock':
      chain.toggleCodeBlock().run()
      return
    case 'bulletList':
      chain.toggleBulletList().run()
      return
    case 'orderedList':
      chain.toggleOrderedList().run()
      return
    case 'table':
      chain.insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
      return
    case 'link':
      onLink()
      return
    default: {
      const _exhaustive: never = kind
      void _exhaustive
    }
  }
}

/**
 * Assignment text entry: the same markdown editor used for pages, with a
 * persistent toolbar so students can apply lists, bold, and other formatting.
 */
export function StudentTextEntryEditor({
  value,
  onChange,
  disabled,
  labelledBy,
}: StudentTextEntryEditorProps) {
  const { t } = useTranslation('common')
  const editorRef = useRef<Editor | null>(null)
  const [linkOpen, setLinkOpen] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')

  function openLinkDialog() {
    const editor = editorRef.current
    const prev = editor?.getAttributes('link').href as string | undefined
    setLinkUrl(prev && prev.length > 0 ? prev : 'https://')
    setLinkOpen(true)
  }

  function confirmLink(url: string) {
    const editor = editorRef.current
    const trimmed = url.trim()
    setLinkOpen(false)
    if (!editor || !trimmed) return
    const { from, to } = editor.state.selection
    if (from === to) {
      editor
        .chain()
        .focus()
        .insertContent({
          type: 'text',
          text: trimmed,
          marks: [{ type: 'link', attrs: { href: trimmed } }],
        })
        .run()
      return
    }
    editor.chain().focus().extendMarkRange('link').setLink({ href: trimmed }).run()
  }

  return (
    <>
      <div className="overflow-hidden rounded-lg border border-border-strong bg-surface-raised shadow-sm dark:border-border-default dark:bg-surface-base">
        <div className="overflow-x-auto border-b border-border-default">
          <BlockFloatingToolbar embedded label="Response formatting">
            <MarkdownFormatToolbar
              disabled={disabled}
              onApply={(kind) => {
                const editor = editorRef.current
                if (!editor) return
                applyFormat(editor, kind, openLinkDialog)
              }}
            />
          </BlockFloatingToolbar>
        </div>
        <div className="px-3 py-2">
          <MarkdownBodyEditor
            sectionId="assignment-text-entry"
            value={value}
            onChange={onChange}
            disabled={disabled}
            labelledBy={labelledBy}
            size="comfortable"
            placeholder="Type your answer here…"
            onEditorChange={(_sectionId, editor) => {
              editorRef.current = editor
            }}
          />
        </div>
      </div>
      <InputDialog
        open={linkOpen}
        title={t('dialogs.linkUrl.title')}
        label={t('dialogs.linkUrl.label')}
        placeholder={t('dialogs.linkUrl.placeholder')}
        value={linkUrl}
        onValueChange={setLinkUrl}
        onConfirm={confirmLink}
        onClose={() => setLinkOpen(false)}
      />
    </>
  )
}
