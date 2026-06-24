-- Flag-for-Review sink: review-queue status and metadata on grading agent results.

ALTER TYPE assessment.grading_agent_item_status ADD VALUE IF NOT EXISTS 'flagged';

ALTER TABLE assessment.grading_agent_results
    ADD COLUMN IF NOT EXISTS flag_reason TEXT,
    ADD COLUMN IF NOT EXISTS flag_priority TEXT;