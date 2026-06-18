import { useCallback, useEffect, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { Bot, Send, Sparkles, X } from 'lucide-react'
import { AiDisclosureBanner } from '../ai-disclosure-banner'
import { usePlatformFeatures } from '../../context/platform-features-context'
import { learnerCourseItemHref } from '../../lib/courses-api'
import {
  fetchAiProcessingOptOut,
  fetchStudyBuddyPrompts,
  sendStudyBuddyMessage,
  type StudyBuddyCitation,
  type StudyBuddyPrompt,
} from '../../lib/study-buddy-api'

type ChatMessage = {
  role: 'user' | 'assistant'
  content: string
  citations?: StudyBuddyCitation[]
}

type StudyBuddyWidgetProps = {
  courseCode: string
  /** When true, render as a fixed bottom-right floating action button (course pages). */
  floating?: boolean
}

export function StudyBuddyWidget({ courseCode, floating = true }: StudyBuddyWidgetProps) {
  const { aiStudyBuddyEnabled, loading: featuresLoading } = usePlatformFeatures()
  const [open, setOpen] = useState(false)
  const [optOut, setOptOut] = useState(false)
  const [optOutLoaded, setOptOutLoaded] = useState(false)
  const [prompts, setPrompts] = useState<StudyBuddyPrompt[]>([])
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [sessionId, setSessionId] = useState<string | null>(null)
  const [input, setInput] = useState('')
  const [streaming, setStreaming] = useState(false)
  const [streamedText, setStreamedText] = useState('')
  const [error, setError] = useState<string | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    let cancelled = false
    void fetchAiProcessingOptOut()
      .then((v) => {
        if (!cancelled) setOptOut(v)
      })
      .finally(() => {
        if (!cancelled) setOptOutLoaded(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!aiStudyBuddyEnabled || optOut) return
    let cancelled = false
    void fetchStudyBuddyPrompts(courseCode)
      .then((p) => {
        if (!cancelled) setPrompts(p)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [aiStudyBuddyEnabled, courseCode, optOut])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, streamedText])

  const sendMessage = useCallback(
    async (textOverride?: string) => {
      const text = (textOverride ?? input).trim()
      if (!text || streaming || optOut) return
      setInput('')
      setError(null)
      setStreaming(true)
      setStreamedText('')
      setMessages((prev) => [...prev, { role: 'user', content: text }])

      try {
        let fullText = ''
        await sendStudyBuddyMessage(courseCode, text, sessionId, (ev) => {
          if (ev.type === 'content' && ev.text) {
            fullText += ev.text
            setStreamedText(fullText)
          } else if (ev.type === 'error') {
            setError(ev.message)
          } else if (ev.type === 'done') {
            setSessionId(ev.sessionId)
            setMessages((prev) => [
              ...prev,
              {
                role: 'assistant',
                content: fullText,
                citations: ev.citations,
              },
            ])
            setStreamedText('')
          }
        })
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : 'Failed to send message.')
      } finally {
        setStreaming(false)
      }
    },
    [courseCode, input, optOut, sessionId, streaming],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        void sendMessage()
      }
    },
    [sendMessage],
  )

  if (featuresLoading || !aiStudyBuddyEnabled) return null

  const disabled = optOutLoaded && optOut

  const panel = open ? (
    <div
      role="dialog"
      aria-label="AI Study Buddy"
      aria-modal="true"
      className={
        floating
          ? 'fixed inset-x-0 bottom-0 z-50 flex max-h-[85vh] flex-col rounded-t-2xl border border-slate-200 bg-white shadow-2xl dark:border-neutral-700 dark:bg-neutral-900 sm:inset-auto sm:bottom-6 sm:end-6 sm:h-[32rem] sm:w-96 sm:rounded-2xl'
          : 'flex h-[28rem] flex-col rounded-2xl border border-slate-200 bg-white shadow-lg dark:border-neutral-700 dark:bg-neutral-900'
      }
    >
      <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-neutral-800">
        <div className="flex items-center gap-2">
          <Sparkles className="h-5 w-5 text-violet-600" aria-hidden />
          <div>
            <span className="font-semibold text-slate-900 dark:text-neutral-100">Study Buddy</span>
            <span className="ms-2 rounded-full bg-violet-100 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-violet-800 dark:bg-violet-950 dark:text-violet-200">
              AI
            </span>
          </div>
        </div>
        <button
          type="button"
          aria-label="Close study buddy"
          onClick={() => setOpen(false)}
          className="rounded p-1 text-slate-500 hover:bg-slate-100 dark:hover:bg-neutral-800"
        >
          <X className="h-5 w-5" />
        </button>
      </div>

      <div className="flex-1 overflow-y-auto px-4 py-3">
        <AiDisclosureBanner featureKey="ai_study_buddy" />
        {disabled ? (
          <p className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-100">
            AI features disabled —{' '}
            <Link to="/settings/account" className="font-semibold underline underline-offset-2">
              enable in settings
            </Link>
            .
          </p>
        ) : null}

        {!disabled && prompts.length > 0 ? (
          <div className="mb-3 space-y-2" aria-label="Study suggestions">
            {prompts.slice(0, 2).map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => void sendMessage(p.message.replace(' — want to review?', '?').replace(' — want a quick quiz?', '?'))}
                className="block w-full rounded-xl border border-violet-100 bg-violet-50/80 px-3 py-2 text-left text-sm text-violet-950 hover:bg-violet-100 dark:border-violet-900/40 dark:bg-violet-950/30 dark:text-violet-100"
              >
                {p.message}
              </button>
            ))}
          </div>
        ) : null}

        {messages.length === 0 && !streaming && !disabled ? (
          <p className="text-sm text-slate-600 dark:text-neutral-400">
            Ask me anything about your course! I&apos;m an AI — I use your course materials and learning
            goals to help you study.
          </p>
        ) : null}

        <div className="space-y-3" aria-live="polite">
          {messages.map((m, idx) => (
            <div
              key={idx}
              className={
                m.role === 'user'
                  ? 'ms-8 rounded-2xl bg-indigo-600 px-3 py-2 text-sm text-white'
                  : 'me-4 rounded-2xl bg-slate-100 px-3 py-2 text-sm text-slate-900 dark:bg-neutral-800 dark:text-neutral-100'
              }
              role={m.role === 'assistant' ? 'status' : undefined}
            >
              <p className="whitespace-pre-wrap">{m.content}</p>
              {m.citations && m.citations.length > 0 ? (
                <ul className="mt-2 space-y-1 border-t border-slate-200 pt-2 text-xs dark:border-neutral-700">
                  {m.citations.map((c) => (
                    <li key={`${c.itemId}-${c.title}`}>
                      {c.itemId ? (
                        <Link
                          to={learnerCourseItemHref(courseCode, { kind: 'content_page', id: c.itemId })}
                          className="font-medium text-indigo-700 underline underline-offset-2 dark:text-indigo-300"
                        >
                          {c.title}
                        </Link>
                      ) : (
                        <span className="font-medium">{c.title}</span>
                      )}
                      {c.excerpt ? <span className="text-slate-500"> — {c.excerpt}</span> : null}
                    </li>
                  ))}
                </ul>
              ) : null}
            </div>
          ))}
          {streamedText ? (
            <div
              className="me-4 rounded-2xl bg-slate-100 px-3 py-2 text-sm text-slate-900 dark:bg-neutral-800 dark:text-neutral-100"
              role="status"
              aria-live="polite"
            >
              <p className="whitespace-pre-wrap">{streamedText}</p>
            </div>
          ) : null}
        </div>
        <div ref={messagesEndRef} />
      </div>

      {!disabled ? (
        <div className="border-t border-slate-200 p-3 dark:border-neutral-800">
          {error ? <p className="mb-2 text-xs text-rose-600 dark:text-rose-400">{error}</p> : null}
          <div className="flex items-end gap-2">
            <textarea
              ref={inputRef}
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              aria-label="Message the study buddy"
              placeholder="Ask about your course…"
              rows={2}
              disabled={streaming}
              className="min-h-[2.5rem] flex-1 resize-none rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900"
            />
            <button
              type="button"
              aria-label="Send message"
              disabled={streaming || !input.trim()}
              onClick={() => void sendMessage()}
              className="inline-flex h-10 w-10 items-center justify-center rounded-xl bg-violet-600 text-white hover:bg-violet-500 disabled:opacity-50"
            >
              <Send className="h-4 w-4" />
            </button>
          </div>
        </div>
      ) : null}
    </div>
  ) : null

  if (!floating) {
    return (
      <div className="relative">
        {panel}
        <button
          type="button"
          onClick={() => setOpen((o) => !o)}
          className="inline-flex items-center gap-2 rounded-xl bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-500"
        >
          <Bot className="h-4 w-4" aria-hidden />
          Open study buddy
        </button>
      </div>
    )
  }

  return (
    <>
      {panel}
      <button
        type="button"
        aria-label="Open AI study buddy"
        aria-expanded={open}
        aria-haspopup="dialog"
        onClick={() => setOpen((o) => !o)}
        className="fixed bottom-6 end-6 z-40 inline-flex h-14 w-14 items-center justify-center rounded-full bg-violet-600 text-white shadow-lg hover:bg-violet-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-violet-400"
      >
        <Sparkles className="h-6 w-6" aria-hidden />
      </button>
    </>
  )
}

