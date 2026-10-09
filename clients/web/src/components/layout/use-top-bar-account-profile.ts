import { useEffect, useState } from 'react'
import { authorizedFetch } from '../../lib/api'
import { getImpersonationToken } from '../../lib/auth'
import { parseAccountProfile, type TopBarAccountProfile } from './top-bar-utils'

/**
 * Account chip for the top bar. Clears the cached name as soon as the
 * Learn-as or view-as token changes so the previous person's name does not
 * linger until the refetch returns. Ordinary access-token refresh keeps the
 * current name on screen.
 */
export function useTopBarAccountProfile(): TopBarAccountProfile | null {
  const [profile, setProfile] = useState<TopBarAccountProfile | null>(null)

  useEffect(() => {
    let cancelled = false
    let request = 0
    let impersonationToken = getImpersonationToken()

    async function loadProfile() {
      const id = ++request
      try {
        const res = await authorizedFetch('/api/v1/settings/account')
        const raw: unknown = await res.json().catch(() => ({}))
        if (!res.ok || cancelled || id !== request) return
        setProfile(parseAccountProfile(raw))
      } catch {
        if (!cancelled && id === request) setProfile(null)
      }
    }

    function onProfileUpdated() {
      void loadProfile()
    }

    function onAuthToken() {
      const next = getImpersonationToken()
      if (next !== impersonationToken) {
        impersonationToken = next
        setProfile(null)
      }
      void loadProfile()
    }

    void loadProfile()
    window.addEventListener('studydrift-profile-updated', onProfileUpdated)
    window.addEventListener('studydrift-auth-token', onAuthToken)
    return () => {
      cancelled = true
      window.removeEventListener('studydrift-profile-updated', onProfileUpdated)
      window.removeEventListener('studydrift-auth-token', onAuthToken)
    }
  }, [])

  return profile
}
