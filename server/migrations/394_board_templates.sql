-- VC.8 — Board templates, duplication, and create-from-template.

CREATE TABLE IF NOT EXISTS board.board_templates (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope         TEXT NOT NULL,
    course_id     UUID REFERENCES course.courses (id) ON DELETE CASCADE,
    org_id        UUID REFERENCES tenant.organizations (id) ON DELETE CASCADE,
    title         TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    tags          TEXT[] NOT NULL DEFAULT '{}',
    definition    JSONB NOT NULL,
    created_by    UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_templates_scope_check
        CHECK (scope IN ('builtin', 'course', 'org')),
    CONSTRAINT board_templates_scope_refs_check
        CHECK (
            (scope = 'builtin' AND course_id IS NULL AND org_id IS NULL)
            OR (scope = 'course' AND course_id IS NOT NULL)
            OR (scope = 'org' AND org_id IS NOT NULL)
        )
);

CREATE INDEX IF NOT EXISTS idx_board_templates_scope
    ON board.board_templates (scope, org_id, course_id);

CREATE INDEX IF NOT EXISTS idx_board_templates_tags
    ON board.board_templates USING GIN (tags);

COMMENT ON TABLE board.board_templates IS
    'VC.8: Built-in, course-local, and org-shared board templates (layout + sections + seed posts + settings).';

CREATE TABLE IF NOT EXISTS board.board_copy_jobs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_course_id UUID NOT NULL REFERENCES course.courses (id) ON DELETE CASCADE,
    source_board_id  UUID NOT NULL REFERENCES board.boards (id) ON DELETE CASCADE,
    created_by       UUID REFERENCES "user".users (id) ON DELETE SET NULL,
    mode             TEXT NOT NULL,
    title            TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'pending',
    progress         INT NOT NULL DEFAULT 0,
    result_board_id  UUID REFERENCES board.boards (id) ON DELETE SET NULL,
    error            TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_copy_jobs_mode_check
        CHECK (mode IN ('structure', 'full')),
    CONSTRAINT board_copy_jobs_status_check
        CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS idx_board_copy_jobs_target
    ON board.board_copy_jobs (target_course_id, created_at DESC);

COMMENT ON TABLE board.board_copy_jobs IS
    'VC.8: Progress tracking for large full board copies (202 + job id).';

