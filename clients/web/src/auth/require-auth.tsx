import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { getBearerToken } from '../lib/impersonation'

export function RequireAuth() {
  const location = useLocation()
  if (!getBearerToken()) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  return <Outlet />
}
