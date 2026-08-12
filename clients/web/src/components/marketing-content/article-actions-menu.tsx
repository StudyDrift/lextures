import { useRef, useState } from 'react'
import { MoreHorizontal } from 'lucide-react'
import { IconButton, Menu } from '../ui'
import type { MarketingArticleRow } from '../../lib/marketing-content-api'

export function ArticleActionsMenu({ article, canAuthor, canPublish, onAction }: {
  article: MarketingArticleRow
  canAuthor: boolean
  canPublish: boolean
  onAction: (action: string, article: MarketingArticleRow) => void
}) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLButtonElement>(null)
  const items = [
    { id: 'open', label: 'Open' },
    { id: 'preview', label: 'Preview' },
    ...(canAuthor ? [{ id: 'duplicate', label: 'Duplicate', disabled: true }] : []),
    ...(canPublish ? [
      { id: article.status === 'published' ? 'unpublish' : 'publish', label: article.status === 'published' ? 'Unpublish' : 'Publish' },
      { id: 'archive', label: 'Archive', danger: true },
    ] : []),
    { id: 'copy_url', label: 'Copy public URL' },
  ]
  return <>
    <IconButton ref={ref} aria-label={`Actions for ${article.title}`} aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((v) => !v)}><MoreHorizontal className="h-4 w-4" /></IconButton>
    <Menu open={open} onOpenChange={setOpen} anchorRef={ref} items={items} placement="bottom-end" onAction={(action) => onAction(action, article)} />
  </>
}
