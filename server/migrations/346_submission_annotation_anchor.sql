-- Allow text-anchor annotations (highlight + comment on reflowable previews: DOCX, PPTX,
-- XLSX, Markdown, plain text, and code). Anchors store character offsets + quoted text in
-- coords_json instead of normalized page geometry, so they survive viewport reflow.

ALTER TABLE course.submission_annotations
    DROP CONSTRAINT IF EXISTS submission_annotations_tool_type_check;

ALTER TABLE course.submission_annotations
    ADD CONSTRAINT submission_annotations_tool_type_check
    CHECK (tool_type IN ('highlight', 'draw', 'text', 'pin', 'anchor'));

COMMENT ON TABLE course.submission_annotations IS
    'Instructor/TA markup on a student submission. Geometric tools (highlight/draw/text/pin) '
    'store normalized page overlay geometry in coords_json; the anchor tool stores '
    '{start,end,quote,prefix,suffix} text-character offsets for reflowable documents.';
