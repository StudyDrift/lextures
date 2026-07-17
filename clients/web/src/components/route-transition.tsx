import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import { useLocation, useNavigationType } from 'react-router-dom'
import { usePlatformFeatures } from '../context/platform-features-context'
import { usePrefersReducedMotion } from '../lib/motion'
import { resolveNavIntent, routeTransitionSpec, type NavIntent } from '../lib/route-transition'

type RouteTransitionProps = {
  children: ReactNode
  /** When true, prefer crossfade even for hierarchical path changes (wide layouts). */
  preferCrossfade?: boolean
}

/**
 * AN.2 — Wraps route content with a directional enter animation.
 * Uses CSS classes from AN.1 tokens (View Transitions optional enhancement).
 * Gated by `ffMotionNavigation`; reduced motion → ≤100ms fade.
 */
export function RouteTransition({ children, preferCrossfade = false }: RouteTransitionProps) {
  const location = useLocation()
  const navType = useNavigationType()
  const { ffMotionNavigation } = usePlatformFeatures()
  const reduceMotion = usePrefersReducedMotion()
  const prevPathRef = useRef(location.pathname)
  const [animClass, setAnimClass] = useState('')
  const [intent, setIntent] = useState<NavIntent>('replace')

  useLayoutEffect(() => {
    const from = prevPathRef.current
    const to = location.pathname
    if (from !== to) {
      let nextIntent = resolveNavIntent(from, to, {
        replace: navType === 'REPLACE',
        historyDelta: navType === 'POP' ? -1 : navType === 'PUSH' ? 1 : 0,
      })
      if (preferCrossfade && (nextIntent === 'forward' || nextIntent === 'back')) {
        nextIntent = 'lateral'
      }
      setIntent(nextIntent)
      prevPathRef.current = to

      if (!ffMotionNavigation) {
        setAnimClass('')
        return
      }
      const spec = routeTransitionSpec({
        intent: nextIntent,
        reduceMotion,
        enabled: ffMotionNavigation,
      })
      if (spec.durationMs <= 0) {
        setAnimClass('')
        return
      }
      setAnimClass(spec.enterClassName)
      const id = window.setTimeout(() => setAnimClass(''), spec.durationMs + 32)
      return () => window.clearTimeout(id)
    }
    return undefined
  }, [location.pathname, location.key, navType, preferCrossfade, ffMotionNavigation, reduceMotion])

  return (
    <div
      className={['lx-route-transition', animClass].filter(Boolean).join(' ')}
      data-nav-intent={intent}
      data-motion-nav={ffMotionNavigation ? 'on' : 'off'}
    >
      {children}
    </div>
  )
}
