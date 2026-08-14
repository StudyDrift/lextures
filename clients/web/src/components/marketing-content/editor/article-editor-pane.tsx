import type { Editor } from '@tiptap/core'
import { MarkdownBodyEditor } from '../../editor/block-editor'
import { Input, Textarea } from '../../ui'
import type { MarketingArticle } from '../../../lib/marketing-content-api'

type EditorPaneProps = {
  article: MarketingArticle
  canAuthor: boolean
  simple: boolean
  titleHighlight?: boolean
  titleError?: string
  onTitleChange: (title: string) => void
  onBodyChange: (bodyMd: string) => void
  onBlur: () => void
  onEditorChange: (sectionId: string, editor: Editor | null) => void
}

// Keep this component at module scope. Defining it inside ArticleEditorPage gives
// it a new component identity on every keystroke, which remounts TipTap while
// PureEditorContent is completing its own mount update.
export function EditorPane({
  article,
  canAuthor,
  simple,
  titleHighlight,
  titleError,
  onTitleChange,
  onBodyChange,
  onBlur,
  onEditorChange,
}: EditorPaneProps) {
  const path = article.path || `${article.locale && article.locale !== 'en' ? `/${article.locale}` : ''}${article.kind === 'blog' ? '/blog/' : '/docs/…/'}${article.slug || '…'}`
  return <section className="min-w-0 overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-sm" aria-label="Article body editor">
    <div className="border-b border-border-subtle px-5 pb-4 pt-6 sm:px-9">
      <Input
        id="article-title"
        value={article.title}
        onChange={(event) => onTitleChange(event.target.value)}
        onBlur={onBlur}
        disabled={!canAuthor}
        aria-label="Article title"
        aria-invalid={titleError ? true : undefined}
        placeholder="Untitled article"
        className={`w-full border-0 bg-transparent p-0 text-2xl font-semibold tracking-tight text-fg-default shadow-none placeholder:text-fg-subtle disabled:opacity-60 sm:text-3xl ${titleHighlight ? 'rounded-lg ring-2 ring-accent-solid ring-offset-2 ring-offset-surface-raised' : ''}`}
      />
      {titleError ? <p className="mt-2 text-xs font-medium text-danger-fg">{titleError}</p> : null}
      <p className="mt-2 font-mono text-xs text-fg-muted">{path}</p>
    </div>
    <div className="px-5 py-5 sm:px-9">
      {simple
        ? <><label className="sr-only" htmlFor="article-markdown">Article body, markdown</label><Textarea id="article-markdown" dir="ltr" aria-describedby="article-findings" className="min-h-[52vh] resize-y border-0 bg-transparent p-0 font-mono text-sm shadow-none" value={article.bodyMd} onChange={(e) => onBodyChange(e.target.value)} onBlur={onBlur} /></>
        : <div className="min-h-[52vh]"><MarkdownBodyEditor sectionId="marketing-article" value={article.bodyMd} onChange={onBodyChange} onBlur={onBlur} disabled={!canAuthor} placeholder="Write the article… type / for blocks" onEditorChange={onEditorChange} /></div>}
    </div>
  </section>
}
