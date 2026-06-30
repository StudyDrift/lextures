import { NavLink, Navigate, Outlet, useSearchParams } from 'react-router-dom'
import {
  Activity,
  BookOpen,
  FileUp,
  LayoutDashboard,
  Megaphone,
  Plug,
  ScrollText,
  Settings,
  Users,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { AdminSearchBar } from '../../components/admin/AdminSearchBar'
import { fetchAdminConsoleCapabilities } from '../../lib/admin-console-api'
import { usePlatformFeatures } from '../../context/platform-features-context'

const bannersNavItem = { to: '/org-admin/banners', label: 'Notices', icon: Megaphone, end: false as const }

const baseNavItems = [
  { to: '/org-admin', label: 'Overview', icon: LayoutDashboard, end: true },
  { to: '/org-admin/users', label: 'Users', icon: Users },
  { to: '/org-admin/courses', label: 'Courses', icon: BookOpen },
  { to: '/org-admin/integrations', label: 'Integrations', icon: Plug },
  { to: '/org-admin/settings', label: 'Settings', icon: Settings },
  { to: '/org-admin/audit-log', label: 'Audit log', icon: ScrollText },
]

const importNavItem = { to: '/org-admin/import', label: 'Import', icon: FileUp, end: false as const }

export default function AdminLayout() {
  const { adminConsoleEnabled, bulkCsvImportEnabled, maintenanceBannerEnabled } = usePlatformFeatures()
  const [searchParams] = useSearchParams()
  const orgId = searchParams.get('orgId')
  const [canAccess, setCanAccess] = useState<boolean | null>(null)

  useEffect(() => {
    let cancelled = false
    void fetchAdminConsoleCapabilities()
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

  if (!adminConsoleEnabled) {
    return <Navigate to="/" replace />
  }

  if (canAccess === false) {
    return (
      <div className="mx-auto max-w-lg p-8 text-center">
        <Activity className="mx-auto mb-3 h-10 w-10 text-slate-400" aria-hidden />
        <h1 className="text-lg font-semibold text-slate-900 dark:text-slate-100">Access denied</h1>
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
          You need org admin or global admin permissions to use the admin console.
        </p>
      </div>
    )
  }

  function linkTo(path: string) {
    if (!orgId) return path
    return `${path}?orgId=${encodeURIComponent(orgId)}`
  }

  const navItems = (() => {
    let items = [...baseNavItems]
    if (bulkCsvImportEnabled) {
      items = [...items.slice(0, 2), importNavItem, ...items.slice(2)]
    }
    if (maintenanceBannerEnabled) {
      const integrationsIdx = items.findIndex((i) => i.to === '/org-admin/integrations')
      items = [
        ...items.slice(0, integrationsIdx + 1),
        bannersNavItem,
        ...items.slice(integrationsIdx + 1),
      ]
    }
    return items
  })()

  return (
    <div className="flex min-h-0 flex-1 flex-col md:flex-row">
      <nav
        aria-label="Admin console"
        className="border-b border-slate-200 bg-slate-50 px-3 py-2 md:w-56 md:shrink-0 md:border-b-0 md:border-r md:py-4 dark:border-neutral-800 dark:bg-neutral-950"
      >
        <p className="mb-2 hidden px-2 text-xs font-semibold uppercase tracking-wide text-slate-500 md:block">
          Admin console
        </p>
        <ul className="flex gap-1 overflow-x-auto md:flex-col md:overflow-visible">
          {navItems.map(({ to, label, icon: Icon, end }) => (
            <li key={to} className="shrink-0">
              <NavLink
                to={linkTo(to)}
                end={end}
                className={({ isActive }) =>
                  `flex items-center gap-2 rounded-lg px-3 py-2 text-sm font-medium whitespace-nowrap ${
                    isActive
                      ? 'bg-indigo-100 text-indigo-900 dark:bg-indigo-950 dark:text-indigo-100'
                      : 'text-slate-700 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-neutral-900'
                  }`
                }
              >
                <Icon className="h-4 w-4 shrink-0" aria-hidden />
                {label}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>
      <div className="min-h-0 min-w-0 flex-1 overflow-auto p-4 md:p-6">
        {canAccess === null ? (
          <p className="text-sm text-slate-500">Loading admin console…</p>
        ) : (
          <>
            <AdminSearchBar />
            <Outlet />
          </>
        )}
      </div>
    </div>
  )
}
