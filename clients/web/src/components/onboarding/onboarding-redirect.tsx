import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useOnboardingRedirect } from './use-onboarding-redirect'

export function OnboardingRedirect({ children }: { children: ReactNode }) {
  const { checking, shouldRedirect } = useOnboardingRedirect()
  if (checking) return null
  if (shouldRedirect) return <Navigate to="/onboarding" replace />
  return <>{children}</>
}
