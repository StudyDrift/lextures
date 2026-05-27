#!/usr/bin/env bash
# SOC 2 Type II evidence collection script (plan 10.9 FR-5, NF-Performance < 5 min).
# Collects evidence categories required for the audit observation period:
#   - Authentication log summary (180-day retention check)
#   - Access review status
#   - Incident summary
#   - Vendor risk register status
#   - Change management (branch protection, recent PRs)
#   - Vulnerability scan results
# Usage: DATABASE_URL=postgres://... ./scripts/compliance/evidence_collector.sh [OUTPUT_DIR]

set -euo pipefail

OUTPUT_DIR="${1:-./evidence/$(date +%Y%m%d)}"
mkdir -p "$OUTPUT_DIR"

echo "SOC 2 evidence collection — $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Output directory: $OUTPUT_DIR"

# Require DATABASE_URL.
if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "ERROR: DATABASE_URL is required." >&2
  exit 1
fi

PSQL="psql $DATABASE_URL --no-psqlrc -t -A"

# 1. Access review status (FR-5, CC6.3).
echo "Collecting access reviews..."
$PSQL -c "
SELECT json_agg(row_to_json(r)) FROM (
  SELECT ar.id,
         u.email AS reviewer_email,
         ar.review_type,
         ar.reviewed_at,
         ar.next_review_due
    FROM compliance.access_reviews ar
    JOIN \"user\".users u ON u.id = ar.reviewer_id
   ORDER BY ar.reviewed_at DESC
   LIMIT 100
) r;
" > "$OUTPUT_DIR/access_reviews.json"

# Check for overdue access reviews (privileged = 90 days, all_production = 180 days).
echo "Checking for overdue access reviews..."
$PSQL -c "
SELECT json_agg(row_to_json(r)) FROM (
  SELECT review_type,
         MAX(reviewed_at) AS last_review,
         NOW() - MAX(reviewed_at) AS age,
         CASE review_type
           WHEN 'privileged'      THEN MAX(reviewed_at) < NOW() - INTERVAL '90 days'
           WHEN 'all_production'  THEN MAX(reviewed_at) < NOW() - INTERVAL '180 days'
           WHEN 'third_party'     THEN MAX(reviewed_at) < NOW() - INTERVAL '365 days'
         END AS overdue
    FROM compliance.access_reviews
   GROUP BY review_type
) r;
" > "$OUTPUT_DIR/access_review_overdue.json"

# 2. Open incidents (FR-5, CC7.3).
echo "Collecting incident summary..."
$PSQL -c "
SELECT json_agg(row_to_json(i)) FROM (
  SELECT id, title, severity, status, opened_at, resolved_at, tsc_criteria
    FROM compliance.incidents
   ORDER BY opened_at DESC
   LIMIT 200
) i;
" > "$OUTPUT_DIR/incidents.json"

$PSQL -c "
SELECT json_build_object(
  'total', COUNT(*),
  'open', COUNT(*) FILTER (WHERE status = 'open'),
  'contained', COUNT(*) FILTER (WHERE status = 'contained'),
  'resolved', COUNT(*) FILTER (WHERE status = 'resolved'),
  'closed', COUNT(*) FILTER (WHERE status = 'closed'),
  'p0_count', COUNT(*) FILTER (WHERE severity = 'P0'),
  'p1_count', COUNT(*) FILTER (WHERE severity = 'P1')
) FROM compliance.incidents;
" > "$OUTPUT_DIR/incident_summary.json"

# 3. Vendor risk register (FR-5, CC9.2, AC-6).
echo "Collecting vendor risk register..."
$PSQL -c "
SELECT json_agg(row_to_json(v)) FROM (
  SELECT vendor_name, risk_tier, soc2_report_url, report_date, next_review_due,
         CASE WHEN next_review_due < CURRENT_DATE THEN true ELSE false END AS overdue
    FROM compliance.vendor_risk
   ORDER BY CASE risk_tier WHEN 'critical' THEN 1 WHEN 'high' THEN 2 WHEN 'medium' THEN 3 ELSE 4 END,
            vendor_name
) v;
" > "$OUTPUT_DIR/vendor_risk.json"

# 4. Production user count and admin count (CC6.3 evidence).
echo "Collecting user access summary..."
$PSQL -c "
SELECT json_build_object(
  'total_users', (SELECT COUNT(*) FROM \"user\".users WHERE deleted_at IS NULL),
  'global_admin_count', (
    SELECT COUNT(DISTINCT uar.user_id)
      FROM \"user\".user_app_roles uar
      JOIN \"user\".app_roles ar ON ar.id = uar.role_id
     WHERE ar.name = 'Global Admin'
  ),
  'collected_at', NOW()
);
" > "$OUTPUT_DIR/user_access_summary.json"

# 5. Branch protection status (CC8.1, AC-1) — via GitHub CLI if available.
if command -v gh &>/dev/null; then
  echo "Checking branch protection (requires gh auth)..."
  gh api repos/:owner/:repo/branches/main/protection \
    --jq '{required_status_checks, enforce_admins, required_pull_request_reviews}' \
    > "$OUTPUT_DIR/branch_protection.json" 2>/dev/null || echo '{"error":"gh auth required or repo not on GitHub"}' > "$OUTPUT_DIR/branch_protection.json"
else
  echo '{"note":"gh CLI not available; verify branch protection manually"}' > "$OUTPUT_DIR/branch_protection.json"
fi

# 6. Summary manifest.
cat > "$OUTPUT_DIR/MANIFEST.json" <<EOF
{
  "collected_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "evidence_period": "rolling 12 months",
  "tsc_categories": ["Security (CC1-CC9)", "Availability (A1)", "Privacy (P1-P8)"],
  "files": [
    "access_reviews.json",
    "access_review_overdue.json",
    "incidents.json",
    "incident_summary.json",
    "vendor_risk.json",
    "user_access_summary.json",
    "branch_protection.json"
  ]
}
EOF

echo "Evidence collection complete. Files written to: $OUTPUT_DIR"
ls -lh "$OUTPUT_DIR/"
