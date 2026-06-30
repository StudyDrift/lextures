import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { formatNumber } from '../lib/format'
import { Bot, Plus, Send, Trash2, X } from 'lucide-react'
import { authorizedFetch } from '../lib/api'
import { usePlatformFeatures } from '../context/platform-features-context'
import {
  createTutorSession,
  deleteTutorSession,
  fetchTutorSession,
  fetchTutorSessions,
  sendTutorSessionMessage,
  type TutorCitation,
  type TutorSessionMessage,
  type TutorSessionSummary,
} from '../lib/tutor-api'

const API_BASE = '/api/v1'

interface LegacyMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
  citations?: TutorCitation[]
}

interface ConversationState {
  conversationId: string
  messages: LegacyMessage[]
  tokensUsed: number
  tokenLimit: number
  periodMonth: string
}

interface AiTutorMenuProps {
  courseCode: string
}

function AiTutorTrigger({ open, onToggle }: { open: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      aria-label="Open AI Tutor"
      aria-expanded={open}
      aria-haspopup="dialog"
      onClick={onToggle}
      className={`relative inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-xl transition-[background-color,color,border-color] focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500/30 ${
        open
          ? 'bg-indigo-100 text-indigo-700 dark:bg-indigo-900/40 dark:text-indigo-300'
          : 'text-slate-600 hover:bg-slate-100 dark:text-neutral-300 dark:hover:bg-neutral-800'
      }`}
    >
      <Bot className="h-5 w-5" aria-hidden />
    </button>
  )
}

function CitationChips({ citations }: { citations: TutorCitation[] }) {
  if (!citations.length) return null
  return (
    <div className="mt-2 flex flex-wrap gap-1.5" aria-label="Source citations">
      {citations.map((c) => (
        <button
          key={`${c.sourceId}-${c.chunkId}`}
          type="button"
          title={c.excerpt}
          className="rounded-full border border-indigo-200 bg-indigo-50 px-2 py-0.5 text-xs text-indigo-700 hover:bg-indigo-100 dark:border-indigo-800 dark:bg-indigo-950 dark:text-indigo-300"
          onClick={() => {
            if (c.sourceId) {
              window.alert(`${c.title ?? 'Course material'}\n\n${c.excerpt}`)
            }
          }}
        >
          {c.title ?? 'Source'}
        </button>
      ))}
    </div>
  )
}

