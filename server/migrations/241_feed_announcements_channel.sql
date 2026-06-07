-- Every course feed gets a default #announcements channel directly under #general.

INSERT INTO course.feed_channels (course_id, name, sort_order, created_by_user_id)
SELECT c.id, 'announcements', 1, NULL
FROM course.courses c
WHERE NOT EXISTS (
    SELECT 1 FROM course.feed_channels fc
    WHERE fc.course_id = c.id
      AND fc.group_id IS NULL
      AND lower(fc.name) = 'announcements'
);

UPDATE course.feed_channels fc
SET sort_order = fc.sort_order + 1
WHERE fc.group_id IS NULL
  AND lower(fc.name) NOT IN ('general', 'announcements')
  AND fc.sort_order >= 1
  AND EXISTS (
    SELECT 1 FROM course.feed_channels ann
    WHERE ann.course_id = fc.course_id
      AND ann.group_id IS NULL
      AND lower(ann.name) = 'announcements'
  );
