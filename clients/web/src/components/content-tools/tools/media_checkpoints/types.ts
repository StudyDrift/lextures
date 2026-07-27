export type QuestionType = 'single' | 'multi' | 'true_false' | 'short_text' | 'numeric'

export type Option = {
  id: string
  text: string
  correct?: boolean
  feedback?: string
}

export type CheckpointQuestion = {
  type: QuestionType
  prompt: string
  options?: Option[]
  acceptedAnswers?: string[]
  correctValue?: number
  tolerance?: { kind: 'absolute' | 'relative'; value: number }
}

export type Checkpoint = {
  id: string
  atSec: number
  question: CheckpointQuestion
  required?: boolean
  attempts?: number
  showFeedback?: boolean
}

export type MediaRef = {
  source?: 'course_file' | 'external'
  fileId?: string
  kind?: 'video' | 'audio'
  durationSec?: number
  url?: string
  captionUrl?: string
}

export type MediaCheckpointsConfig = {
  media?: MediaRef
  captionsTrackId?: string
  transcriptSource?: 'captions' | 'inline'
  transcriptMarkdown?: string
  checkpoints?: Checkpoint[]
  preventSkipPastUnanswered?: boolean
  practiceOnly?: boolean
}

export type Attempt = { value: unknown; correct: boolean; at: string }
export type CheckpointAnswer = { attempts?: Attempt[]; done?: boolean }

export type MediaCheckpointsState = {
  v?: number
  answers?: Record<string, CheckpointAnswer>
  watchedSegments?: [number, number][]
  furthestSec?: number
  usedTranscriptOnly?: boolean
  scoreRaw?: number
  scoreMax?: number
  completedAt?: string
}

export type AnswerResult = {
  correct?: boolean
  feedback?: string
  attemptsRemaining?: number
  done?: boolean
  error?: string
  message?: string
  checkpointId?: string
}

export type TranscriptLine = {
  id: string
  atSec: number
  text: string
}

export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export function formatTimestamp(sec: number): string {
  const s = Math.max(0, Math.floor(sec))
  const m = Math.floor(s / 60)
  const r = s % 60
  return `${m}:${r.toString().padStart(2, '0')}`
}

/** Parse simple transcript markdown lines like "0:30 Concept" or "1:02:15 Wrap". */
export function parseTranscriptMarkdown(md: string): TranscriptLine[] {
  const lines = (md || '').split(/\r?\n/)
  const out: TranscriptLine[] = []
  for (let i = 0; i < lines.length; i++) {
    const raw = lines[i]!.trim()
    if (!raw) continue
    const m = raw.match(/^(\d{1,2}):(\d{2})(?::(\d{2}))?\s+(.+)$/)
    if (!m) {
      out.push({ id: `line-${i}`, atSec: 0, text: raw })
      continue
    }
    const a = Number(m[1])
    const b = Number(m[2])
    const c = m[3] != null ? Number(m[3]) : null
    const atSec = c == null ? a * 60 + b : a * 3600 + b * 60 + c
    out.push({ id: `line-${i}`, atSec, text: m[4]!.trim() })
  }
  return out
}

export function checkpointRequired(cp: Checkpoint): boolean {
  return cp.required !== false
}

export function checkpointAttempts(cp: Checkpoint): number {
  const n = cp.attempts ?? 2
  if (n < 1) return 2
  if (n > 10) return 10
  return n
}

export function isCheckpointDone(
  answers: Record<string, CheckpointAnswer>,
  cp: Checkpoint,
): boolean {
  const ans = answers[cp.id]
  if (!ans) return false
  if (ans.done) return true
  const attempts = ans.attempts ?? []
  if (attempts.length === 0) return false
  const last = attempts[attempts.length - 1]!
  if (last.correct) return true
  return attempts.length >= checkpointAttempts(cp)
}