-- Built-in templates (stable IDs). Seed posts are English defaults; locale overlays
-- are applied at instantiate time from the code registry (FR-9 / AC-7).
INSERT INTO board.board_templates (id, scope, title, description, tags, definition)
VALUES
(
  'a1000000-0000-4000-8000-000000000001',
  'builtin',
  'Brainstorm wall',
  'Open wall for rapid idea generation.',
  ARRAY['brainstorm', 'ideas', 'wall'],
  '{
    "layout": "wall",
    "settings": {},
    "reactionMode": "like",
    "attribution": "named",
    "moderationMode": "open",
    "canPost": true,
    "canInteract": true,
    "canArrange": true,
    "sections": [],
    "seedPosts": [
      {
        "key": "prompt",
        "contentType": "text",
        "title": "Prompt",
        "body": {"text": "What ideas do you have? Add a card for each thought."},
        "sortIndex": 0
      }
    ]
  }'::jsonb
),
(
  'a1000000-0000-4000-8000-000000000002',
  'builtin',
  'Exit ticket',
  'Quick end-of-class reflection with a single prompt.',
  ARRAY['exit-ticket', 'formative', 'reflection'],
  '{
    "layout": "stream",
    "settings": {},
    "reactionMode": "none",
    "attribution": "named",
    "moderationMode": "open",
    "canPost": true,
    "canInteract": false,
    "canArrange": false,
    "sections": [],
    "seedPosts": [
      {
        "key": "prompt",
        "contentType": "text",
        "title": "Exit ticket",
        "body": {"text": "What is one thing you learned today, and one question you still have?"},
        "sortIndex": 0
      }
    ]
  }'::jsonb
),
(
  'a1000000-0000-4000-8000-000000000003',
  'builtin',
  'KWL chart',
  'Know / Want to know / Learned columns.',
  ARRAY['kwl', 'columns', 'prior-knowledge'],
  '{
    "layout": "columns",
    "settings": {},
    "reactionMode": "none",
    "attribution": "named",
    "moderationMode": "open",
    "canPost": true,
    "canInteract": true,
    "canArrange": true,
    "sections": [
      {"key": "know", "title": "Know", "sortIndex": 0},
      {"key": "want", "title": "Want to know", "sortIndex": 1},
      {"key": "learned", "title": "Learned", "sortIndex": 2}
    ],
    "seedPosts": [
      {
        "key": "prompt-k",
        "contentType": "text",
        "title": "What do you already know?",
        "body": {"text": "Add cards under Know for prior knowledge."},
        "sectionKey": "know",
        "sortIndex": 0
      }
    ]
  }'::jsonb
),
(
  'a1000000-0000-4000-8000-000000000004',
  'builtin',
  'Discussion',
  'Threaded-style prompt wall for class discussion.',
  ARRAY['discussion', 'prompt', 'conversation'],
  '{
    "layout": "stream",
    "settings": {},
    "reactionMode": "like",
    "attribution": "named",
    "moderationMode": "open",
    "canPost": true,
    "canInteract": true,
    "canArrange": false,
    "sections": [],
    "seedPosts": [
      {
        "key": "prompt",
        "contentType": "text",
        "title": "Discussion prompt",
        "body": {"text": "Share your response, then react to two classmates."},
        "sortIndex": 0
      }
    ]
  }'::jsonb
),
(
  'a1000000-0000-4000-8000-000000000005',
  'builtin',
  'Gallery',
  'Grid for showcasing images and media.',
  ARRAY['gallery', 'images', 'showcase'],
  '{
    "layout": "grid",
    "settings": {},
    "reactionMode": "star",
    "attribution": "named",
    "moderationMode": "open",
    "canPost": true,
    "canInteract": true,
    "canArrange": true,
    "sections": [],
    "seedPosts": [
      {
        "key": "prompt",
        "contentType": "text",
        "title": "Gallery brief",
        "body": {"text": "Upload an image or file that represents your work. Add a short caption."},
        "sortIndex": 0
      }
    ]
  }'::jsonb
),
(
  'a1000000-0000-4000-8000-000000000006',
  'builtin',
  'Timeline',
  'Chronological board for events and milestones.',
  ARRAY['timeline', 'history', 'sequence'],
  '{
    "layout": "timeline",
    "settings": {},
    "reactionMode": "none",
    "attribution": "named",
    "moderationMode": "open",
    "canPost": true,
    "canInteract": true,
    "canArrange": true,
    "sections": [],
    "seedPosts": [
      {
        "key": "prompt",
        "contentType": "text",
        "title": "Build the timeline",
        "body": {"text": "Add cards for key events and set each card date."},
        "sortIndex": 0
      }
    ]
  }'::jsonb
),
(
  'a1000000-0000-4000-8000-000000000007',
  'builtin',
  'Map',
  'Geographic board for place-based activities.',
  ARRAY['map', 'geography', 'places'],
  '{
    "layout": "map",
    "settings": {"mapCenter": {"lat": 39.8283, "lng": -98.5795}, "mapZoom": 4},
    "reactionMode": "none",
    "attribution": "named",
    "moderationMode": "open",
    "canPost": true,
    "canInteract": true,
    "canArrange": true,
    "sections": [],
    "seedPosts": [
      {
        "key": "prompt",
        "contentType": "text",
        "title": "Pin a place",
        "body": {"text": "Add a card and set coordinates for a location that matters to this topic."},
        "sortIndex": 0
      }
    ]
  }'::jsonb
),
(
  'a1000000-0000-4000-8000-000000000008',
  'builtin',
  'Q&A',
  'Question wall with upvote reactions.',
  ARRAY['qa', 'questions', 'vote'],
  '{
    "layout": "wall",
    "settings": {},
    "reactionMode": "vote",
    "attribution": "anon_to_peers",
    "moderationMode": "open",
    "canPost": true,
    "canInteract": true,
    "canArrange": false,
    "sections": [],
    "seedPosts": [
      {
        "key": "prompt",
        "contentType": "text",
        "title": "Ask a question",
        "body": {"text": "Post a question. Upvote the questions you want answered first."},
        "sortIndex": 0
      }
    ]
  }'::jsonb
)
ON CONFLICT (id) DO NOTHING;
