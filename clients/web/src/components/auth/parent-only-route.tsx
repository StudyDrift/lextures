import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useLearnAsSession } from '../../hooks/use-learn-as-session'

/**
 * Wraps routes the server refuses while a parent is in a Learn as session (dependents, billing),
 * so a child lands on the dashboard instead of an error page.
 */
export function ParentOnlyRoute({ children }: { children: ReactNode }) {
  const learningAs = useLearnAsSession()
  if (learningAs) {
    return <Navigate to="/" replace />
  }
  return <>{children}</>
}
