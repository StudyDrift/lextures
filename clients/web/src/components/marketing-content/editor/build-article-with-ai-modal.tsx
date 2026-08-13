import { useEffect, useState } from 'react'
import { Sparkles } from 'lucide-react'
import { Button, Dialog, Field, InlineAlert, Textarea } from '../../ui'
import type { MarketingArticleAIDraft } from '../../../lib/marketing-content-ai-api'

type Props = {
  open: boolean
  kind: 'blog' | 'doc'
  existingTitle: string
  existingBodyMd: string
  onClose: () => void
  onBuild: (prompt: string) => Promise<MarketingArticleAIDraft>
  onBuilt: (draft: MarketingArticleAIDraft) => void
}

export function BuildArticleWithAiModal({
  open,
  kind,
  existingTitle,
  existingBodyMd,
  onClose,
  onBuild,
  onBuilt,
}: Props) {
  const [prompt, setPrompt] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!open) return
    setPrompt('')
    setBusy(false)
    setError('')
  }, [open])

  const noun = kind === 'doc' ? 'help article' : 'blog post'
  const replaceNote = existingTitle.trim() || existingBodyMd.trim()
    ? ' This replaces the current title, summary, metadata, and body. Nothing is saved until you click Save.'
    : ' Nothing is saved until you click Save.'

  async function submit() {
    const text = prompt.trim()
    if (!text || busy) return
    setBusy(true)
    setError('')
    try {
      const draft = await onBuild(text)
      if (!draft.title.trim() && !draft.bodyMd.trim()) {
        setError('No article was generated. Try a more specific description.')
        return
      }
      onBuilt(draft)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Could not generate the article.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog
      open={open}
      onClose={() => { if (!busy) onClose() }}
      size="lg"
      title={<span className="inline-flex items-center gap-1.5"><Sparkles className="h-4 w-4 text-accent-fg" aria-hidden />Build with AI</span>}
      description={`Describe what this ${noun} should cover.${replaceNote}`}
      closeLabel="Close"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={busy}>Cancel</Button>
          <Button loading={busy} disabled={busy || prompt.trim() === ''} onClick={() => void submit()}>
            Generate
          </Button>
        </>
      }
    >
      <Field label="What should this article cover?" description="⌘/Ctrl + Enter to generate">
        <Textarea
          rows={6}
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          disabled={busy}
          autoFocus
          placeholder={kind === 'doc'
            ? 'e.g. A how-to for teachers creating a course, covering modules, publishing, and inviting students…'
            : 'e.g. Why homeschool families need one place for grades, transcripts, and lesson plans — practical, not salesy…'}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
              e.preventDefault()
              void submit()
            }
          }}
        />
      </Field>
      {error ? <InlineAlert tone="danger" className="mt-3">{error}</InlineAlert> : null}
    </Dialog>
  )
}
