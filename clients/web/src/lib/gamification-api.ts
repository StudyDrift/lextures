import { authorizedFetch } from './api'

export type GamificationBadge = {
  badgeType: string
  awardedAt: string
}

export type GamificationProfile = {
  xpTotal: number
  level: number
  xpToNextLevel: number
  levelProgressPct: number
  currentStreak: number
  longestStreak: number
  streakFreezes: number
  streakAtRisk: boolean
  streakHoursLeft?: number
  streakEnded?: boolean
  leaderboardVisible: boolean
  badges: GamificationBadge[]
  recentBadges: GamificationBadge[]
}

export type LeaderboardEntry = {
  rank: number
  userId: string
  displayName: string
  xpEarned: number
  isCurrentUser?: boolean
}

export type CourseLeaderboard = {
  topEntries: LeaderboardEntry[]
  currentUser?: LeaderboardEntry
}

export async function fetchGamificationProfile(): Promise<GamificationProfile> {
  const res = await authorizedFetch('/api/v1/me/gamification')
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(typeof raw === 'object' && raw && 'message' in raw ? String((raw as { message: string }).message) : 'Could not load gamification profile.')
  }
  return raw as GamificationProfile
}

export async function fetchCourseLeaderboard(courseCode: string): Promise<CourseLeaderboard> {
  const res = await authorizedFetch(`/api/v1/courses/${encodeURIComponent(courseCode)}/leaderboard`)
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error('Could not load leaderboard.')
  }
  return raw as CourseLeaderboard
}

export async function spendStreakFreeze(): Promise<GamificationProfile> {
  const res = await authorizedFetch('/api/v1/me/gamification/freeze-streak', { method: 'POST' })
  const raw: unknown = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(typeof raw === 'object' && raw && 'message' in raw ? String((raw as { message: string }).message) : 'Could not apply streak freeze.')
  }
  return raw as GamificationProfile
}

export const BADGE_LABELS: Record<string, string> = {
  streak_7: '7-day streak',
  streak_30: '30-day streak',
  xp_100: '100 XP',
  xp_1000: '1000 XP',
  first_course_complete: 'First course complete',
}

export function badgeLabel(badgeType: string): string {
  return BADGE_LABELS[badgeType] ?? badgeType.replace(/_/g, ' ')
}
