-- Rollback companion to 432_parent_link_assign_perm_fix.sql
-- See docs/runbooks/database-migration-rollback.md

-- No-op: 431 already owns the four-segment permission; do not delete it here.
SELECT 1;
