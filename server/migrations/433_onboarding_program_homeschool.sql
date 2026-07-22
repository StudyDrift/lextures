-- HS.5: Widen onboarding_events.program CHECK for Homeschool rebrand.
--
-- Allowed values (keep in sync with server/internal/httpserver/onboarding_http.go):
--   'k-12' | 'higher-ed' | 'self-learner' | 'homeschool' | 'school'
--
-- 'self-learner' is retained for historical rows written before the Homeschool
-- rebrand (HS.5). Do not rewrite past analytics rows.
--
-- Reporting: treat 'self-learner' and 'homeschool' as the same segment when
-- comparing funnels across the cutover date (HS.2 GA4 / analytics notes).
--
-- 'school' was accepted by onboarding_http.go but rejected by the original
-- CHECK from migration 142, so those events were silently dropped; adding it
-- here closes that gap.
--
-- This is a constraint swap on a small analytics table. A plain DROP/ADD is
-- fine below ~1M rows; if the table has grown larger at deploy time, switch to
-- ADD CONSTRAINT … NOT VALID + VALIDATE CONSTRAINT to shorten the exclusive lock.

ALTER TABLE onboarding_events DROP CONSTRAINT IF EXISTS onboarding_events_program_check;

ALTER TABLE onboarding_events ADD CONSTRAINT onboarding_events_program_check
  CHECK (program IN ('k-12', 'higher-ed', 'self-learner', 'homeschool', 'school'));
