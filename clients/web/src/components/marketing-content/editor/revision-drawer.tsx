import { useEffect, useMemo, useState } from 'react'
import { getMarketingRevision, listMarketingRevisions, type MarketingRevision } from '../../../lib/marketing-content-api'
import { Button, Drawer, InlineAlert, Select } from '../../ui'
import { simpleLineDiff } from './article-editor-utils'

export function RevisionDrawer({ open, articleId, currentBody, onClose, onRestore }: { open: boolean; articleId: string; currentBody: string; onClose: () => void; onRestore: (no: number) => Promise<void> }) {
  const [items, setItems] = useState<MarketingRevision[]>([])
  const [selected, setSelected] = useState<number | null>(null)
  const [body, setBody] = useState('')
  const [error, setError] = useState('')
  useEffect(() => { if (open) void listMarketingRevisions(articleId).then((v) => { setItems(v.items); setSelected(v.items[0]?.revisionNo ?? null) }).catch((e) => setError(String(e))) }, [articleId, open])
  useEffect(() => { if (open && selected != null) void getMarketingRevision(articleId, selected).then((v) => setBody(v.bodyMd ?? '')).catch((e) => setError(String(e))) }, [articleId, open, selected])
  const diff = useMemo(() => simpleLineDiff(body, currentBody), [body, currentBody])
  const added = diff.filter((v) => v.type === 'added').length
  const removed = diff.filter((v) => v.type === 'removed').length
  return <Drawer open={open} onClose={onClose} title={'Revision history'}>
    <div className="space-y-4">
      {error ? <InlineAlert tone="danger">{error}</InlineAlert> : null}
      <label className="block text-sm font-medium">Revision<Select className="mt-1" value={selected ?? ''} onChange={(e) => setSelected(Number(e.target.value))}>{items.map((item) => <option key={item.revisionNo} value={item.revisionNo}>#{item.revisionNo} · {new Date(item.createdAt).toLocaleString()} · {item.statusAfter}{item.changeNote ? ` · ${item.changeNote}` : ''}</option>)}</Select></label>
      <p className="text-sm text-fg-muted" aria-live="polite">{added} lines added, {removed} removed compared with the current draft.</p>
      <div className="max-h-[55vh] overflow-auto rounded-lg border border-border-default bg-surface-sunken font-mono text-xs" aria-label={`Diff: ${added} lines added, ${removed} removed`}>
        {diff.map((line, index) => <div key={`${index}-${line.type}`} className={line.type === 'added' ? 'bg-success-surface text-success-fg' : line.type === 'removed' ? 'bg-danger-surface text-danger-fg' : 'text-fg-muted'}><span className="inline-block w-7 select-none px-1 text-end" aria-hidden>{line.type === 'added' ? '+' : line.type === 'removed' ? '−' : ' '}</span><span className="whitespace-pre-wrap">{line.text || ' '}</span></div>)}
      </div>
      <Button disabled={selected == null} onClick={() => selected == null ? undefined : void onRestore(selected)}>Restore this revision</Button>
    </div>
  </Drawer>
}
