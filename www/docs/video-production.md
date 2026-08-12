# Video production and publishing

Videos supplement the page's written instructions; they never replace them. Product walkthroughs target 90–180 seconds, concept explainers 3–6 minutes, and research summaries 2–4 minutes.

Before publishing:

1. Remove real learner names, work, and other personal data.
2. Human-review captions and the full transcript.
3. Add a descriptive title, a source-page link, and chapter markers to YouTube.
4. Commit metadata and transcript cues to `src/lib/videos.ts`.
5. Render `VideoEmbed` on the supporting page and add its `VideoObject` to the page graph.
6. Confirm the page makes no YouTube request before activation and that the transcript remains in the server-rendered DOM.

Embeds use `youtube-nocookie.com`; the facade is a static poster and ordinary link targeting the empty player frame, so third-party content loads only after activation.
