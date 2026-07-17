import type { LiveGameScoringConfig, LiveGameScoringProfile, LeaderboardPrivacy } from '../../lib/live-quiz-api'

export type ScoringStartOptions = {
  scoringProfile: LiveGameScoringProfile
  scoringConfig: LiveGameScoringConfig
  leaderboardPrivacy: LeaderboardPrivacy
  powerUpsEnabled: boolean
}

export const defaultCustomScoringConfig: LiveGameScoringConfig = {
  base: 1000,
  speedWeight: 1,
  streakStep: 100,
  streakCap: 5,
}

export function defaultScoringStartOptions(): ScoringStartOptions {
  return {
    scoringProfile: 'competitive',
    scoringConfig: {},
    leaderboardPrivacy: 'names',
    powerUpsEnabled: false,
  }
}
