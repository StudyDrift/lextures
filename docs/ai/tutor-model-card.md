# AI Tutor Model Card

> Plan 19.1 — Persistent AI Tutor Across Sessions

## Model

- **Default provider:** OpenRouter
- **Default model:** Configurable per org via AI governance (plan 19.10); typically Claude 3.5 Sonnet class models
- **Routing:** Same infrastructure as other Lextures AI features (`server/internal/service/openrouter/`)

## Purpose

Course-grounded conversational tutoring for enrolled students. The tutor retrieves relevant course content chunks via lexical RAG (`server/internal/service/notebookrag/`) and cites sources in structured `citations[]` fields.

## Training data

The tutor uses pre-trained general-purpose LLMs. Lextures does **not** fine-tune models on student data. Prompt context includes:

- Course title and retrieved content excerpts
- Up to 10 recent messages from the current tutor session
- Optional concept tags when student messages mention course concepts

## Limitations

- May hallucinate when course materials do not cover the question
- Citations are validated against retrieved chunks but semantic accuracy is not guaranteed
- Not intended for graded work completion; system prompt instructs scaffolding over direct answers
- English-first; multilingual behavior depends on the selected model

## Privacy & safety

- User messages pass through regex PII redaction before leaving the system (plan 19.11 baseline)
- Students may opt out via `ai_tutor_opt_out`; tutor endpoints return HTTP 403 when set
- Session transcripts are education records with org-configurable retention (default 365 days)
- Instructors see aggregate concept-confusion summaries only, not individual transcripts

## Bias notes

Responses inherit biases present in the underlying LLM and course-authored materials. Instructors should review aggregate confusion digests and supplement with human instruction.

## Contact

For AI governance questions, see Settings → Global platform and the org AI disclosure controls (plan 10.17).
