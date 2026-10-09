import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useImpersonatingSession } from '../hooks/use-impersonating-session'

/** Parent-only pages the API rejects for a Learn-as or view-as token. */
export function BlockDuringImpersonation({ children }: { children: ReactNode }) {
  const impersonating = useImpersonatingSession()
  if (impersonating) return <Navigate to="/" replace />
  return children
}
