# Content Tools (CT.M3 / CT.M4) — mobile host

Native iOS mounts Content Tools via `ContentToolHostView` inside the shared markdown
renderer. Pack tools (including `inline_questions`, flashcards, media checkpoints, etc.)
render natively with autosave, conflict handling, and offline state writes. When
`ffMobileContentToolsSandbox` is on, tools without a native renderer mount in a sandboxed
opaque-origin WebView (`Features/ContentTools/Sandbox/`) using the CT.5 bridge. Otherwise
unsupported tools show a first-class “Open in browser” placeholder. Courses with
`contentToolsEnabled` off hide fences; `ffMobileContentTools` (default on) can be flipped
off to fall back to the CT.M1 “Interactive tool: …” placeholder.
