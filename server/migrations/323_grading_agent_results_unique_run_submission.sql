-- De-duplicate any existing (run_id, submission_id) pairs (keep the latest row per pair)
-- before creating the unique index so the index creation does not fail.
DELETE FROM assessment.grading_agent_results
WHERE id IN (
    SELECT id
    FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY run_id, submission_id
                   ORDER BY created_at DESC
               ) AS rn
        FROM assessment.grading_agent_results
        WHERE run_id IS NOT NULL AND is_dry_run = false
    ) ranked
    WHERE rn > 1
);

-- Enforce exactly-once grading per (run, submission) for non-dry-run rows.
-- This makes redelivered queue messages safe: a second INSERT will conflict and
-- the handler can detect "already processed" without re-calling the LLM.
CREATE UNIQUE INDEX grading_agent_results_run_submission_unique
    ON assessment.grading_agent_results (run_id, submission_id)
    WHERE run_id IS NOT NULL AND is_dry_run = false;
