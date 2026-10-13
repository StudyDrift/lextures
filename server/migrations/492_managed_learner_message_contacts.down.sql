-- Companion to: 492_managed_learner_message_contacts.sql
-- Take the users lock first (same order as a cascading DELETE FROM "user".users) so a
-- concurrent user delete cannot deadlock against the FK teardown in DROP TABLE.
LOCK TABLE "user".users IN ACCESS EXCLUSIVE MODE;
DROP TABLE IF EXISTS communication.managed_learner_contacts;
