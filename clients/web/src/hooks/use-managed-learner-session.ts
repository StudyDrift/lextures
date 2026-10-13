import { useEffect, useState } from 'react'
import { isManagedLearnerSession } from '../lib/managed-learner-session'

/**
 * True while a managed learner (kid) session is the active bearer (#775).
 * Re-reads on `studydrift-auth-token` so the shell updates without a reload.
 */
export function useManagedLearnerSession(): boolean {
  const [tokenVersion, setTokenVersion] = useState(0)

  useEffect(() => {
    function onAuth() {
      setTokenVersion((version) => version + 1)
    }
    window.addEventListener('studydrift-auth-token', onAuth)
    return () => window.removeEventListener('studydrift-auth-token', onAuth)
  }, [])

  void tokenVersion
  return isManagedLearnerSession()
}
