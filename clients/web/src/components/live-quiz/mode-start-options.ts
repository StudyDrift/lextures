import type { LiveGameMode, PacedConfig, TeamConfig } from '../../lib/live-quiz-api'

export type ModeStartOptions = {
  mode: Exclude<LiveGameMode, 'homework'>
  teamConfig: TeamConfig
  pacedConfig: PacedConfig
}

export function defaultModeStartOptions(): ModeStartOptions {
  return {
    mode: 'live_classic',
    teamConfig: {
      teamCount: 4,
      aggregate: 'average',
      answerRule: 'each_member_answers',
      autoBalance: true,
    },
    pacedConfig: {
      shuffle: true,
      timeBudgetSeconds: 0,
      perQuestionTimers: true,
      liveLeaderboard: false,
    },
  }
}
