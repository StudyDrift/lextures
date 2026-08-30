import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { getBearerToken } from '../lib/auth'
import { authHandoffHref } from '../lib/post-auth-redirect'

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
    const from = returnPathFromLocation(location)
    return <Navigate to={authHandoffHref('/login', from)} replace state={{ from }} />
  }
  return <Outlet />
}
