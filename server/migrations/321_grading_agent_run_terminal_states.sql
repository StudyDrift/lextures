-- GA-B1 — Add last_progress_at for stuck-run reconciler.
ALTER TABLE assessment.grading_agent_runs
    ADD COLUMN IF NOT EXISTS last_progress_at TIMESTAMPTZ;

-- Backfill: seed from created_at so the reconciler has a baseline for existing rows.
UPDATE assessment.grading_agent_runs
SET last_progress_at = created_at
WHERE last_progress_at IS NULL;
