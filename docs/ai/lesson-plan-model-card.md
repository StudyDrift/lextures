# Lesson plan generation — model card

| Field | Value |
|---|---|
| **Feature** | AI Lesson Generator (19.2) |
| **Default model** | OpenRouter course-setup model (org/instructor configurable) |
| **Purpose** | Generate lesson plans, differentiated activities, formative quizzes, and rubrics from instructor-supplied learning objectives |
| **Input data** | Learning objective, grade level, subject, optional duration, standards code, differentiation levels |
| **Output** | Markdown lesson plans and activities; JSON quiz questions; JSON rubric definitions |
| **PII handling** | Instructor objective text passes through regex PII redaction before LLM calls (shared with AI tutor) |
| **Human review** | All outputs are editable; save-to-course creates draft (unpublished) modules only |
| **Provenance** | `generated_by: lextures-ai`, `model_id`, `generation_ts` on each component |
| **Failure mode** | Partial results returned when individual components fail; per-component regenerate supported |
| **Estimated tokens** | ~8 000 per full package (parallel sub-calls) |

## Sub-prompts (system_prompts keys)

- `lesson_generation_plan` — lesson outline
- `lesson_generation_activity` — differentiated activities
- `quiz_generation` — formative assessment (shared with quiz authoring)
- `lesson_generation_rubric` — open-ended task rubric

## Risks

- Generated content may be factually incorrect — instructors must review before publishing.
- Standards alignment is indicative when concept-graph mappings are unavailable.
