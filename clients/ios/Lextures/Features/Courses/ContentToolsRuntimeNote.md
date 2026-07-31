# Content Tools (CT.M3 / CT.M4) — mobile host

Native iOS mounts Content Tools via `ContentToolHostView` inside the shared markdown
renderer. Supported tools (`noop_probe` today) render natively with autosave, conflict
handling, and offline state writes. When `ffMobileContentToolsSandbox` is on, tools
without a native renderer mount in a sandboxed opaque-origin WebView
(`Features/ContentTools/Sandbox/`) using the CT.5 bridge. Otherwise unsupported tools
show a first-class “Open in browser” placeholder. Courses with `contentToolsEnabled` off
hide fences; the client dark-launch flag `ffMobileContentTools` (default off) falls back
to the CT.M1 placeholder until enabled.
