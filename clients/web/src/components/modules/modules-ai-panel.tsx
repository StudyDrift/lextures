import { useCallback, useEffect, useId, useRef, useState } from 'react'
import { Bot, Send, Sparkles, X } from 'lucide-react'
import {
  postModulesAiChat,
  type CourseStructureItem,
  type ModulesAiProposal,
} from '../../lib/courses-api'
import {
  applyModulesAiProposal,
  applyModulesAiProposals,
  describeModulesAiProposal,
} from '../../lib/modules-ai-apply'
import { toast, toastMutationError } from '../../lib/lms-toast'

type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  proposals?: ModulesAiProposal[]
}

type Props = {
  courseCode: string
  structureItems: CourseStructureItem[]
  onStructureChanged: () => void | Promise<void>
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ModulesAiPanel({
  courseCode,
  structureItems,
  onStructureChanged,
  open,
  onOpenChange,
}: Props) {
  const titleId = useId()
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [busy, setBusy] = useState(false)
  const [applyingKey, setApplyingKey] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    if (open) {
      inputRef.current?.focus()
    }
  }, [open])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, busy])

  const sendMessage = useCallback(async () => {
    const text = input.trim()
    if (!text || busy) return
    setInput('')
    setError(null)
    setMessages((prev) => [...prev, { role: 'user', content: text }])
    setBusy(true)
    try {
      const history = messages.map((m) => ({ role: m.role, content: m.content }))
      const res = await postModulesAiChat(courseCode, { message: text, history })
      setMessages((prev) => [
        ...prev,
        {
          role: 'assistant',
          content: res.reply || 'Here are proposed changes.',
          proposals: res.proposals,
        },
      ])
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to reach the Modules AI assistant.'
      setError(msg)
      toastMutationError(msg)
    } finally {
      setBusy(false)
    }
  }, [busy, courseCode, input, messages])

  const applyOne = useCallback(
    async (proposal: ModulesAiProposal, key: string, siblings: ModulesAiProposal[] = []) => {
      setApplyingKey(key)
      setError(null)
      try {
        const needsParentCreate =
          (proposal.op === 'create_content_page' ||
            proposal.op === 'create_assignment' ||
            proposal.op === 'create_quiz' ||
            proposal.op === 'create_heading') &&
          !proposal.moduleId &&
          Boolean(proposal.moduleTitle?.trim())
        const parentCreate = needsParentCreate
          ? siblings.find(
              (s) =>
                s.op === 'create_module' &&
                s.title.trim().toLowerCase() === proposal.moduleTitle!.trim().toLowerCase(),
            )
          : undefined
        if (parentCreate) {
          await applyModulesAiProposals(courseCode, [parentCreate, proposal], structureItems)
        } else {
          await applyModulesAiProposal(courseCode, proposal, structureItems)
        }
        toast('Applied outline change')
        await onStructureChanged()
      } catch (e) {
        const msg = e instanceof Error ? e.message : 'Could not apply change.'
        setError(msg)
        toastMutationError(msg)
      } finally {
        setApplyingKey(null)
      }
    },
    [courseCode, onStructureChanged, structureItems],
  )

  const applyAll = useCallback(
    async (proposals: ModulesAiProposal[], msgIndex: number) => {
      setApplyingKey(`all-${msgIndex}`)
      setError(null)
      try {
        await applyModulesAiProposals(courseCode, proposals, structureItems)
        toast(`Applied ${proposals.length} outline change${proposals.length === 1 ? '' : 's'}`)
        await onStructureChanged()
      } catch (e) {
        const msg = e instanceof Error ? e.message : 'Could not apply changes.'
        setError(msg)
        toastMutationError(msg)
        await onStructureChanged()
      } finally {
        setApplyingKey(null)
      }
    },
    [courseCode, onStructureChanged, structureItems],
  )

  if (!open) return null

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="fixed inset-y-0 end-0 z-50 flex w-full flex-col border-s border-border-default bg-surface-raised shadow-2xl dark:border-border-subtle dark:bg-surface-raised sm:w-96"
    >
      <div className="flex items-center justify-between border-b border-border-default px-4 py-3 dark:border-border-subtle">
        <div className="flex items-center gap-2">
          <Sparkles className="h-5 w-5 text-accent-fg" aria-hidden />
          <h2 id={titleId} className="font-semibold text-fg-default">
            Modules AI
          </h2>
        </div>
        <button
          type="button"
          aria-label="Close Modules AI"
          onClick={() => onOpenChange(false)}
          className="rounded p-1 text-fg-muted hover:bg-surface-sunken dark:hover:bg-surface-overlay"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      <div
        role="log"
        aria-live="polite"
        aria-label="Modules AI conversation"
        className="flex-1 overflow-y-auto px-4 py-4"
      >
        {messages.length === 0 && !busy && (
          <div className="rounded-xl bg-surface-base px-3 py-3 text-sm text-fg-muted dark:bg-surface-overlay dark:text-fg-muted">
            <p className="flex items-start gap-2">
              <Bot className="mt-0.5 h-4 w-4 shrink-0 text-accent-fg" aria-hidden />
              <span>
                Ask for outline changes — for example “Add a Week 3 module with a quiz” or “Rename
                Module 1 to Introduction”. Proposed changes appear for your review before they are
                applied.
              </span>
            </p>
          </div>
        )}
        {messages.map((msg, i) => (
          <div
            key={`${msg.role}-${i}`}
            className={`mb-3 flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
          >
            <div
              className={`max-w-[90%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed ${ msg.role === 'user' ? 'bg-accent-solid text-white' : 'bg-surface-sunken text-fg-default dark:bg-surface-overlay dark:text-fg-default' }`}
            >
              {msg.content}
              {msg.role === 'assistant' && msg.proposals && msg.proposals.length > 0 ? (
                <div className="mt-3 space-y-2 border-t border-slate-200/70 pt-2 dark:border-border-default">
                  <p className="text-xs font-medium text-fg-muted">
                    Proposed changes
                  </p>
                  <ul className="space-y-1.5">
                    {msg.proposals.map((p, pi) => {
                      const key = `${i}-${pi}`
                      return (
                        <li
                          key={key}
                          className="flex items-start justify-between gap-2 rounded-lg bg-white/70 px-2 py-1.5/50"
                        >
                          <span className="text-xs text-fg-default">
                            {describeModulesAiProposal(p)}
                          </span>
                          <button
                            type="button"
                            disabled={busy || applyingKey !== null}
                            onClick={() => void applyOne(p, key, msg.proposals ?? [])}
                            className="shrink-0 text-xs font-semibold text-accent-fg hover:underline disabled:opacity-50 dark:text-indigo-400"
                          >
                            {applyingKey === key ? '…' : 'Apply'}
                          </button>
                        </li>
                      )
                    })}
                  </ul>
                  {msg.proposals.length > 1 ? (
                    <button
                      type="button"
                      disabled={busy || applyingKey !== null}
                      onClick={() => void applyAll(msg.proposals!, i)}
                      className="text-xs font-semibold text-accent-fg hover:underline disabled:opacity-50 dark:text-indigo-400"
                    >
                      {applyingKey === `all-${i}` ? 'Applying…' : 'Apply all'}
                    </button>
                  ) : null}
                </div>
              ) : null}
            </div>
          </div>
        ))}
        {busy && (
          <p className="mb-3 text-sm text-fg-subtle">Thinking…</p>
        )}
        {error && (
          <div className="mb-3 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-800 dark:bg-rose-950 dark:text-rose-300">
            {error}
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      <div className="border-t border-border-default px-4 py-3 dark:border-border-subtle">
        <div className="flex items-end gap-2">
          <textarea
            ref={inputRef}
            aria-label="Describe outline changes"
            placeholder="Describe changes to the modules outline…"
            rows={2}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                void sendMessage()
              }
            }}
            disabled={busy}
            className="flex-1 resize-none rounded-xl border border-border-default bg-surface-base px-3 py-2 text-sm disabled:opacity-50 dark:border-border-default dark:bg-surface-overlay"
          />
          <button
            type="button"
            aria-label="Send message"
            onClick={() => void sendMessage()}
            disabled={!input.trim() || busy}
            className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-accent-solid text-white disabled:opacity-50"
          >
            <Send className="h-4 w-4" />
          </button>
        </div>
        <p className="mt-1.5 text-center text-xs text-fg-subtle">
          Proposals are reviewed before they change the course outline.
        </p>
      </div>
    </div>
  )
}
