/* eslint-disable react-refresh/only-export-components -- provider + hook live together */
import { type ReactNode, useEffect } from 'react'
import { useLocation } from 'react-router-dom'
import { getAccessToken } from '../lib/auth'
import { emitContentFilterActivity } from '../lib/content-filter-api'
import { decodeJwtPayload } from '../lib/jwt-payload'
import {
  PERM_COURSE_CREATE,
  PERM_PARENT_DASHBOARD,
  PERM_RBAC_MANAGE,
  PERM_TENANT_ORG_ROLES_MANAGE,
} from '../lib/rbac-api'
import { usePlatformFeatures } from './platform-features-context'
import { usePermissions } from './use-permissions'

function setMeta(name: string, content: string) {
  let el = document.querySelector<HTMLMetaElement>(`meta[name="${name}"][data-lextures]`)
  if (!el) {
    el = document.createElement('meta')
    el.setAttribute('name', name)
    el.setAttribute('data-lextures', 'true')
    document.head.appendChild(el)
  }
  el.setAttribute('content', content)
}

function removeMeta(name: string) {
  document.querySelector<HTMLMetaElement>(`meta[name="${name}"][data-lextures]`)?.remove()
}

function deriveContentFilterRole(allows: (perm: string) => boolean): string {
  if (allows(PERM_PARENT_DASHBOARD)) return 'parent'
  if (allows(PERM_RBAC_MANAGE) || allows(PERM_TENANT_ORG_ROLES_MANAGE)) return 'admin'
  if (allows(PERM_COURSE_CREATE)) return 'teacher'
  return 'student'
}

export function ContentFilterProvider({ children }: { children: ReactNode }) {
  const { ffContentFilterIntegration, loading: featuresLoading } = usePlatformFeatures()
  const { allows, loading: permsLoading } = usePermissions()
  const location = useLocation()

  useEffect(() => {
    if (featuresLoading || !ffContentFilterIntegration) {
      removeMeta('lextures:user-role')
      removeMeta('lextures:org-id')
      return
    }
    if (permsLoading) return

    const payload = decodeJwtPayload(getAccessToken())
    const orgId = payload?.org_id ?? ''
    if (orgId) {
      setMeta('lextures:org-id', orgId)
    } else {
      removeMeta('lextures:org-id')
    }
    setMeta('lextures:user-role', deriveContentFilterRole(allows))
  }, [featuresLoading, ffContentFilterIntegration, permsLoading, allows])

  useEffect(() => {
    if (featuresLoading || !ffContentFilterIntegration) return
    const url = window.location.href
    const title = document.title || 'Lextures'
    void emitContentFilterActivity(url, title)
  }, [featuresLoading, ffContentFilterIntegration, location.pathname, location.search])

  return children
}
