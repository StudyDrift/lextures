-- Refresh the built-in AI Grader template prompt for courses seeded with the legacy text.

UPDATE assessment.grading_agent_templates
SET
    prompt = 'You are a TA who is tasked with grading student submissions. You are kind, attentive to detail, provide helpful feedback. Use the following information to help you grade:

# Student Submissions
```
$StudentSubmission.Submissions
```

# Activity Content
```
$Activity.Content
```

# Activity Rubric
```
$Activity.Rubric
```',
    workflow_graph = '{
        "version": 1,
        "nodes": [
            {"id": "output", "type": "output", "position": {"x": 0, "y": 0}, "data": {}},
            {"id": "sub", "type": "studentSubmission", "position": {"x": -640, "y": -40}, "data": {}},
            {"id": "act", "type": "activity", "position": {"x": -640, "y": 80}, "data": {}},
            {
                "id": "ai",
                "type": "ai",
                "position": {"x": -320, "y": 0},
                "data": {
                    "prompt": "You are a TA who is tasked with grading student submissions. You are kind, attentive to detail, provide helpful feedback. Use the following information to help you grade:\n\n# Student Submissions\n```\n$StudentSubmission.Submissions\n```\n\n# Activity Content\n```\n$Activity.Content\n```\n\n# Activity Rubric\n```\n$Activity.Rubric\n```"
                }
            }
        ],
        "edges": [
            {"id": "e-sub-ai", "source": "sub", "sourceHandle": "submission", "target": "ai", "targetHandle": "input"},
            {"id": "e-act-content-ai", "source": "act", "sourceHandle": "content", "target": "ai", "targetHandle": "input"},
            {"id": "e-act-rubric-ai", "source": "act", "sourceHandle": "rubric", "target": "ai", "targetHandle": "input"},
            {"id": "e-ai-output", "source": "ai", "sourceHandle": "output", "target": "output", "targetHandle": "grade"}
        ]
    }'::jsonb,
    updated_at = NOW()
WHERE name = 'AI Grader'
  AND prompt = 'Grade the student submission using the assignment instructions and rubric. Assign an appropriate score and provide constructive feedback.';