export function StudyBuddyPromptsCard({ courseCode }: { courseCode: string }) {
  const { aiStudyBuddyEnabled, loading } = usePlatformFeatures()
  const [prompts, setPrompts] = useState<StudyBuddyPrompt[]>([])

  useEffect(() => {
    if (loading || !aiStudyBuddyEnabled) return
    let cancelled = false
    void fetchStudyBuddyPrompts(courseCode)
      .then((p) => {
        if (!cancelled) setPrompts(p)
      })
      .catch(() => {})
    return () => {
      cancelled = true
    }
  }, [aiStudyBuddyEnabled, courseCode, loading])

  if (loading || !aiStudyBuddyEnabled || prompts.length === 0) return null

  return (
    <section aria-label="Study buddy suggestions" className="rounded-2xl border border-violet-100 bg-violet-50/80 px-5 py-4 dark:border-violet-900/40 dark:bg-violet-950/30">
      <div className="flex items-start gap-3">
        <Sparkles className="mt-0.5 h-5 w-5 shrink-0 text-violet-600" aria-hidden />
        <div className="min-w-0 flex-1">
          <p className="text-sm font-semibold text-slate-900 dark:text-neutral-100">Ready for a review?</p>
          <ul className="mt-2 space-y-1 text-sm text-slate-700 dark:text-neutral-300">
            {prompts.slice(0, 3).map((p) => (
              <li key={p.id}>{p.message}</li>
            ))}
          </ul>
          <p className="mt-2 text-xs text-slate-500 dark:text-neutral-400">
            Open the study buddy chat (bottom-right) to ask questions grounded in your course materials.
          </p>
        </div>
      </div>
    </section>
  )
}
