import { useSyncExternalStore } from 'react'
import { isLearnAsSession } from '../lib/auth'

function subscribe(onChange: () => void): () => void {
  window.addEventListener('studydrift-auth-token', onChange)
  window.addEventListener('storage', onChange)
  return () => {
    window.removeEventListener('studydrift-auth-token', onChange)
    window.removeEventListener('storage', onChange)
  }
}

/** Reactive {@link isLearnAsSession}: true while a parent is using Learn as a managed learner. */
export function useLearnAsSession(): boolean {
  return useSyncExternalStore(subscribe, isLearnAsSession, () => false)
}
