import { useEffect, useRef, useState } from 'react'
import { X, Sparkles, Send, Code, Eye, Save, RotateCcw } from 'lucide-react'
import { generateVibeActivityHTML, type VibeGenerateMessage } from '../../lib/courses-api'

type ChatMessage = { role: 'user' | 'assistant'; content: string }

type VibeActivityCreateModalProps = {
  open: boolean
  onClose: () => void
  onSave: (title: string, html: string) => void | Promise<void>
  saving?: boolean
  error?: string | null
  initialTitle?: string
  initialHtml?: string
  courseCode: string
}

export function VibeActivityCreateModal({
  open,
  onClose,
  onSave,
  saving,
  error,
  initialTitle = '',
  initialHtml = '',
  courseCode,
}: VibeActivityCreateModalProps) {
  const [phase, setPhase] = useState<'prompt' | 'split'>(initialHtml ? 'split' : 'prompt')
  const [title, setTitle] = useState(initialTitle)
  const [html, setHtml] = useState(initialHtml)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [generating, setGenerating] = useState(false)
  const [genError, setGenError] = useState<string | null>(null)
  const [rightTab, setRightTab] = useState<'preview' | 'code'>('preview')
  const chatEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (!open) return
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  if (!open) return null

  const canSave = title.trim().length > 0 && html.trim().length > 0

  async function submit() {
    const prompt = input.trim()
    if (!prompt || generating) return
    setInput('')
    setGenError(null)
    setGenerating(true)

    const userMsg: ChatMessage = { role: 'user', content: prompt }
    const nextMessages = [...messages, userMsg]
    setMessages(nextMessages)

    const history: VibeGenerateMessage[] = messages.map((m) => ({ role: m.role, content: m.content }))

    try {
      const generated = await generateVibeActivityHTML(courseCode, prompt, history)
      const assistantMsg: ChatMessage = { role: 'assistant', content: generated }
      setMessages([...nextMessages, assistantMsg])
      setHtml(generated)
      setPhase('split')
      setRightTab('preview')
      if (!title) {
        // Auto-derive a title from the first prompt (truncated)
        setTitle(prompt.length > 60 ? prompt.slice(0, 57) + '…' : prompt)
      }
    } catch (e) {
      setGenError(e instanceof Error ? e.message : 'Generation failed.')
      setMessages(nextMessages.slice(0, -1))
    } finally {
      setGenerating(false)
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      void submit()
    }
  }

  async function handleSave() {
    if (!canSave || saving) return
    await onSave(title.trim(), html)
  }

  return (
    <div className="fixed inset-0 z-[70] flex items-center justify-center bg-black/60 p-2" role="dialog" aria-modal>
      <div className="flex h-[96vh] w-[98vw] flex-col overflow-hidden rounded-2xl border border-border-default bg-surface-raised shadow-2xl dark:border-border-default dark:bg-surface-raised">

        {/* Header */}
        <div className="flex shrink-0 items-center justify-between border-b border-border-default px-4 py-2.5 dark:border-border-default">
          <div className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-rose-600" />
            <span className="font-semibold text-slate-950 dark:text-fg-default">Vibe Activity Builder</span>
          </div>
          <div className="flex items-center gap-2">
            {phase === 'split' && (
              <>
                <input
                  value={title}
                  onChange={(e) => setTitle(e.target.value)}
                  placeholder="Activity title…"
                  className="rounded-lg border border-border-strong bg-surface-raised px-3 py-1.5 text-sm dark:border-border-default dark:bg-surface-overlay dark:text-fg-default"
                />
                <button
                  type="button"
                  onClick={handleSave}
                  disabled={!canSave || saving}
                  className="inline-flex items-center gap-1.5 rounded-lg bg-rose-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-rose-700 disabled:cursor-not-allowed disabled:opacity-60"
                >
                  <Save className="h-4 w-4" />
                  {saving ? 'Saving…' : 'Save'}
                </button>
              </>
            )}
            {error && <span className="text-xs text-danger-fg">{error}</span>}
            <button
              onClick={onClose}
              className="rounded p-1 text-fg-muted hover:bg-surface-sunken dark:hover:bg-surface-overlay"
              aria-label="Close"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        </div>

        {/* Body */}
        {phase === 'prompt' ? (
          /* ── INITIAL CENTERED PROMPT STATE ── */
          <div className="flex flex-1 flex-col items-center justify-center gap-6 p-8">
            <div className="text-center">
              <Sparkles className="mx-auto mb-3 h-10 w-10 text-rose-500" />
              <h2 className="text-2xl font-semibold text-fg-default">What should this activity do?</h2>
              <p className="mt-1 text-sm text-fg-muted">
                Describe the interactive activity you want to create. The AI will generate it as a self-contained HTML page.
              </p>
            </div>

            <div className="w-full max-w-2xl">
              <div className="flex flex-col gap-2 rounded-2xl border border-border-strong bg-surface-raised p-3 shadow-sm dark:border-border-default dark:bg-surface-overlay">
                <textarea
                  ref={inputRef}
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="e.g. A drag-and-drop cell labeling activity for biology students…"
                  rows={4}
                  className="w-full resize-none bg-transparent text-sm text-fg-default outline-none placeholder:text-fg-subtle dark:text-fg-default dark:placeholder:text-neutral-500"
                />
                <div className="flex items-center justify-between">
                  {genError && <span className="text-xs text-danger-fg">{genError}</span>}
                  <div className="ml-auto">
                    <button
                      type="button"
                      onClick={() => void submit()}
                      disabled={!input.trim() || generating}
                      className="inline-flex items-center gap-2 rounded-xl bg-rose-600 px-4 py-2 text-sm font-medium text-white hover:bg-rose-700 disabled:cursor-not-allowed disabled:opacity-60"
                    >
                      <Send className="h-4 w-4" />
                      {generating ? 'Generating…' : 'Generate'}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div className="text-xs text-fg-subtle">
              Tip: Include Tailwind CSS, buttons, animations, and anything else to make it interactive. Press Shift+Enter for a new line.
            </div>
          </div>
        ) : (
          /* ── SPLIT CHAT / PREVIEW STATE ── */
          <div className="flex min-h-0 flex-1">
            {/* Left: conversation */}
            <div className="flex w-[38%] shrink-0 flex-col border-r border-border-default">
              {/* Chat messages */}
              <div className="flex-1 overflow-y-auto p-4 space-y-4">
                {messages.map((m, i) => (
                  <div key={i} className={m.role === 'user' ? 'flex justify-end' : 'flex justify-start'}>
                    {m.role === 'user' ? (
                      <div className="max-w-[80%] rounded-2xl rounded-br-sm bg-rose-600 px-3 py-2 text-sm text-white">
                        {m.content}
                      </div>
                    ) : (
                      <div className="max-w-[80%] rounded-2xl rounded-bl-sm bg-surface-sunken px-3 py-2 text-xs text-fg-muted dark:bg-surface-overlay dark:text-fg-muted">
                        <span className="font-medium text-rose-600">HTML generated</span>
                        <span className="ml-1 text-fg-subtle">— see preview →</span>
                      </div>
                    )}
                  </div>
                ))}
                {generating && (
                  <div className="flex justify-start">
                    <div className="rounded-2xl rounded-bl-sm bg-surface-sunken px-3 py-2 text-sm text-fg-muted dark:bg-surface-overlay dark:text-fg-muted">
                      Generating…
                    </div>
                  </div>
                )}
                <div ref={chatEndRef} />
              </div>

              {/* Input */}
              <div className="shrink-0 border-t border-border-default p-3 dark:border-border-default">
                {genError && <p className="mb-2 text-xs text-danger-fg">{genError}</p>}
                <div className="flex items-end gap-2 rounded-xl border border-border-strong bg-surface-raised p-2 dark:border-border-default dark:bg-surface-overlay">
                  <textarea
                    value={input}
                    onChange={(e) => setInput(e.target.value)}
                    onKeyDown={handleKeyDown}
                    placeholder="Describe a change or addition… (Enter to send)"
                    rows={2}
                    className="flex-1 resize-none bg-transparent text-sm text-fg-default outline-none placeholder:text-fg-subtle dark:text-fg-default dark:placeholder:text-neutral-500"
                  />
                  <button
                    type="button"
                    onClick={() => void submit()}
                    disabled={!input.trim() || generating}
                    className="shrink-0 rounded-lg bg-rose-600 p-2 text-white hover:bg-rose-700 disabled:opacity-50"
                  >
                    <Send className="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>

            {/* Right: code / preview */}
            <div className="flex min-w-0 flex-1 flex-col">
              {/* Tab bar */}
              <div className="flex shrink-0 items-center gap-1 border-b border-border-default px-4 py-2 dark:border-border-default">
                <button
                  type="button"
                  onClick={() => setRightTab('preview')}
                  className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${ rightTab === 'preview' ? 'bg-surface-sunken text-fg-default dark:bg-neutral-700 dark:text-fg-default' : 'text-fg-muted hover:text-fg-muted dark:text-fg-muted dark:hover:text-fg-default' }`}
                >
                  <Eye className="h-4 w-4" />
                  Preview
                </button>
                <button
                  type="button"
                  onClick={() => setRightTab('code')}
                  className={`inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition-colors ${ rightTab === 'code' ? 'bg-surface-sunken text-fg-default dark:bg-neutral-700 dark:text-fg-default' : 'text-fg-muted hover:text-fg-muted dark:text-fg-muted dark:hover:text-fg-default' }`}
                >
                  <Code className="h-4 w-4" />
                  Code
                </button>
                <button
                  type="button"
                  onClick={() => setHtml('')}
                  className="ml-auto inline-flex items-center gap-1 text-xs text-fg-subtle hover:text-fg-muted dark:hover:text-fg-muted"
                  title="Clear HTML"
                >
                  <RotateCcw className="h-3.5 w-3.5" />
                  Reset
                </button>
              </div>

              {/* Tab content */}
              <div className="min-h-0 flex-1">
                {rightTab === 'preview' ? (
                  <iframe
                    key={html}
                    title="vibe-preview"
                    sandbox="allow-scripts allow-forms allow-same-origin"
                    srcDoc={html || '<!doctype html><html><body style="padding:2rem;font-family:sans-serif;color:#888">No content yet — describe your activity on the left.</body></html>'}
                    className="block h-full w-full bg-surface-raised"
                  />
                ) : (
                  <textarea
                    value={html}
                    onChange={(e) => setHtml(e.target.value)}
                    spellCheck={false}
                    className="h-full w-full resize-none bg-slate-950 p-4 font-mono text-xs text-slate-100 outline-none"
                  />
                )}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
