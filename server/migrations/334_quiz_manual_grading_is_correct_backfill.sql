-- Canvas imports may have set is_correct=false on manual question types before instructor review.
-- Ungraded manual items should keep is_correct NULL until scored in Lextures.
UPDATE course.quiz_responses qr
SET is_correct = NULL
WHERE qr.question_type IN (
  'essay', 'short_answer', 'fill_in_blank', 'file_upload',
  'audio_response', 'video_response', 'code', 'hotspot', 'formula'
)
  AND qr.is_correct IS NOT NULL
  AND COALESCE(qr.points_awarded, 0) = 0
  AND qr.response_json::text NOT IN ('{}', 'null');