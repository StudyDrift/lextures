import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { authHandoffHref } from '../../lib/post-auth-redirect'
import { useOnboardingRedirect } from './use-onboarding-redirect'

export function OnboardingRedirect({ children }: { children: ReactNode }) {
  const { checking, shouldRedirect } = useOnboardingRedirect()
  const location = useLocation()
  if (checking) return null
  if (shouldRedirect) {
    const from = `${location.pathname}${location.search}${location.hash}`
    return <Navigate to={authHandoffHref('/onboarding', from)} replace state={{ from }} />
  }
  return <>{children}</>
}
