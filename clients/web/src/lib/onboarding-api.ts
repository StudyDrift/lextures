import { authorizedFetch } from './api'
import { readApiErrorMessage } from './errors'

export type PriorKnowledgeLevel = 'beginner' | 'intermediate' | 'advanced'

export type LearnerGoals = {
  id: string
  userId: string
  topic: string
  goalText?: string | null
  targetDate?: string | null
  dailyMinutes: number
  priorKnowledgeLevel: PriorKnowledgeLevel
  diagnosticScore?: number | null
  diagnosticSkipped: boolean
  onboardingStep: number
  onboardingCompleted: boolean
  reminderOptIn: boolean
  reminderTime?: string | null
  recommendedCourseCode?: string | null
  recommendedCourseTitle?: string | null
  createdAt: string
  updatedAt: string
}

export type OnboardingStatus = {
  completed: boolean
  step: number
  shouldShowFlow: boolean
}

export type DiagnosticQuestion = {
  id: string
  prompt: string
  choices: string[]
}

export const ONBOARDING_TOPICS = [
  { id: 'python', label: 'Python' },
  { id: 'javascript', label: 'JavaScript' },
  { id: 'data-science', label: 'Data Science' },
  { id: 'math', label: 'Math' },
  { id: 'business', label: 'Business' },
  { id: 'design', label: 'Design' },
] as const

export async function fetchOnboardingStatus(): Promise<OnboardingStatus | null> {
  const res = await authorizedFetch('/api/v1/me/onboarding-status')
  if (res.status === 404) return null
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  return raw as OnboardingStatus
}

export async function fetchLearnerGoals(): Promise<LearnerGoals | null> {
  const res = await authorizedFetch('/api/v1/me/goals')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { goals?: LearnerGoals | null }
  return data.goals ?? null
}

export async function postOnboarding(body: Record<string, unknown>): Promise<LearnerGoals> {
  const res = await authorizedFetch('/api/v1/me/onboarding', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { goals: LearnerGoals }
  return data.goals
}

export async function patchLearnerGoals(body: Record<string, unknown>): Promise<LearnerGoals> {
  const res = await authorizedFetch('/api/v1/me/goals', {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { goals: LearnerGoals }
  return data.goals
}

export async function fetchDiagnosticQuestions(topic: string): Promise<DiagnosticQuestion[]> {
  const res = await authorizedFetch(
    `/api/v1/me/onboarding/diagnostic-questions?topic=${encodeURIComponent(topic)}`,
  )
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(readApiErrorMessage(raw))
  const data = raw as { questions: DiagnosticQuestion[] }
  return data.questions ?? []
}

export async function grantMarketingConsent(): Promise<void> {
  const res = await authorizedFetch('/api/v1/compliance/gdpr/consents', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      purpose: 'marketing',
      lawfulBasis: 'consent',
      consentVersion: '1.0',
    }),
  })
  if (res.status === 404) return
  if (!res.ok) {
    const raw: unknown = await res.json().catch(() => ({}))
    throw new Error(readApiErrorMessage(raw))
  }
}

export async function saveStudyReminderPrefs(optIn: boolean, reminderTime: string): Promise<void> {
  if (!optIn) return
  await authorizedFetch('/api/v1/me/notification-preferences', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      preferences: [
        { eventType: 'study_reminder', emailEnabled: true, pushEnabled: true, digestMode: 'instant' },
      ],
    }),
  }).catch(() => {
    /* 15.10 scheduler not shipped; prefs row is still useful */
  })
  void reminderTime
}
