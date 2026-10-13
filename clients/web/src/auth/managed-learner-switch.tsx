import type { ReactNode } from 'react'
import { useManagedLearnerSession } from '../hooks/use-managed-learner-session'

/**
 * Renders `kid` instead of `children` while a managed learner (kid) session is active (#775),
 * e.g. the simplified Settings page in place of the parent-owned account settings.
 */
export function ManagedLearnerSwitch({ kid, children }: { kid: ReactNode; children: ReactNode }) {
  const managedLearner = useManagedLearnerSession()
  return managedLearner ? kid : children
}
