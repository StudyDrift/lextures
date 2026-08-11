import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { getBearerToken } from '../lib/auth'

/** Full return path including query (e.g. ?coupon=) so share links survive sign-in (MKTC.5 FR-11). */
function returnPathFromLocation(location: {
  pathname: string
  search?: string
  hash?: string
}): string {
  return `${location.pathname}${location.search ?? ''}${location.hash ?? ''}`
}

export function RequireAuth() {
  const location = useLocation()
  if (!getBearerToken()) {
    return (
      <Navigate to="/login" replace state={{ from: returnPathFromLocation(location) }} />
    )
  }
  return <Outlet />
}
