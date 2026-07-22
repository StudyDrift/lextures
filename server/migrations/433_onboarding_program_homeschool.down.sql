-- Rollback companion to 433_onboarding_program_homeschool.sql
-- See docs/runbooks/database-migration-rollback.md
--
-- IMPORTANT: This down migration restores the pre-HS.5 CHECK constraint
-- ('k-12' | 'higher-ed' | 'self-learner'). It will FAIL LOUDLY if any rows
-- still use 'homeschool' or 'school'.
--
-- Before running this down migration, the operator MUST purge or remap those
-- rows, for example:
--
--   -- Remap new-value rows to the historical self-learner segment (loses school distinction):
--   UPDATE onboarding_events SET program = 'self-learner'
--     WHERE program IN ('homeschool', 'school');
--
--   -- Or delete them if they must not appear in the restored schema:
--   DELETE FROM onboarding_events WHERE program IN ('homeschool', 'school');
--
-- Do not wrap the constraint add in a way that silently drops data.

ALTER TABLE onboarding_events DROP CONSTRAINT IF EXISTS onboarding_events_program_check;

ALTER TABLE onboarding_events ADD CONSTRAINT onboarding_events_program_check
  CHECK (program IN ('k-12', 'higher-ed', 'self-learner'));
