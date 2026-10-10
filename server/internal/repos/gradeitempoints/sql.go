// Package gradeitempoints holds the shared SQL fragment for an assignment or quiz's gradable maximum.
package gradeitempoints

// WorthSQL is a SQL expression for a graded item's maximum points. It expects the query to
// LEFT JOIN course.module_assignments AS ma and course.module_quizzes AS mq on the structure item.
//
// Explicit points worth wins; otherwise a non-adaptive quiz falls back to the sum of its question
// points, exactly like the gradebook / My grades (coursestructure.GradebookMaxPoints). The result is
// NULL when no positive maximum is known.
const WorthSQL = `(COALESCE(
    ma.points_worth,
    mq.points_worth,
    CASE WHEN mq.is_adaptive IS TRUE THEN NULL ELSE NULLIF((
        SELECT SUM((elem->>'points')::int)
        FROM jsonb_array_elements(mq.questions_json) AS elem
    ), 0) END
))::int`
