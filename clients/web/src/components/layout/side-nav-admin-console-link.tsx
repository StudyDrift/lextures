import { useEffect, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { LayoutDashboard } from 'lucide-react'
import { authorizedFetch } from '../../lib/api'
import { sideNavActiveClass } from './side-nav-styles'
import { SideNavLink } from './side-nav-link'
import { SideNavSectionLabel } from './side-nav-section-label'

function orgPath(base: string, orgId: string | null): string {
  if (!orgId) return base
  const sep = base.includes('?') ? '&' : '?'
  return `${base}${sep}orgId=${encodeURIComponent(orgId)}`
}

export default function SideNavAdminConsoleLink({ orgId }: { orgId: string | null }) {
  const location = useLocation()
  const [canAccess, setCanAccess] = useState(false)

  useEffect(() => {
    let cancelled = false
    void authorizedFetch('/api/v1/me/admin-console-capabilities')
      .then(async (res) => {
        if (!res.ok) return { enabled: false, canAccess: false }
        return (await res.json()) as { enabled: boolean; canAccess: boolean }
      })
      .then((c) => {
        if (!cancelled) setCanAccess(c.enabled && c.canAccess)
      })
      .catch(() => {
        if (!cancelled) setCanAccess(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  if (!canAccess) return null

  const active =
    location.pathname === '/org-admin' ||
    location.pathname.startsWith('/org-admin/users') ||
    location.pathname.startsWith('/org-admin/courses') ||
    location.pathname.startsWith('/org-admin/settings') ||
    location.pathname.startsWith('/org-admin/audit-log') ||
    location.pathname === '/org-admin/integrations'

  return (
    <>
      <SideNavSectionLabel>Admin console</SideNavSectionLabel>
      <SideNavLink
        to={orgPath('/org-admin', orgId)}
        className={() => (active ? sideNavActiveClass : '')}
        icon={<LayoutDashboard className="h-5 w-5" />}
      >
        Admin console
      </SideNavLink>
    </>
  )
}
