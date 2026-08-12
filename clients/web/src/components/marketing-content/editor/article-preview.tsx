import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

function previewMarkdown(markdown: string) {
  return markdown.replace(/^:::[^\n]*$/gm, '').replace(/^<!--.*-->$/gm, '')
}

export function ArticlePreview({ title, body, dir = 'auto' }: { title: string; body: string; dir?: 'ltr' | 'rtl' | 'auto' }) {
  return <article dir={dir} className="prose prose-sm max-w-none rounded-xl bg-surface-raised p-5 text-fg-default">
    <h1>{title || 'Untitled article'}</h1>
    <ReactMarkdown remarkPlugins={[remarkGfm]}>{previewMarkdown(body)}</ReactMarkdown>
  </article>
}
