# Content Tools (CT.M3) — mobile host

Native iOS mounts Content Tools via `ContentToolHostView` inside the shared markdown
renderer. Supported tools (`noop_probe` today) render natively with autosave, conflict
handling, and offline state writes. Unsupported tools show a first-class
“Open in browser” placeholder. Courses with `contentToolsEnabled` off hide fences;
the client dark-launch flag `ffMobileContentTools` (default off) falls back to the
CT.M1 placeholder until enabled.
