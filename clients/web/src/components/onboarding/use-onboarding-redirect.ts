import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { getAccountType } from '../../lib/auth'
import { fetchOnboardingStatus } from '../../lib/onboarding-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

/** Redirects new homeschool learners to onboarding when the feature flag is on and flow is incomplete. */
export function useOnboardingRedirect(): { checking: boolean; shouldRedirect: boolean } {
  const { ffOnboardingFlow, loading: featuresLoading } = usePlatformFeatures()
  const location = useLocation()
  const [checking, setChecking] = useState(true)
  const [shouldRedirect, setShouldRedirect] = useState(false)

  useEffect(() => {
    if (featuresLoading) {
      setShouldRedirect(false)
      setChecking(false)
      return
    }
    if (!ffOnboardingFlow || getAccountType() === 'parent') {
      setShouldRedirect(false)
      setChecking(false)
      return
    }
    if (location.pathname.startsWith('/onboarding')) {
      setShouldRedirect(false)
      setChecking(false)
      return
    }
    let cancelled = false
    void fetchOnboardingStatus()
      .then((status) => {
        if (cancelled) return
        setShouldRedirect(Boolean(status && !status.completed))
      })
      .catch(() => {
        if (!cancelled) setShouldRedirect(false)
      })
      .finally(() => {
        if (!cancelled) setChecking(false)
      })
    return () => {
      cancelled = true
    }
  }, [featuresLoading, ffOnboardingFlow, location.pathname])

  return { checking, shouldRedirect }
}
