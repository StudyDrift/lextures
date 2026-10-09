import { useEffect, useState } from 'react'
import { isImpersonating } from '../lib/impersonation'

/**
 * True while an admin view-as or Learn-as token is the active bearer.
 * Re-reads on `studydrift-auth-token` so the shell updates without a reload.
 */
export function useImpersonatingSession(): boolean {
  const [tokenVersion, setTokenVersion] = useState(0)

  useEffect(() => {
    function onAuth() {
      setTokenVersion((version) => version + 1)
    }
    window.addEventListener('studydrift-auth-token', onAuth)
    return () => window.removeEventListener('studydrift-auth-token', onAuth)
  }, [])

  // Read the token on every render so a navigation in the same turn as
  // endImpersonationSession sees the restored parent session.
  void tokenVersion
  return isImpersonating()
}
