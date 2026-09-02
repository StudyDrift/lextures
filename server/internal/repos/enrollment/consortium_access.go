package enrollment

// consortiumGuestAccessOr extends native org enrollment checks for cross-institutional
// guest students (plan 14.18). Append after c.org_id = u.org_id within an OR group.
const consortiumGuestAccessOr = `
OR (
  ce.home_org_id IS NOT NULL
  AND ce.home_org_id = u.org_id
  AND EXISTS (
    SELECT 1 FROM tenant.consortium_agreements ca
    WHERE ca.host_org_id = c.org_id
      AND ca.guest_org_id = ce.home_org_id
      AND ca.status = 'active'
      AND (ca.expires_at IS NULL OR ca.expires_at > NOW())
  )
)
`

// globalAdminAccessOr lets Global Admins (global:app:rbac:manage) use an enrollment
// even when the course lives in another organization. Regular users stay org-scoped
// (see TestEnrollment_OrgIsolation_Pg).
const globalAdminAccessOr = `
OR EXISTS (
  SELECT 1
  FROM "user".user_app_roles uar
  INNER JOIN "user".rbac_role_permissions rp ON rp.role_id = uar.role_id
  INNER JOIN "user".permissions p ON p.id = rp.permission_id
  WHERE uar.user_id = u.id
    AND p.permission_string = 'global:app:rbac:manage'
)
`

const userCourseOrgMatch = `(c.org_id = u.org_id` + consortiumGuestAccessOr + globalAdminAccessOr + `)`
