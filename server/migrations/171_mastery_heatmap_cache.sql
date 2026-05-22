-- 9.3 Mastery Heatmap: analytics schema and materialized view cache.
-- Depends on: 087_learner_model (course.learner_concept_states, course.concepts),
--             and course.course_enrollments.

CREATE SCHEMA IF NOT EXISTS analytics;

-- Materialized cache: one row per (enrollment, concept) pair with mastery state.
-- Refresh with: REFRESH MATERIALIZED VIEW CONCURRENTLY analytics.mastery_heatmap
CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.mastery_heatmap AS
SELECT
    ce.id            AS enrollment_id,
    ce.user_id,
    ce.course_id,
    lcs.concept_id,
    (lcs.mastery)::float8   AS mastery_score,
    lcs.updated_at          AS state_updated_at
FROM course.learner_concept_states lcs
JOIN course.concepts c
    ON c.id = lcs.concept_id
JOIN course.course_enrollments ce
    ON ce.user_id = lcs.user_id
    AND ce.course_id = c.course_id
WHERE ce.active = true;

CREATE UNIQUE INDEX IF NOT EXISTS mastery_heatmap_enrollment_concept_idx
    ON analytics.mastery_heatmap (enrollment_id, concept_id);
CREATE INDEX IF NOT EXISTS mastery_heatmap_course_concept_score_idx
    ON analytics.mastery_heatmap (course_id, concept_id, mastery_score);
CREATE INDEX IF NOT EXISTS mastery_heatmap_course_user_idx
    ON analytics.mastery_heatmap (course_id, user_id);
