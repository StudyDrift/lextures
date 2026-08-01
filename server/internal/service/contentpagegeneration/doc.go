// Package contentpagegeneration drafts module content-page sections from an instructor prompt via the AI provider.
//
// When IncludeTools is set, drafts may also include interactive content-tool configs
// (inline questions, flashcards, etc.) as structured tools on sections — not lex-tool fences.
// The client materializes tool instances and embeds fences on apply.
package contentpagegeneration
