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

const userCourseOrgMatch = `(c.org_id = u.org_id` + consortiumGuestAccessOr + `)`
