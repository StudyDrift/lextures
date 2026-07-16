import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearPlayerSession,
  loadPlayerSession,
  savePlayerSession,
} from '../live-quiz-player-storage'

describe('live-quiz-player-storage', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('round-trips a player session', () => {
    savePlayerSession({
      gameId: 'g1',
      courseCode: 'CS101',
      playerId: 'p1',
      playerToken: 'tok',
      nickname: 'Ada',
    })
    expect(loadPlayerSession('g1')).toMatchObject({
      gameId: 'g1',
      playerToken: 'tok',
      nickname: 'Ada',
    })
    clearPlayerSession('g1')
    expect(loadPlayerSession('g1')).toBeNull()
  })
})
