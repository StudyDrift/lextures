import { useEffect, useState } from 'react'
import { getImpersonationToken } from '../lib/auth'
import { fetchMeProfile } from '../lib/impersonation'

/** Padding offset when the impersonation banner is visible. */
export function useImpersonationBannerOffset(): string {
  const [offset, setOffset] = useState('')

  useEffect(() => {
    let cancelled = false
    async function check() {
      if (!getImpersonationToken()) {
        if (!cancelled) setOffset('')
        return
      }
      const me = await fetchMeProfile()
      if (!cancelled) {
        setOffset(me?.impersonating ? 'pt-11' : '')
      }
    }
    void check()
    function onAuthChange() {
      void check()
    }
    window.addEventListener('studydrift-auth-token', onAuthChange)
    return () => {
      cancelled = true
      window.removeEventListener('studydrift-auth-token', onAuthChange)
    }
  }, [])

  return offset
}
