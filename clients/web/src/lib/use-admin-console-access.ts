import { useEffect, useState } from 'react'
import { fetchAdminConsoleCapabilities } from './admin-console-api'

export function useAdminConsoleAccess(): { loading: boolean; canAccess: boolean } {
  const [loading, setLoading] = useState(true)
  const [canAccess, setCanAccess] = useState(false)

  useEffect(() => {
    let cancelled = false
    void fetchAdminConsoleCapabilities()
      .then((c) => {
        if (!cancelled) {
          setCanAccess(c.enabled && c.canAccess)
          setLoading(false)
        }
      })
      .catch(() => {
        if (!cancelled) {
          setCanAccess(false)
          setLoading(false)
        }
      })
    return () => {
      cancelled = true
    }
  }, [])

  return { loading, canAccess }
}
