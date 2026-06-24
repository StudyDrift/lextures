-- Human Review Gate: held-grade metadata on grading agent results.

ALTER TABLE assessment.grading_agent_results
    ADD COLUMN IF NOT EXISTS held_reason TEXT,
    ADD COLUMN IF NOT EXISTS held_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS held_queue TEXT;

CREATE INDEX IF NOT EXISTS idx_grading_agent_results_held
    ON assessment.grading_agent_results (config_id, status, held_at DESC)
    WHERE status = 'suggested' AND held_at IS NOT NULL;