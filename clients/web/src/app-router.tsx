import type { ReactNode } from 'react'
import { BrowserRouter } from 'react-router-dom'

/**
 * Location updates are synchronous on purpose.
 *
 * React Router otherwise wraps them in `startTransition`. The address bar moves
 * immediately, but the previous page stays painted if that transition suspends
 * (lazy routes) or is interrupted (`useSyncExternalStore` on course view / forms).
 * Menu links and activity next/previous then look dead: the URL changes, the
 * screen does not.
 */
export function AppRouter({ children }: { children: ReactNode }) {
  return <BrowserRouter useTransitions={false}>{children}</BrowserRouter>
}