async function fetchLegacyConversation(courseCode: string): Promise<ConversationState> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/tutor/conversation`,
  )
  if (!res.ok) throw new Error(`Failed to load conversation: ${res.status}`)
  return res.json() as Promise<ConversationState>
}

async function resetLegacyConversation(courseCode: string): Promise<void> {
  const res = await authorizedFetch(
    `${API_BASE}/courses/${encodeURIComponent(courseCode)}/tutor/conversation`,
    { method: 'DELETE' },
  )
  if (!res.ok) throw new Error(`Failed to reset conversation: ${res.status}`)
}

export function AiTutorMenu({ courseCode }: AiTutorMenuProps) {
  const { ffPersistentTutor } = usePlatformFeatures()
  const persistent = ffPersistentTutor === true
  const disclosureKey = `tutor-disclosure-${courseCode}`

  const [open, setOpen] = useState(false)
  const [showDisclosure, setShowDisclosure] = useState(false)
  const [legacyConv, setLegacyConv] = useState<ConversationState | null>(null)
  const [sessions, setSessions] = useState<TutorSessionSummary[]>([])
  const [activeSessionId, setActiveSessionId] = useState<string | null>(null)
  const [messages, setMessages] = useState<TutorSessionMessage[]>([])
  const [loading, setLoading] = useState(false)
  const [streaming, setStreaming] = useState(false)
  const [input, setInput] = useState('')
  const [streamedText, setStreamedText] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [tokenBudget, setTokenBudget] = useState<{ used: number; limit: number; month: string } | null>(
    null,
  )
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [])

  useEffect(() => {
    scrollToBottom()
  }, [messages, legacyConv?.messages, streamedText, scrollToBottom])

  useEffect(() => {
    if (!open) return
    setShowDisclosure(localStorage.getItem(disclosureKey) !== '1')
  }, [open, disclosureKey])

  const loadPersistent = useCallback(async () => {
    const list = await fetchTutorSessions(courseCode)
    setSessions(list)
    if (list.length > 0) {
      const detail = await fetchTutorSession(courseCode, list[0].id)
      setActiveSessionId(detail.id)
      setMessages(detail.messages.filter((m) => m.role !== 'system'))
    } else {
      const created = await createTutorSession(courseCode)
      setSessions([created])
      setActiveSessionId(created.id)
      setMessages([])
    }
  }, [courseCode])

  const loadLegacy = useCallback(async () => {
    const conv = await fetchLegacyConversation(courseCode)
    setLegacyConv(conv)
    setTokenBudget({ used: conv.tokensUsed, limit: conv.tokenLimit, month: conv.periodMonth })
  }, [courseCode])

  useEffect(() => {
    if (!open) return
    setLoading(true)
    setError(null)
    void (async () => {
      try {
        if (persistent) {
          await loadPersistent()
        } else {
          await loadLegacy()
        }
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to load tutor.')
      } finally {
        setLoading(false)
      }
    })()
  }, [open, persistent, loadPersistent, loadLegacy])

  const dismissDisclosure = useCallback(() => {
    localStorage.setItem(disclosureKey, '1')
    setShowDisclosure(false)
  }, [disclosureKey])

  const switchSession = useCallback(
    async (sessionId: string) => {
      setLoading(true)
      setError(null)
      try {
        const detail = await fetchTutorSession(courseCode, sessionId)
        setActiveSessionId(detail.id)
        setMessages(detail.messages.filter((m) => m.role !== 'system'))
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to load session.')
      } finally {
        setLoading(false)
      }
    },
    [courseCode],
  )

  const startNewSession = useCallback(async () => {
    setLoading(true)
    try {
      const created = await createTutorSession(courseCode)
      setSessions((prev) => [created, ...prev])
      setActiveSessionId(created.id)
      setMessages([])
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to create session.')
    } finally {
      setLoading(false)
    }
  }, [courseCode])

  const sendMessage = useCallback(async () => {
    const text = input.trim()
    if (!text || streaming) return
    setInput('')
    setError(null)
    setStreaming(true)
    setStreamedText('')

    if (persistent && activeSessionId) {
      setMessages((prev) => [...prev, { id: `tmp-${Date.now()}`, role: 'user', content: text }])
      try {
        let fullText = ''
        let citations: TutorCitation[] = []
        await sendTutorSessionMessage(courseCode, activeSessionId, text, (ev) => {
          if (ev.type === 'content' && ev.text) {
            fullText += ev.text
            setStreamedText(fullText)
          } else if (ev.type === 'error') {
            setError(ev.message)
          } else if (ev.type === 'done') {
            citations = ev.citations ?? []
          }
        })
        if (fullText) {
          setMessages((prev) => [
            ...prev,
            { id: `assistant-${Date.now()}`, role: 'assistant', content: fullText, citations },
          ])
        }
        setStreamedText('')
        setSessions((prev) =>
          prev.map((s) => (s.id === activeSessionId ? { ...s, lastActive: new Date().toISOString() } : s)),
        )
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to send message.')
      } finally {
        setStreaming(false)
      }
      return
    }

    setLegacyConv((prev) =>
      prev ? { ...prev, messages: [...prev.messages, { role: 'user', content: text }] } : prev,
    )
    try {
      const res = await authorizedFetch(
        `${API_BASE}/courses/${encodeURIComponent(courseCode)}/tutor/message`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ message: text }),
        },
      )
      if (!res.ok || !res.body) {
        setError(await res.text())
        setStreaming(false)
        return
      }
      const reader = res.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''
      let fullText = ''
      while (true) {
        const { done, value } = await reader.read()
        if (done) break
        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''
        for (const line of lines) {
          if (!line.startsWith('data: ')) continue
          try {
            const ev = JSON.parse(line.slice('data: '.length)) as {
              type: string
              text?: string
              message?: string
            }
            if (ev.type === 'content' && ev.text) {
              fullText += ev.text
              setStreamedText(fullText)
            } else if (ev.type === 'error') {
              setError(ev.message ?? 'An error occurred.')
            } else if (ev.type === 'done') {
              setLegacyConv((prev) =>
                prev
                  ? { ...prev, messages: [...prev.messages, { role: 'assistant', content: fullText }] }
                  : prev,
              )
              setStreamedText('')
            }
          } catch {
            /* skip */
          }
        }
      }
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to send message.')
    } finally {
      setStreaming(false)
    }
  }, [activeSessionId, courseCode, input, persistent, streaming])

  const handleReset = useCallback(async () => {
    try {
      if (persistent && activeSessionId) {
        await deleteTutorSession(courseCode, activeSessionId)
        await startNewSession()
      } else {
        await resetLegacyConversation(courseCode)
        setLegacyConv((prev) => (prev ? { ...prev, messages: [] } : prev))
      }
      setStreamedText('')
      setError(null)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to reset.')
    }
  }, [activeSessionId, courseCode, persistent, startNewSession])

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        void sendMessage()
      }
    },
    [sendMessage],
  )

  const displayMessages: LegacyMessage[] = persistent
    ? messages.map((m) => ({
        role: m.role as 'user' | 'assistant' | 'system',
        content: m.content,
        citations: m.citations,
      }))
    : (legacyConv?.messages ?? [])

  const budgetPct = tokenBudget ? Math.min(100, (tokenBudget.used / tokenBudget.limit) * 100) : 0

  return (
    <>
      <AiTutorTrigger open={open} onToggle={() => setOpen((o) => !o)} />

      {open && (
        <div
          role="dialog"
          aria-label="AI Tutor"
          aria-modal="true"
          className="fixed inset-y-0 end-0 z-50 flex w-full flex-col bg-white shadow-2xl dark:bg-neutral-900 sm:w-96"
        >
          <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-800">
            <div className="flex items-center gap-2">
              <Bot className="h-5 w-5 text-indigo-600" />
              <span className="font-semibold text-slate-900 dark:text-neutral-100">AI Tutor</span>
            </div>
            <div className="flex items-center gap-2">
              {persistent && (
                <button
                  type="button"
                  aria-label="New tutor session"
                  onClick={() => void startNewSession()}
                  className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
                >
                  <Plus className="h-4 w-4" />
                </button>
              )}
              <button
                type="button"
                aria-label="Reset conversation"
                onClick={() => void handleReset()}
                className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
              >
                <Trash2 className="h-4 w-4" />
              </button>
              <button
                type="button"
                aria-label="Close AI Tutor"
                onClick={() => setOpen(false)}
                className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          </div>

          {persistent && sessions.length > 0 && (
            <div className="border-b border-slate-100 px-4 py-2 dark:border-neutral-800">
              <label htmlFor="tutor-session-select" className="sr-only">
                Previous sessions
              </label>
              <select
                id="tutor-session-select"
                value={activeSessionId ?? ''}
                onChange={(e) => void switchSession(e.target.value)}
                className="w-full rounded-lg border border-slate-200 bg-slate-50 px-2 py-1.5 text-sm dark:border-neutral-700 dark:bg-neutral-800"
              >
                {sessions.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.title?.trim() || `Session ${new Date(s.createdAt).toLocaleDateString()}`}
                  </option>
                ))}
              </select>
            </div>
          )}

          {showDisclosure && (
            <div className="border-b border-amber-100 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900 dark:bg-amber-950 dark:text-amber-200">
              <p>
                This AI tutor uses your course materials and conversation history.{' '}
                <Link to="/settings/privacy" className="underline">
                  Manage AI settings
                </Link>
              </p>
              <button
                type="button"
                className="mt-2 text-xs font-medium underline"
                onClick={dismissDisclosure}
              >
                Got it
              </button>
            </div>
          )}

          {!persistent && tokenBudget && (
            <div className="border-b border-slate-100 px-4 py-2 dark:border-neutral-800">
              <div className="mb-1 flex items-center justify-between text-xs text-slate-500">
                <span>
                  {formatNumber(tokenBudget.used)} / {formatNumber(tokenBudget.limit)} tokens used
                </span>
                <span>{tokenBudget.month}</span>
              </div>
              <div className="h-1.5 w-full overflow-hidden rounded-full bg-slate-100 dark:bg-neutral-800">
                <div className="h-full rounded-full bg-indigo-500" style={{ width: `${budgetPct}%` }} />
              </div>
            </div>
          )}

          <div
            role="log"
            aria-live="polite"
            aria-label="Tutor conversation"
            className="flex-1 overflow-y-auto px-4 py-4"
          >
            {loading && (
              <p className="text-sm text-slate-400 dark:text-neutral-500">Loading conversation…</p>
            )}
            {!loading && displayMessages.length === 0 && !streamedText && (
              <p className="text-center text-sm text-slate-400 dark:text-neutral-500">
                Ask the AI tutor a question about this course.
              </p>
            )}
            {displayMessages.map((msg, i) => (
              <div
                key={i}
                className={`mb-3 flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}
              >
                <div
                  className={`max-w-[85%] rounded-2xl px-4 py-2.5 text-sm leading-relaxed ${
                    msg.role === 'user'
                      ? 'bg-indigo-600 text-white'
                      : 'bg-slate-100 text-slate-900 dark:bg-neutral-800 dark:text-neutral-100'
                  }`}
                >
                  {msg.content}
                  {msg.role === 'assistant' && msg.citations ? (
                    <CitationChips citations={msg.citations} />
                  ) : null}
                </div>
              </div>
            ))}
            {streamedText && (
              <div className="mb-3 flex justify-start">
                <div className="max-w-[85%] rounded-2xl bg-slate-100 px-4 py-2.5 text-sm dark:bg-neutral-800">
                  {streamedText}
                  <span className="ms-0.5 inline-block h-3 w-0.5 animate-pulse bg-current" />
                </div>
              </div>
            )}
            {error && (
              <div className="mb-3 rounded-lg border border-rose-200 bg-rose-50 px-3 py-2 text-sm text-rose-700 dark:border-rose-800 dark:bg-rose-950 dark:text-rose-300">
                {error}
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>

          <div className="border-t border-slate-200 px-4 py-3 dark:border-neutral-800">
            <div className="flex items-end gap-2">
              <textarea
                ref={inputRef}
                aria-label="Type your message"
                placeholder="Ask the tutor…"
                rows={1}
                value={input}
                onChange={(e) => setInput(e.target.value)}
                onKeyDown={handleKeyDown}
                disabled={streaming}
                className="flex-1 resize-none rounded-xl border border-slate-200 bg-slate-50 px-3 py-2 text-sm disabled:opacity-50 dark:border-neutral-700 dark:bg-neutral-800"
              />
              <button
                type="button"
                aria-label="Send message"
                onClick={() => void sendMessage()}
                disabled={!input.trim() || streaming}
                className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-indigo-600 text-white disabled:opacity-50"
              >
                <Send className="h-4 w-4" />
              </button>
            </div>
            <p className="mt-1.5 text-center text-xs text-slate-400 dark:text-neutral-500">
              I am an AI tutor. I can make mistakes — please verify important information with your
              instructor.
            </p>
          </div>
        </div>
      )}
    </>
  )
}
