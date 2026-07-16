export type StoredPlayerSession = {
  gameId: string
  courseCode: string
  playerId: string
  playerToken: string
  nickname: string
  joinCode?: string
}

const KEY_PREFIX = 'lextures.liveQuiz.player.'

function keyFor(gameId: string): string {
  return `${KEY_PREFIX}${gameId}`
}

export function savePlayerSession(session: StoredPlayerSession): void {
  try {
    sessionStorage.setItem(keyFor(session.gameId), JSON.stringify(session))
  } catch {
    // private mode / quota — reconnect will require re-join
  }
}

export function loadPlayerSession(gameId: string): StoredPlayerSession | null {
  try {
    const raw = sessionStorage.getItem(keyFor(gameId))
    if (!raw) return null
    const parsed = JSON.parse(raw) as StoredPlayerSession
    if (!parsed?.gameId || !parsed.playerToken || !parsed.courseCode) return null
    return parsed
  } catch {
    return null
  }
}

export function clearPlayerSession(gameId: string): void {
  try {
    sessionStorage.removeItem(keyFor(gameId))
  } catch {
    // ignore
  }
}
